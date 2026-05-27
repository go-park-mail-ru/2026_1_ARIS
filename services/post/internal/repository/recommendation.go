package repository

import (
	"context"
	"math"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type RecommendationRepo interface {
	GetForYouCandidates(ctx context.Context, in ForYouInput) ([]int64, error)
	GetTrendingCandidates(ctx context.Context, limit int, excludeAuthorID int64) ([]int64, error)
	GetAuthorAffinity(ctx context.Context, profileID int64, days int) (map[int64]float64, error)
	GetCommunityAffinity(ctx context.Context, profileID int64, days int) (map[int64]float64, error)
	GetPostStats(ctx context.Context, postIDs []int64) (map[int64]PostStats, error)
}

type ForYouInput struct {
	ProfileID       int64
	FriendIDs       []int64
	Limit           int
	ExcludeAuthorID int64 // исключить собственные посты пользователя
}

type PostStats struct {
	Views, Likes, Comments, Reposts int64
}

type clickhouseRecommendation struct {
	conn driver.Conn
}

func NewClickhouseRecommendationRepo(conn driver.Conn) RecommendationRepo {
	return &clickhouseRecommendation{conn: conn}
}

const coldStartThreshold = 20

func (r *clickhouseRecommendation) GetForYouCandidates(ctx context.Context, in ForYouInput) ([]int64, error) {
	seen := make(map[int64]struct{})
	var result []int64

	add := func(ids []int64) {
		for _, id := range ids {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				result = append(result, id)
			}
		}
	}

	// 1. Свежие посты от друзей (last 7 days)
	if len(in.FriendIDs) > 0 {
		if posts, err := r.friendCandidates(ctx, in); err == nil {
			add(posts)
		}
	}

	// 2. Посты авторов с высоким affinity (top-50 за 30 дней)
	if posts, err := r.affinityCandidates(ctx, in); err == nil {
		add(posts)
	}

	// 3. Trending (top-100 за 3 дня)
	if posts, err := r.GetTrendingCandidates(ctx, 100, in.ExcludeAuthorID); err == nil {
		add(posts)
	}

	// 4. Cold-start: свежие активные посты, если кандидатов мало
	if len(result) < coldStartThreshold {
		if posts, err := r.coldStartCandidates(ctx, in); err == nil {
			add(posts)
		}
	}

	return result, nil
}

func (r *clickhouseRecommendation) coldStartCandidates(ctx context.Context, in ForYouInput) ([]int64, error) {
	rows, err := r.conn.Query(ctx, `
		SELECT s.post_id
		FROM post_snapshot AS s FINAL
		LEFT ANTI JOIN (
			SELECT post_id FROM user_recent_viewed FINAL WHERE profile_id = ?
		) AS v ON s.post_id = v.post_id
		WHERE s.is_active = true
		  AND s.author_profile_id != ?
		ORDER BY s.created_at DESC
		LIMIT ?
	`, in.ProfileID, in.ExcludeAuthorID, in.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectInt64Rows(rows)
}

func (r *clickhouseRecommendation) friendCandidates(ctx context.Context, in ForYouInput) ([]int64, error) {
	rows, err := r.conn.Query(ctx, `
		SELECT DISTINCT s.post_id
		FROM post_snapshot AS s FINAL
		LEFT ANTI JOIN (
			SELECT post_id FROM user_recent_viewed FINAL WHERE profile_id = ?
		) AS v ON s.post_id = v.post_id
		WHERE s.author_profile_id IN (?)
		  AND s.author_profile_id != ?
		  AND s.is_active = true
		  AND s.created_at >= now() - INTERVAL 7 DAY
		LIMIT 200
	`, in.ProfileID, in.FriendIDs, in.ExcludeAuthorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectInt64Rows(rows)
}

func (r *clickhouseRecommendation) affinityCandidates(ctx context.Context, in ForYouInput) ([]int64, error) {
	rows, err := r.conn.Query(ctx, `
		SELECT DISTINCT s.post_id
		FROM post_snapshot AS s FINAL
		INNER JOIN (
			SELECT author_profile_id
			FROM user_author_affinity_1d
			WHERE profile_id = ?
			  AND day >= today() - 30
			GROUP BY author_profile_id
			ORDER BY sum(score) DESC
			LIMIT 50
		) AS a ON s.author_profile_id = a.author_profile_id
		LEFT ANTI JOIN (
			SELECT post_id FROM user_recent_viewed FINAL WHERE profile_id = ?
		) AS v ON s.post_id = v.post_id
		WHERE s.is_active = true
		  AND s.author_profile_id != ?
		  AND s.created_at >= now() - INTERVAL 30 DAY
		LIMIT 200
	`, in.ProfileID, in.ProfileID, in.ExcludeAuthorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectInt64Rows(rows)
}

func (r *clickhouseRecommendation) GetTrendingCandidates(ctx context.Context, limit int, excludeAuthorID int64) ([]int64, error) {
	rows, err := r.conn.Query(ctx, `
		SELECT st.post_id
		FROM post_stats_1d AS st
		JOIN post_snapshot AS s FINAL ON s.post_id = st.post_id
		WHERE st.day >= today() - 3
		  AND s.author_profile_id != ?
		GROUP BY st.post_id
		ORDER BY sum(st.likes * 3 + st.comments * 5 + st.reposts * 4 + st.views) DESC
		LIMIT ?
	`, excludeAuthorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectInt64Rows(rows)
}

func (r *clickhouseRecommendation) GetAuthorAffinity(ctx context.Context, profileID int64, days int) (map[int64]float64, error) {
	rows, err := r.conn.Query(ctx, `
		SELECT author_profile_id, sum(score) AS total
		FROM user_author_affinity_1d
		WHERE profile_id = ?
		  AND day >= today() - ?
		GROUP BY author_profile_id
	`, profileID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]float64)
	for rows.Next() {
		var authorID int64
		var score int64
		if err := rows.Scan(&authorID, &score); err != nil {
			continue
		}
		result[authorID] = float64(score) / 1000.0
	}
	return result, rows.Err()
}

func (r *clickhouseRecommendation) GetCommunityAffinity(ctx context.Context, profileID int64, days int) (map[int64]float64, error) {
	rows, err := r.conn.Query(ctx, `
		SELECT community_id, sum(score) AS total
		FROM user_community_affinity_1d
		WHERE profile_id = ?
		  AND day >= today() - ?
		GROUP BY community_id
	`, profileID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]float64)
	for rows.Next() {
		var communityID int64
		var score int64
		if err := rows.Scan(&communityID, &score); err != nil {
			continue
		}
		result[communityID] = float64(score) / 1000.0
	}
	return result, rows.Err()
}

func (r *clickhouseRecommendation) GetPostStats(ctx context.Context, postIDs []int64) (map[int64]PostStats, error) {
	if len(postIDs) == 0 {
		return map[int64]PostStats{}, nil
	}
	rows, err := r.conn.Query(ctx, `
		SELECT post_id,
		       sum(views)    AS views,
		       sum(likes)    AS likes,
		       sum(comments) AS comments,
		       sum(reposts)  AS reposts
		FROM post_stats_1d
		WHERE post_id IN (?)
		GROUP BY post_id
	`, postIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]PostStats, len(postIDs))
	for rows.Next() {
		var postID int64
		var stats PostStats
		if err := rows.Scan(&postID, &stats.Views, &stats.Likes, &stats.Comments, &stats.Reposts); err != nil {
			continue
		}
		result[postID] = stats
	}
	return result, rows.Err()
}

func ScorePost(stats PostStats, authorAffinity, communityAffinity float64, isFriend, isMemberCommunity bool) float64 {
	base := math.Log1p(float64(stats.Likes*3 + stats.Comments*4 + stats.Reposts*5 + stats.Views/5))
	score := base
	if isFriend {
		score += 3.0
	}
	if isMemberCommunity {
		score += 2.5
	}
	score += authorAffinity * 2.0
	score += communityAffinity * 1.5
	return score
}

func collectInt64Rows(rows driver.Rows) ([]int64, error) {
	var result []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			continue
		}
		result = append(result, id)
	}
	return result, rows.Err()
}
