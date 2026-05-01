package service

import (
	"context"
	"errors"
	"strings"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/community/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	legacycommunity "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/community"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrCommunityNotFound = legacycommunity.ErrCommunityNotFound
	ErrProfileNotFound   = errors.New("profile not found")
	ErrForbidden         = errors.New("denied")
	ErrNothingToUpdate   = errors.New("no fields provided for update")
	ErrAlreadyExists     = models.ErrDuplicateEntry
)

type Service struct {
	store       repository.Store
	userClient  userpb.UserServiceClient
	mediaClient mediapb.MediaServiceClient
}

type CreateInput struct {
	Title    string
	Bio      *string
	Type     models.CommunityType
	Username string
	AvatarID *int64
}

type UpdateInput struct {
	Title    *string
	Bio      *string
	Type     *models.CommunityType
	Username *string
	AvatarID *int64
}

type Permissions struct {
	CanEdit   bool
	CanDelete bool
	CanPost   bool
}

type Membership struct {
	IsMember bool
	Role     *models.CommunityMemberRole
}

type Details struct {
	Community  models.Community
	AvatarID   *int64
	AvatarURL  *string
	Membership Membership
	Permission Permissions
}

func New(store repository.Store, userClient userpb.UserServiceClient, mediaClient ...mediapb.MediaServiceClient) *Service {
	var media mediapb.MediaServiceClient
	if len(mediaClient) > 0 {
		media = mediaClient[0]
	}
	return &Service{store: store, userClient: userClient, mediaClient: media}
}

func (s *Service) Create(ctx context.Context, userAccountID int64, input CreateInput) (*Details, error) {
	ownerProfileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if err := validateCreate(input); err != nil {
		return nil, err
	}

	community := models.Community{
		Title:    strings.TrimSpace(input.Title),
		Bio:      trimPtr(input.Bio),
		Type:     input.Type,
		Username: normalizeUsername(input.Username),
	}
	created, err := s.store.Communities.Create(ctx, community, ownerProfileID, input.AvatarID)
	if err != nil {
		return nil, err
	}
	return s.decorate(ctx, *created, ownerProfileID)
}

func (s *Service) Get(ctx context.Context, communityID int64, userAccountID *int64) (*Details, error) {
	if communityID <= 0 {
		return nil, ErrInvalidInput
	}
	community, err := s.store.Communities.Get(ctx, communityID)
	if err != nil {
		return nil, err
	}
	return s.decorateForAccount(ctx, *community, userAccountID)
}

func (s *Service) GetByProfileID(ctx context.Context, profileID int64, userAccountID *int64) (*Details, error) {
	if profileID <= 0 {
		return nil, ErrInvalidInput
	}
	community, err := s.store.Communities.GetByProfileID(ctx, profileID)
	if err != nil {
		return nil, err
	}
	return s.decorateForAccount(ctx, *community, userAccountID)
}

func (s *Service) List(ctx context.Context, limit, offset int, userAccountID *int64) ([]Details, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	communities, err := s.store.Communities.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]Details, 0, len(communities))
	for _, community := range communities {
		details, err := s.decorateForAccount(ctx, community, userAccountID)
		if err == nil {
			result = append(result, *details)
		}
	}
	return result, nil
}

func (s *Service) Update(ctx context.Context, userAccountID, communityID int64, input UpdateInput) (*Details, error) {
	if communityID <= 0 {
		return nil, ErrInvalidInput
	}
	viewerProfileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	current, err := s.store.Communities.Get(ctx, communityID)
	if err != nil {
		return nil, err
	}
	member, _ := s.store.Communities.GetMember(ctx, communityID, viewerProfileID)
	if !canManage(member) {
		return nil, ErrForbidden
	}

	if input.Title == nil && input.Bio == nil && input.Type == nil && input.Username == nil && input.AvatarID == nil {
		return nil, ErrNothingToUpdate
	}
	next := *current
	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" || len(title) > 64 {
			return nil, ErrInvalidInput
		}
		next.Title = title
	}
	if input.Bio != nil {
		next.Bio = trimPtr(input.Bio)
	}
	if input.Type != nil {
		if !isValidType(*input.Type) {
			return nil, ErrInvalidInput
		}
		next.Type = *input.Type
	}
	if input.Username != nil {
		username := normalizeUsername(*input.Username)
		if len(username) < 3 || len(username) > 20 {
			return nil, ErrInvalidInput
		}
		next.Username = username
	}
	updated, err := s.store.Communities.Update(ctx, next)
	if err != nil {
		return nil, err
	}
	if input.AvatarID != nil {
		if *input.AvatarID <= 0 {
			return nil, ErrInvalidInput
		}
		if err := s.store.Communities.UpdateAvatar(ctx, updated.ProfileID, input.AvatarID); err != nil {
			return nil, err
		}
	}
	return s.decorate(ctx, *updated, viewerProfileID)
}

func (s *Service) Delete(ctx context.Context, userAccountID, communityID int64) error {
	if communityID <= 0 {
		return ErrInvalidInput
	}
	viewerProfileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return err
	}
	member, _ := s.store.Communities.GetMember(ctx, communityID, viewerProfileID)
	if member == nil || member.Role != models.Owner {
		return ErrForbidden
	}
	return s.store.Communities.Delete(ctx, communityID)
}

func (s *Service) CanPostByProfile(ctx context.Context, communityProfileID, actorProfileID int64) (bool, error) {
	community, err := s.store.Communities.GetByProfileID(ctx, communityProfileID)
	if err != nil {
		return false, err
	}
	member, err := s.store.Communities.GetMember(ctx, community.ID, actorProfileID)
	if err != nil {
		return false, nil
	}
	return canManage(member), nil
}

func (s *Service) decorateForAccount(ctx context.Context, community models.Community, userAccountID *int64) (*Details, error) {
	if userAccountID == nil || *userAccountID <= 0 {
		return s.decorate(ctx, community, 0)
	}
	profileID, err := s.profileIDByUserAccount(ctx, *userAccountID)
	if err != nil {
		return s.decorate(ctx, community, 0)
	}
	return s.decorate(ctx, community, profileID)
}

func (s *Service) decorate(ctx context.Context, community models.Community, viewerProfileID int64) (*Details, error) {
	membership := Membership{}
	permissions := Permissions{}
	avatarID, err := s.store.Communities.GetAvatarID(ctx, community.ProfileID)
	if err != nil {
		return nil, err
	}
	if viewerProfileID > 0 {
		member, err := s.store.Communities.GetMember(ctx, community.ID, viewerProfileID)
		if err == nil && member != nil {
			role := normalizeRole(member.Role)
			membership = Membership{IsMember: true, Role: &role}
			member.Role = role
			permissions = Permissions{
				CanEdit:   canManage(member),
				CanDelete: member.Role == models.Owner,
				CanPost:   canManage(member),
			}
		}
	}
	return &Details{
		Community:  community,
		AvatarID:   avatarID,
		AvatarURL:  s.mediaURL(ctx, avatarID),
		Membership: membership,
		Permission: permissions,
	}, nil
}

func (s *Service) profileIDByUserAccount(ctx context.Context, userAccountID int64) (int64, error) {
	if userAccountID <= 0 {
		return 0, ErrInvalidInput
	}
	if s.userClient == nil {
		return 0, ErrProfileNotFound
	}
	resp, err := s.userClient.GetProfileByUserAccount(ctx, &userpb.GetProfileByUserAccountRequest{UserAccountId: userAccountID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return 0, ErrProfileNotFound
		}
		return 0, err
	}
	if resp.GetProfileId() <= 0 {
		return 0, ErrProfileNotFound
	}
	return resp.GetProfileId(), nil
}

func validateCreate(input CreateInput) error {
	if strings.TrimSpace(input.Title) == "" || len(strings.TrimSpace(input.Title)) > 64 {
		return ErrInvalidInput
	}
	if !isValidType(input.Type) {
		return ErrInvalidInput
	}
	username := normalizeUsername(input.Username)
	if len(username) < 3 || len(username) > 20 {
		return ErrInvalidInput
	}
	if input.Bio != nil && len(strings.TrimSpace(*input.Bio)) > 2047 {
		return ErrInvalidInput
	}
	return nil
}

func isValidType(value models.CommunityType) bool {
	return value == models.PublicGroup || value == models.PrivateGroup
}

func normalizeUsername(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func trimPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (s *Service) mediaURL(ctx context.Context, mediaID *int64) *string {
	if s.mediaClient == nil || mediaID == nil || *mediaID <= 0 {
		return nil
	}
	resp, err := s.mediaClient.GetMediaURL(ctx, &mediapb.GetMediaURLRequest{MediaId: *mediaID})
	if err != nil || resp == nil || strings.TrimSpace(resp.GetUrl()) == "" {
		return nil
	}
	url := resp.GetUrl()
	return &url
}

func canManage(member *models.CommunityMember) bool {
	if member == nil || !member.IsActive {
		return false
	}
	role := normalizeRole(member.Role)
	return role == models.Owner || role == models.Admin || role == models.Manager
}

func normalizeRole(role models.CommunityMemberRole) models.CommunityMemberRole {
	if role == models.Moderator {
		return models.Manager
	}
	return role
}
