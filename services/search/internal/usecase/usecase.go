package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	esindex "github.com/go-park-mail-ru/2026_1_ARIS/pkg/elasticsearch"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
)

const (
	defaultLimit = 10
	maxLimit     = 50
)

var ErrInvalidInput = errors.New("invalid input")

type Service struct {
	esClient    *elasticsearch.Client
	mediaClient mediapb.MediaServiceClient
}

type Result struct {
	Users       []UserResult
	Communities []CommunityResult
	Posts       []PostResult
}

type UserResult struct {
	ProfileID     int64
	UserAccountID int64
	Username      string
	FirstName     string
	LastName      string
	AvatarID      *int64
	AvatarURL     *string
}

type CommunityResult struct {
	ID           int64
	ProfileID    int64
	Username     string
	Title        string
	Bio          *string
	Type         string
	AvatarID     *int64
	AvatarURL    *string
	CoverMediaID *int64
	CoverURL     *string
}

type PostResult struct {
	ID              int64
	Text            string
	AuthorID        int64
	AuthorProfileID int64
	AuthorUsername  string
	AuthorFirstName string
	AuthorLastName  string
	AuthorAvatarID  *int64
	AuthorAvatarURL *string
	CommunityID     *int64
	CreatedAt       time.Time
}

func New(esClient *elasticsearch.Client, mediaClient mediapb.MediaServiceClient) *Service {
	return &Service{esClient: esClient, mediaClient: mediaClient}
}

func (s *Service) Search(ctx context.Context, query string, limit int) (*Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrInvalidInput
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	users, err := s.searchUsers(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	communities, err := s.searchCommunities(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	posts, err := s.searchPosts(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	return &Result{Users: users, Communities: communities, Posts: posts}, nil
}

func (s *Service) searchUsers(ctx context.Context, query string, limit int) ([]UserResult, error) {
	body := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": map[string]any{
					"multi_match": map[string]any{
						"query":     query,
						"fields":    []string{"username^2", "full_name^1.5", "first_name", "last_name"},
						"type":      "best_fields",
						"fuzziness": "AUTO",
					},
				},
				"filter": []map[string]any{
					{"term": map[string]any{"is_active": true}},
				},
			},
		},
		"size": limit,
	}

	type hit struct {
		Source struct {
			UserAccountID int64   `json:"user_account_id"`
			ProfileID     int64   `json:"profile_id"`
			Username      string  `json:"username"`
			FirstName     string  `json:"first_name"`
			LastName      string  `json:"last_name"`
			AvatarID      *int64  `json:"avatar_id"`
		} `json:"_source"`
	}
	hits, err := esSearch[hit](ctx, s.esClient, esindex.IndexUsers, body)
	if err != nil {
		return nil, err
	}

	users := make([]UserResult, 0, len(hits))
	for _, h := range hits {
		u := UserResult{
			UserAccountID: h.Source.UserAccountID,
			ProfileID:     h.Source.ProfileID,
			Username:      h.Source.Username,
			FirstName:     h.Source.FirstName,
			LastName:      h.Source.LastName,
			AvatarID:      h.Source.AvatarID,
		}
		u.AvatarURL = s.mediaURL(ctx, u.AvatarID)
		users = append(users, u)
	}
	return users, nil
}

func (s *Service) searchCommunities(ctx context.Context, query string, limit int) ([]CommunityResult, error) {
	body := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": map[string]any{
					"multi_match": map[string]any{
						"query":     query,
						"fields":    []string{"username^2", "title^1.5", "bio"},
						"type":      "best_fields",
						"fuzziness": "AUTO",
					},
				},
				"filter": []map[string]any{
					{"term": map[string]any{"is_active": true}},
				},
			},
		},
		"size": limit,
	}

	type hit struct {
		Source struct {
			CommunityID   int64   `json:"community_id"`
			ProfileID     int64   `json:"profile_id"`
			Username      string  `json:"username"`
			Title         string  `json:"title"`
			Bio           *string `json:"bio"`
			CommunityType string  `json:"community_type"`
			AvatarID      *int64  `json:"avatar_id"`
			CoverMediaID  *int64  `json:"cover_media_id"`
		} `json:"_source"`
	}
	hits, err := esSearch[hit](ctx, s.esClient, esindex.IndexCommunities, body)
	if err != nil {
		return nil, err
	}

	communities := make([]CommunityResult, 0, len(hits))
	for _, h := range hits {
		c := CommunityResult{
			ID:           h.Source.CommunityID,
			ProfileID:    h.Source.ProfileID,
			Username:     h.Source.Username,
			Title:        h.Source.Title,
			Bio:          h.Source.Bio,
			Type:         h.Source.CommunityType,
			AvatarID:     h.Source.AvatarID,
			CoverMediaID: h.Source.CoverMediaID,
		}
		c.AvatarURL = s.mediaURL(ctx, c.AvatarID)
		c.CoverURL = s.mediaURL(ctx, c.CoverMediaID)
		communities = append(communities, c)
	}
	return communities, nil
}

func (s *Service) searchPosts(ctx context.Context, query string, limit int) ([]PostResult, error) {
	body := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": map[string]any{
					"match": map[string]any{
						"post_text": map[string]any{
							"query":     query,
							"fuzziness": "AUTO",
						},
					},
				},
				"filter": []map[string]any{
					{"term": map[string]any{"is_active": true}},
					{"term": map[string]any{"is_public": true}},
				},
			},
		},
		"size": limit,
		"sort": []map[string]any{
			{"_score": map[string]any{"order": "desc"}},
			{"created_at": map[string]any{"order": "desc"}},
		},
	}

	type hit struct {
		Source struct {
			PostID          int64   `json:"post_id"`
			PostText        string  `json:"post_text"`
			AuthorID        int64   `json:"author_id"`
			AuthorProfileID int64   `json:"author_profile_id"`
			AuthorUsername  string  `json:"author_username"`
			AuthorFirstName string  `json:"author_first_name"`
			AuthorLastName  string  `json:"author_last_name"`
			AuthorAvatarID  *int64  `json:"author_avatar_id"`
			CommunityID     *int64  `json:"community_id"`
			CreatedAt       string  `json:"created_at"`
		} `json:"_source"`
	}
	hits, err := esSearch[hit](ctx, s.esClient, esindex.IndexPosts, body)
	if err != nil {
		return nil, err
	}

	posts := make([]PostResult, 0, len(hits))
	for _, h := range hits {
		createdAt, _ := time.Parse(time.RFC3339, h.Source.CreatedAt)
		p := PostResult{
			ID:              h.Source.PostID,
			Text:            h.Source.PostText,
			AuthorID:        h.Source.AuthorID,
			AuthorProfileID: h.Source.AuthorProfileID,
			AuthorUsername:  h.Source.AuthorUsername,
			AuthorFirstName: h.Source.AuthorFirstName,
			AuthorLastName:  h.Source.AuthorLastName,
			AuthorAvatarID:  h.Source.AuthorAvatarID,
			CommunityID:     h.Source.CommunityID,
			CreatedAt:       createdAt,
		}
		p.AuthorAvatarURL = s.mediaURL(ctx, p.AuthorAvatarID)
		posts = append(posts, p)
	}
	return posts, nil
}

func esSearch[T any](ctx context.Context, client *elasticsearch.Client, index string, body map[string]any) ([]T, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	res, err := client.Search(
		client.Search.WithContext(ctx),
		client.Search.WithIndex(index),
		client.Search.WithBody(bytes.NewReader(buf)),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, errors.New("elasticsearch search error: " + res.String())
	}

	var envelope struct {
		Hits struct {
			Hits []T `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	return envelope.Hits.Hits, nil
}

func (s *Service) mediaURL(ctx context.Context, mediaID *int64) *string {
	if mediaID == nil || *mediaID <= 0 || s.mediaClient == nil {
		return nil
	}

	resp, err := s.mediaClient.GetMediaURL(ctx, &mediapb.GetMediaURLRequest{MediaId: *mediaID})
	if err != nil || resp == nil || strings.TrimSpace(resp.GetUrl()) == "" {
		return nil
	}
	url := resp.GetUrl()
	return &url
}
