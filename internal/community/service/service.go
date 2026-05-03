package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/community/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	legacycommunity "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/community"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidInput            = errors.New("invalid input")
	ErrCommunityNotFound       = legacycommunity.ErrCommunityNotFound
	ErrCommunityMemberNotFound = legacycommunity.ErrCommunityMemberNotFound
	ErrProfileNotFound         = errors.New("profile not found")
	ErrForbidden               = errors.New("denied")
	ErrNothingToUpdate         = errors.New("no fields provided for update")
	ErrAlreadyExists           = models.ErrDuplicateEntry
)

type Service struct {
	store       repository.Store
	userClient  userpb.UserServiceClient
	mediaClient mediapb.MediaServiceClient
}

type CreateInput struct {
	Title        string
	Bio          *string
	Type         models.CommunityType
	Username     string
	AvatarID     *int64
	CoverMediaID *int64
}

type UpdateInput struct {
	Title        *string
	Bio          *string
	Type         *models.CommunityType
	Username     *string
	AvatarID     *int64
	CoverMediaID *int64
}

type Permissions struct {
	CanEditCommunity   bool
	CanDeleteCommunity bool
	CanPost            bool
	CanPostAsCommunity bool
	CanPostAsMember    bool
	CanManageMembers   bool
	CanChangeRoles     bool
}

type Membership struct {
	IsMember bool
	Role     *models.CommunityMemberRole
	Blocked  bool
}

type MemberDetails struct {
	ProfileID     int64
	UserAccountID int64
	FirstName     string
	LastName      string
	Username      string
	AvatarID      *int64
	AvatarURL     *string
	Role          models.CommunityMemberRole
	Blocked       bool
	IsSelf        bool
	JoinedAt      string
}

type Details struct {
	Community  models.Community
	AvatarID   *int64
	AvatarURL  *string
	CoverURL   *string
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
		Title:        strings.TrimSpace(input.Title),
		Bio:          trimPtr(input.Bio),
		Type:         input.Type,
		Username:     normalizeUsername(input.Username),
		CoverMediaID: input.CoverMediaID,
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

	if input.Title == nil && input.Bio == nil && input.Type == nil && input.Username == nil && input.AvatarID == nil && input.CoverMediaID == nil {
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
	if input.CoverMediaID != nil {
		if *input.CoverMediaID <= 0 {
			return nil, ErrInvalidInput
		}
		next.CoverMediaID = input.CoverMediaID
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

func (s *Service) ListMembers(ctx context.Context, communityID int64, userAccountID *int64, includeBlocked bool) ([]MemberDetails, error) {
	if communityID <= 0 {
		return nil, ErrInvalidInput
	}
	viewerProfileID, viewerRole, err := s.viewerMembership(ctx, communityID, userAccountID)
	if err != nil {
		return nil, err
	}
	if includeBlocked && !canManageMembers(viewerRole) {
		includeBlocked = false
	}

	members, err := s.store.Communities.ListMembers(ctx, communityID, includeBlocked)
	if err != nil {
		return nil, err
	}

	result := make([]MemberDetails, 0, len(members))
	for _, member := range members {
		role := normalizeRole(member.Role)
		summary, err := s.profileSummary(ctx, member.MemberID)
		if err != nil {
			continue
		}
		result = append(result, MemberDetails{
			ProfileID:     summary.ProfileID,
			UserAccountID: summary.UserAccountID,
			FirstName:     summary.FirstName,
			LastName:      summary.LastName,
			Username:      summary.Username,
			AvatarID:      summary.AvatarID,
			AvatarURL:     summary.AvatarURL,
			Role:          role,
			Blocked:       role == models.Blocked,
			IsSelf:        viewerProfileID > 0 && summary.ProfileID == viewerProfileID,
			JoinedAt:      member.JoinedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

func (s *Service) Join(ctx context.Context, userAccountID, communityID int64) (*MemberDetails, error) {
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	member, err := s.store.Communities.GetMember(ctx, communityID, profileID)
	if err == nil && member != nil {
		role := normalizeRole(member.Role)
		if role == models.Blocked {
			return nil, ErrForbidden
		}
	}

	updated, err := s.store.Communities.UpsertMemberRole(ctx, communityID, profileID, models.Member)
	if err != nil {
		return nil, err
	}
	return s.memberDetails(ctx, *updated, profileID)
}

func (s *Service) Leave(ctx context.Context, userAccountID, communityID int64) error {
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return err
	}
	member, err := s.store.Communities.GetMember(ctx, communityID, profileID)
	if err != nil {
		return err
	}
	if normalizeRole(member.Role) == models.Owner {
		return ErrForbidden
	}
	return s.store.Communities.DeactivateMember(ctx, communityID, profileID)
}

func (s *Service) RemoveMember(ctx context.Context, userAccountID, communityID, memberProfileID int64) error {
	if memberProfileID <= 0 {
		return ErrInvalidInput
	}
	actorProfileID, actorRole, err := s.requireActorRole(ctx, communityID, userAccountID)
	if err != nil {
		return err
	}
	target, err := s.store.Communities.GetMember(ctx, communityID, memberProfileID)
	if err != nil {
		return err
	}
	targetRole := normalizeRole(target.Role)
	if memberProfileID == actorProfileID || !canRemoveMember(actorRole, targetRole) {
		return ErrForbidden
	}
	return s.store.Communities.DeactivateMember(ctx, communityID, memberProfileID)
}

func (s *Service) ChangeMemberRole(ctx context.Context, userAccountID, communityID, memberProfileID int64, role models.CommunityMemberRole) (*MemberDetails, error) {
	if memberProfileID <= 0 {
		return nil, ErrInvalidInput
	}
	role = normalizeRole(role)
	if !isAssignableRole(role) {
		return nil, ErrInvalidInput
	}
	actorProfileID, actorRole, err := s.requireActorRole(ctx, communityID, userAccountID)
	if err != nil {
		return nil, err
	}
	if actorRole != models.Owner || actorProfileID == memberProfileID {
		return nil, ErrForbidden
	}
	target, err := s.store.Communities.GetMember(ctx, communityID, memberProfileID)
	if err != nil {
		return nil, err
	}
	if normalizeRole(target.Role) == models.Owner {
		return nil, ErrForbidden
	}
	updated, err := s.store.Communities.UpsertMemberRole(ctx, communityID, memberProfileID, role)
	if err != nil {
		return nil, err
	}
	return s.memberDetails(ctx, *updated, actorProfileID)
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
	role := normalizeRole(member.Role)
	return canPostAsCommunityRole(role), nil
}

func (s *Service) CanPostAsMember(ctx context.Context, communityID, actorProfileID int64) (bool, error) {
	member, err := s.store.Communities.GetMember(ctx, communityID, actorProfileID)
	if err != nil {
		return false, nil
	}
	role := normalizeRole(member.Role)
	return canPostAsMemberRole(role), nil
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
			membership = Membership{IsMember: role != models.Blocked, Role: &role, Blocked: role == models.Blocked}
			member.Role = role
			permissions = Permissions{
				CanEditCommunity:   canEditCommunity(member.Role),
				CanDeleteCommunity: member.Role == models.Owner,
				CanPost:            canPostAsMemberRole(role) || canPostAsCommunityRole(role),
				CanPostAsCommunity: canPostAsCommunityRole(role),
				CanPostAsMember:    canPostAsMemberRole(role),
				CanManageMembers:   canManageMembers(role),
				CanChangeRoles:     role == models.Owner,
			}
		}
	}
	return &Details{
		Community:  community,
		AvatarID:   avatarID,
		AvatarURL:  s.mediaURL(ctx, avatarID),
		CoverURL:   s.mediaURL(ctx, community.CoverMediaID),
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
	if input.AvatarID != nil && *input.AvatarID <= 0 {
		return ErrInvalidInput
	}
	if input.CoverMediaID != nil && *input.CoverMediaID <= 0 {
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
	return role == models.Owner || role == models.Admin || role == models.Moderator
}

func normalizeRole(role models.CommunityMemberRole) models.CommunityMemberRole {
	if role == models.CommunityMemberRole("manager") {
		return models.Moderator
	}
	return role
}

func canEditCommunity(role models.CommunityMemberRole) bool {
	role = normalizeRole(role)
	return role == models.Owner || role == models.Admin
}

func canPostAsCommunityRole(role models.CommunityMemberRole) bool {
	role = normalizeRole(role)
	return role == models.Owner || role == models.Admin || role == models.Moderator
}

func canPostAsMemberRole(role models.CommunityMemberRole) bool {
	role = normalizeRole(role)
	return role == models.Owner || role == models.Admin || role == models.Moderator || role == models.Member
}

func canManageMembers(role models.CommunityMemberRole) bool {
	role = normalizeRole(role)
	return role == models.Owner || role == models.Admin
}

func canRemoveMember(actorRole, targetRole models.CommunityMemberRole) bool {
	actorRole = normalizeRole(actorRole)
	targetRole = normalizeRole(targetRole)
	if targetRole == models.Owner {
		return false
	}
	if actorRole == models.Owner {
		return true
	}
	if actorRole == models.Admin {
		return targetRole == models.Member || targetRole == models.Moderator || targetRole == models.Blocked
	}
	return false
}

func isAssignableRole(role models.CommunityMemberRole) bool {
	role = normalizeRole(role)
	return role == models.Admin || role == models.Moderator || role == models.Member || role == models.Blocked
}

func (s *Service) requireActorRole(ctx context.Context, communityID, userAccountID int64) (int64, models.CommunityMemberRole, error) {
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return 0, "", err
	}
	member, err := s.store.Communities.GetMember(ctx, communityID, profileID)
	if err != nil {
		return 0, "", err
	}
	role := normalizeRole(member.Role)
	return profileID, role, nil
}

func (s *Service) viewerMembership(ctx context.Context, communityID int64, userAccountID *int64) (int64, models.CommunityMemberRole, error) {
	if userAccountID == nil || *userAccountID <= 0 {
		return 0, "", nil
	}
	profileID, err := s.profileIDByUserAccount(ctx, *userAccountID)
	if err != nil {
		return 0, "", nil
	}
	member, err := s.store.Communities.GetMember(ctx, communityID, profileID)
	if err != nil {
		return profileID, "", nil
	}
	return profileID, normalizeRole(member.Role), nil
}

func (s *Service) memberDetails(ctx context.Context, member models.CommunityMember, viewerProfileID int64) (*MemberDetails, error) {
	role := normalizeRole(member.Role)
	summary, err := s.profileSummary(ctx, member.MemberID)
	if err != nil {
		return nil, err
	}
	return &MemberDetails{
		ProfileID:     summary.ProfileID,
		UserAccountID: summary.UserAccountID,
		FirstName:     summary.FirstName,
		LastName:      summary.LastName,
		Username:      summary.Username,
		AvatarID:      summary.AvatarID,
		AvatarURL:     summary.AvatarURL,
		Role:          role,
		Blocked:       role == models.Blocked,
		IsSelf:        viewerProfileID > 0 && viewerProfileID == summary.ProfileID,
		JoinedAt:      member.JoinedAt.Format(time.RFC3339),
	}, nil
}

type profileSummaryDetails struct {
	ProfileID     int64
	UserAccountID int64
	FirstName     string
	LastName      string
	Username      string
	AvatarID      *int64
	AvatarURL     *string
}

func (s *Service) profileSummary(ctx context.Context, profileID int64) (*profileSummaryDetails, error) {
	resp, err := s.userClient.GetProfileSummary(ctx, &userpb.GetProfileSummaryRequest{ProfileId: profileID})
	if err != nil {
		return nil, err
	}
	summary := &profileSummaryDetails{
		ProfileID:     resp.GetProfileId(),
		UserAccountID: resp.GetUserAccountId(),
		FirstName:     resp.GetFirstName(),
		LastName:      resp.GetLastName(),
		Username:      resp.GetUsername(),
	}
	if resp.AvatarId != nil {
		summary.AvatarID = resp.AvatarId
		summary.AvatarURL = s.mediaURL(ctx, resp.AvatarId)
	}
	return summary, nil
}
