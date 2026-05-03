package repository

import (
	"context"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type Store struct {
	Search SearchRepo
}

func NewStore(search SearchRepo) Store {
	return Store{Search: search}
}

type SearchRepo interface {
	SearchUsers(ctx context.Context, query string, limit int) ([]UserResult, error)
	SearchCommunities(ctx context.Context, query string, limit int) ([]CommunityResult, error)
}

type UserResult struct {
	ProfileID     int64  `db:"profile_id"`
	UserAccountID int64  `db:"user_account_id"`
	Username      string `db:"username"`
	FirstName     string `db:"first_name"`
	LastName      string `db:"last_name"`
	AvatarID      *int64 `db:"avatar_id"`
}

type CommunityResult struct {
	ID           int64                `db:"id"`
	ProfileID    int64                `db:"profile_id"`
	Username     string               `db:"username"`
	Title        string               `db:"title"`
	Bio          *string              `db:"bio"`
	Type         models.CommunityType `db:"community_type"`
	AvatarID     *int64               `db:"avatar_id"`
	CoverMediaID *int64               `db:"cover_media_id"`
}

type searchStorage struct {
	db searchDB
}

type searchDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func NewSearchStorage(db searchDB) SearchRepo {
	return &searchStorage{db: db}
}

func (storage *searchStorage) SearchUsers(ctx context.Context, query string, limit int) ([]UserResult, error) {
	pattern := likePattern(query)
	prefixPattern := likePrefixPattern(query)
	sql := `
		SELECT p.id AS profile_id, ua.id AS user_account_id, ua.username,
		       up.first_name, up.last_name, p.avatar_id
		FROM user_profile up
		JOIN user_account ua ON ua.id = up.user_account_id
		JOIN profile p ON p.id = up.profile_id
		WHERE up.is_active = TRUE
		  AND ua.is_active = TRUE
		  AND p.is_active = TRUE
		  AND (
			ua.username ILIKE $1 ESCAPE E'\\'
			OR up.first_name ILIKE $1 ESCAPE E'\\'
			OR up.last_name ILIKE $1 ESCAPE E'\\'
			OR (up.first_name || ' ' || up.last_name) ILIKE $1 ESCAPE E'\\'
		  )
		ORDER BY
		  CASE
			WHEN ua.username ILIKE $2 ESCAPE E'\\' THEN 0
			WHEN up.first_name ILIKE $2 ESCAPE E'\\' THEN 1
			WHEN up.last_name ILIKE $2 ESCAPE E'\\' THEN 2
			ELSE 3
		  END,
		  up.first_name ASC,
		  up.last_name ASC,
		  p.id ASC
		LIMIT $3`

	start := time.Now()
	rows, err := storage.db.Query(ctx, sql, pattern, prefixPattern, limit)
	logDuration(ctx, "searchRepo.SearchUsers", start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[UserResult])
}

func (storage *searchStorage) SearchCommunities(ctx context.Context, query string, limit int) ([]CommunityResult, error) {
	pattern := likePattern(query)
	prefixPattern := likePrefixPattern(query)
	sql := `
		SELECT c.id, c.profile_id, c.username, c.title, c.bio,
		       c.community_type, p.avatar_id, c.cover_media_id
		FROM community c
		JOIN profile p ON p.id = c.profile_id
		WHERE c.is_active = TRUE
		  AND p.is_active = TRUE
		  AND c.community_type = $1
		  AND (
			c.username ILIKE $2 ESCAPE E'\\'
			OR c.title ILIKE $2 ESCAPE E'\\'
			OR COALESCE(c.bio, '') ILIKE $2 ESCAPE E'\\'
		  )
		ORDER BY
		  CASE
			WHEN c.username ILIKE $3 ESCAPE E'\\' THEN 0
			WHEN c.title ILIKE $3 ESCAPE E'\\' THEN 1
			ELSE 2
		  END,
		  c.title ASC,
		  c.id ASC
		LIMIT $4`

	start := time.Now()
	rows, err := storage.db.Query(ctx, sql, models.PublicGroup, pattern, prefixPattern, limit)
	logDuration(ctx, "searchRepo.SearchCommunities", start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByName[CommunityResult])
}

func likePattern(query string) string {
	return "%" + escapeLike(strings.TrimSpace(query)) + "%"
}

func likePrefixPattern(query string) string {
	return escapeLike(strings.TrimSpace(query)) + "%"
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}

func logDuration(ctx context.Context, query string, start time.Time) {
	logg := logger.FromContext(ctx)
	if logg == nil {
		return
	}
	logg.Debug("db query", zap.String("query", query), zap.Duration("duration_ms", time.Since(start)))
}
