package usecase

import (
	"context"
	"errors"
	"strings"

	communitypb "github.com/go-park-mail-ru/2026_1_ARIS/proto/community"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
)

const (
	defaultLimit = 10
	maxLimit     = 50
)

var ErrInvalidInput = errors.New("invalid input")

type Service struct {
	userClient      userpb.UserServiceClient
	communityClient communitypb.CommunityServiceClient
	mediaClient     mediapb.MediaServiceClient
}

type Result struct {
	Users       []UserResult
	Communities []CommunityResult
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

func New(userClient userpb.UserServiceClient, communityClient communitypb.CommunityServiceClient, mediaClient mediapb.MediaServiceClient) *Service {
	return &Service{userClient: userClient, communityClient: communityClient, mediaClient: mediaClient}
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

	return &Result{Users: users, Communities: communities}, nil
}

func (s *Service) searchUsers(ctx context.Context, query string, limit int) ([]UserResult, error) {
	if s.userClient == nil {
		return []UserResult{}, nil
	}
	resp, err := s.userClient.SearchProfiles(ctx, &userpb.SearchProfilesRequest{Query: query, Limit: int32(limit)})
	if err != nil {
		return nil, err
	}

	users := make([]UserResult, 0, len(resp.GetProfiles()))
	for _, profile := range resp.GetProfiles() {
		user := UserResult{
			ProfileID:     profile.GetProfileId(),
			UserAccountID: profile.GetUserAccountId(),
			Username:      profile.GetUsername(),
			FirstName:     profile.GetFirstName(),
			LastName:      profile.GetLastName(),
			AvatarID:      profile.AvatarId,
		}
		user.AvatarURL = s.mediaURL(ctx, user.AvatarID)
		users = append(users, user)
	}
	return users, nil
}

func (s *Service) searchCommunities(ctx context.Context, query string, limit int) ([]CommunityResult, error) {
	if s.communityClient == nil {
		return []CommunityResult{}, nil
	}
	resp, err := s.communityClient.SearchCommunities(ctx, &communitypb.SearchCommunitiesRequest{Query: query, Limit: int32(limit)})
	if err != nil {
		return nil, err
	}

	communities := make([]CommunityResult, 0, len(resp.GetCommunities()))
	for _, item := range resp.GetCommunities() {
		community := CommunityResult{
			ID:           item.GetCommunityId(),
			ProfileID:    item.GetProfileId(),
			Username:     item.GetUsername(),
			Title:        item.GetTitle(),
			Bio:          item.Bio,
			Type:         item.GetType(),
			AvatarID:     item.AvatarId,
			CoverMediaID: item.CoverMediaId,
		}
		community.AvatarURL = s.mediaURL(ctx, community.AvatarID)
		community.CoverURL = s.mediaURL(ctx, community.CoverMediaID)
		communities = append(communities, community)
	}
	return communities, nil
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
