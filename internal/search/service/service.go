package service

import (
	"context"
	"errors"
	"strings"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/search/repository"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
)

const (
	defaultLimit = 10
	maxLimit     = 50
)

var ErrInvalidInput = errors.New("invalid input")

type Service struct {
	store       repository.Store
	mediaClient mediapb.MediaServiceClient
}

func New(store repository.Store, mediaClient mediapb.MediaServiceClient) *Service {
	return &Service{store: store, mediaClient: mediaClient}
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

	users, err := s.store.Search.SearchUsers(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	communities, err := s.store.Search.SearchCommunities(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	return &Result{
		Users:       s.mapUsers(ctx, users),
		Communities: s.mapCommunities(ctx, communities),
	}, nil
}

func (s *Service) mapUsers(ctx context.Context, users []repository.UserResult) []UserResult {
	result := make([]UserResult, 0, len(users))
	for _, user := range users {
		result = append(result, UserResult{
			ProfileID:     user.ProfileID,
			UserAccountID: user.UserAccountID,
			Username:      user.Username,
			FirstName:     user.FirstName,
			LastName:      user.LastName,
			AvatarID:      user.AvatarID,
			AvatarURL:     s.mediaURL(ctx, user.AvatarID),
		})
	}
	return result
}

func (s *Service) mapCommunities(ctx context.Context, communities []repository.CommunityResult) []CommunityResult {
	result := make([]CommunityResult, 0, len(communities))
	for _, community := range communities {
		result = append(result, CommunityResult{
			ID:           community.ID,
			ProfileID:    community.ProfileID,
			Username:     community.Username,
			Title:        community.Title,
			Bio:          community.Bio,
			Type:         community.Type,
			AvatarID:     community.AvatarID,
			AvatarURL:    s.mediaURL(ctx, community.AvatarID),
			CoverMediaID: community.CoverMediaID,
			CoverURL:     s.mediaURL(ctx, community.CoverMediaID),
		})
	}
	return result
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
