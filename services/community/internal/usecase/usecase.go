package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/community/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidInput            = errors.New("invalid input")
	ErrCommunityNotFound       = repository.ErrCommunityNotFound
	ErrCommunityMemberNotFound = repository.ErrCommunityMemberNotFound
	ErrProfileNotFound         = errors.New("profile not found")
	ErrForbidden               = errors.New("denied")
	ErrNothingToUpdate         = errors.New("no fields provided for update")
	ErrAlreadyExists           = repository.ErrDuplicateEntry
)

type Service struct {
	communities repository.CommunityRepo
	userClient  userpb.UserServiceClient
	mediaClient mediapb.MediaServiceClient
}

type CreateInput struct {
	Title        string
	Bio          *string
	Type         model.CommunityType
	Username     string
	AvatarID     *int64
	CoverMediaID *int64
}

type UpdateInput struct {
	Title        *string
	Bio          *string
	Type         *model.CommunityType
	Username     *string
	AvatarID     *int64
	CoverMediaID *int64
	RemoveAvatar *bool
	RemoveCover  *bool
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
	Role     *model.CommunityMemberRole
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
	Role          model.CommunityMemberRole
	Blocked       bool
	IsSelf        bool
	JoinedAt      string
}

type CommunityDetails struct {
	Community model.Community
	AvatarID  *int64
}

type Details struct {
	Community  model.Community
	AvatarID   *int64
	AvatarURL  *string
	CoverURL   *string
	Membership Membership
	Permission Permissions
}

type SearchCommunityResult struct {
	ID           int64
	ProfileID    int64
	Username     string
	Title        string
	Bio          *string
	Type         model.CommunityType
	AvatarID     *int64
	CoverMediaID *int64
}

func New(communities repository.CommunityRepo, userClient userpb.UserServiceClient, mediaClient ...mediapb.MediaServiceClient) *Service {
	var media mediapb.MediaServiceClient
	if len(mediaClient) > 0 {
		media = mediaClient[0]
	}
	return &Service{communities: communities, userClient: userClient, mediaClient: media}
}

func (s *Service) Create(ctx context.Context, userAccountID int64, input CreateInput) (*Details, error) {
	ownerProfileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return nil, err
	}
	if err := validateCreate(input); err != nil {
		return nil, err
	}

	community := model.Community{
		Title:        strings.TrimSpace(input.Title),
		Bio:          trimPtr(input.Bio),
		Type:         input.Type,
		Username:     normalizeUsername(input.Username),
		CoverMediaID: input.CoverMediaID,
	}
	created, err := s.communities.Create(ctx, community, ownerProfileID, input.AvatarID)
	if err != nil {
		return nil, err
	}
	return s.decorate(ctx, *created, ownerProfileID)
}

func (s *Service) Get(ctx context.Context, communityID int64, userAccountID ...*int64) (*CommunityDetails, error) {
	if communityID <= 0 {
		return nil, ErrInvalidInput
	}
	community, err := s.communities.Get(ctx, communityID)
	if err != nil {
		return nil, err
	}
	if len(userAccountID) > 0 {
		details, err := s.decorateForAccount(ctx, *community, userAccountID[0])
		if err != nil {
			return nil, err
		}
		return &CommunityDetails{Community: details.Community, AvatarID: details.AvatarID}, nil
	}
	return s.details(ctx, *community)
}

func (s *Service) GetDetails(ctx context.Context, communityID int64, userAccountID *int64) (*Details, error) {
	if communityID <= 0 {
		return nil, ErrInvalidInput
	}
	community, err := s.communities.Get(ctx, communityID)
	if err != nil {
		return nil, err
	}
	return s.decorateForAccount(ctx, *community, userAccountID)
}

func (s *Service) GetByProfileID(ctx context.Context, profileID int64, userAccountID ...*int64) (*CommunityDetails, error) {
	if profileID <= 0 {
		return nil, ErrInvalidInput
	}
	community, err := s.communities.GetByProfileID(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if len(userAccountID) > 0 {
		details, err := s.decorateForAccount(ctx, *community, userAccountID[0])
		if err != nil {
			return nil, err
		}
		return &CommunityDetails{Community: details.Community, AvatarID: details.AvatarID}, nil
	}
	return s.details(ctx, *community)
}

func (s *Service) GetDetailsByProfileID(ctx context.Context, profileID int64, userAccountID *int64) (*Details, error) {
	if profileID <= 0 {
		return nil, ErrInvalidInput
	}
	community, err := s.communities.GetByProfileID(ctx, profileID)
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
	communities, err := s.communities.List(ctx, limit, offset)
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
	current, err := s.communities.Get(ctx, communityID)
	if err != nil {
		return nil, err
	}
	member, _ := s.communities.GetMember(ctx, communityID, viewerProfileID)
	if !canManage(member) {
		return nil, ErrForbidden
	}

	if input.Title == nil && input.Bio == nil && input.Type == nil && input.Username == nil && input.AvatarID == nil && input.CoverMediaID == nil && input.RemoveAvatar == nil && input.RemoveCover == nil {
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
	if input.RemoveCover != nil && *input.RemoveCover {
		next.CoverMediaID = nil
	} else if input.CoverMediaID != nil {
		if *input.CoverMediaID <= 0 {
			return nil, ErrInvalidInput
		}
		next.CoverMediaID = input.CoverMediaID
	}
	updated, err := s.communities.Update(ctx, next)
	if err != nil {
		return nil, err
	}
	if input.RemoveAvatar != nil && *input.RemoveAvatar {
		if err := s.communities.UpdateAvatar(ctx, updated.ProfileID, nil); err != nil {
			return nil, err
		}
	} else if input.AvatarID != nil {
		if *input.AvatarID <= 0 {
			return nil, ErrInvalidInput
		}
		if err := s.communities.UpdateAvatar(ctx, updated.ProfileID, input.AvatarID); err != nil {
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
	member, _ := s.communities.GetMember(ctx, communityID, viewerProfileID)
	if member == nil || member.Role != model.Owner {
		return ErrForbidden
	}
	return s.communities.Delete(ctx, communityID)
}

func (s *Service) GetMember(ctx context.Context, communityID, profileID int64) (*model.CommunityMember, error) {
	if communityID <= 0 || profileID <= 0 {
		return nil, ErrInvalidInput
	}
	return s.communities.GetMember(ctx, communityID, profileID)
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

	members, err := s.communities.ListMembers(ctx, communityID, includeBlocked)
	if err != nil {
		return nil, err
	}

	result := make([]MemberDetails, 0, len(members))
	for _, member := range members {
		role := NormalizeRole(member.Role)
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
			Blocked:       role == model.Blocked,
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
	member, err := s.communities.GetMember(ctx, communityID, profileID)
	if err == nil && member != nil {
		role := NormalizeRole(member.Role)
		if role == model.Blocked {
			return nil, ErrForbidden
		}
	}

	updated, err := s.communities.UpsertMemberRole(ctx, communityID, profileID, model.Member)
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
	member, err := s.communities.GetMember(ctx, communityID, profileID)
	if err != nil {
		return err
	}
	if NormalizeRole(member.Role) == model.Owner {
		return ErrForbidden
	}
	return s.communities.DeactivateMember(ctx, communityID, profileID)
}

func (s *Service) RemoveMember(ctx context.Context, userAccountID, communityID, memberProfileID int64) error {
	if memberProfileID <= 0 {
		return ErrInvalidInput
	}
	actorProfileID, actorRole, err := s.requireActorRole(ctx, communityID, userAccountID)
	if err != nil {
		return err
	}
	target, err := s.communities.GetMember(ctx, communityID, memberProfileID)
	if err != nil {
		return err
	}
	targetRole := NormalizeRole(target.Role)
	if memberProfileID == actorProfileID || !canRemoveMember(actorRole, targetRole) {
		return ErrForbidden
	}
	return s.communities.DeactivateMember(ctx, communityID, memberProfileID)
}

func (s *Service) ChangeMemberRole(ctx context.Context, userAccountID, communityID, memberProfileID int64, role model.CommunityMemberRole) (*MemberDetails, error) {
	if memberProfileID <= 0 {
		return nil, ErrInvalidInput
	}
	role = NormalizeRole(role)
	if !isAssignableRole(role) {
		return nil, ErrInvalidInput
	}
	actorProfileID, actorRole, err := s.requireActorRole(ctx, communityID, userAccountID)
	if err != nil {
		return nil, err
	}
	if actorRole != model.Owner || actorProfileID == memberProfileID {
		return nil, ErrForbidden
	}
	target, err := s.communities.GetMember(ctx, communityID, memberProfileID)
	if err != nil {
		return nil, err
	}
	if NormalizeRole(target.Role) == model.Owner {
		return nil, ErrForbidden
	}
	updated, err := s.communities.UpsertMemberRole(ctx, communityID, memberProfileID, role)
	if err != nil {
		return nil, err
	}
	return s.memberDetails(ctx, *updated, actorProfileID)
}

func (s *Service) CanPostByProfile(ctx context.Context, communityProfileID, actorProfileID int64) (bool, error) {
	if communityProfileID <= 0 || actorProfileID <= 0 {
		return false, ErrInvalidInput
	}
	community, err := s.communities.GetByProfileID(ctx, communityProfileID)
	if err != nil {
		return false, err
	}
	member, err := s.communities.GetMember(ctx, community.ID, actorProfileID)
	if err != nil {
		return false, nil
	}
	return canPostAsCommunityRole(member.Role), nil
}

func (s *Service) CanPostAsMember(ctx context.Context, communityID, actorProfileID int64) (bool, error) {
	if communityID <= 0 || actorProfileID <= 0 {
		return false, ErrInvalidInput
	}
	member, err := s.communities.GetMember(ctx, communityID, actorProfileID)
	if err != nil {
		return false, nil
	}
	return canPostAsMemberRole(member.Role), nil
}

func (s *Service) Search(ctx context.Context, query string, limit int) ([]SearchCommunityResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, ErrInvalidInput
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	communities, err := s.communities.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	result := make([]SearchCommunityResult, 0, len(communities))
	for _, community := range communities {
		result = append(result, SearchCommunityResult{
			ID:           community.ID,
			ProfileID:    community.ProfileID,
			Username:     community.Username,
			Title:        community.Title,
			Bio:          community.Bio,
			Type:         community.Type,
			AvatarID:     community.AvatarID,
			CoverMediaID: community.CoverMediaID,
		})
	}
	return result, nil
}

func (s *Service) details(ctx context.Context, community model.Community) (*CommunityDetails, error) {
	avatarID, err := s.communities.GetAvatarID(ctx, community.ProfileID)
	if err != nil {
		return nil, err
	}
	return &CommunityDetails{Community: community, AvatarID: avatarID}, nil
}

func (s *Service) decorateForAccount(ctx context.Context, community model.Community, userAccountID *int64) (*Details, error) {
	if userAccountID == nil || *userAccountID <= 0 {
		return s.decorate(ctx, community, 0)
	}
	profileID, err := s.profileIDByUserAccount(ctx, *userAccountID)
	if err != nil {
		return s.decorate(ctx, community, 0)
	}
	return s.decorate(ctx, community, profileID)
}

func (s *Service) decorate(ctx context.Context, community model.Community, viewerProfileID int64) (*Details, error) {
	membership := Membership{}
	permissions := Permissions{}
	avatarID, err := s.communities.GetAvatarID(ctx, community.ProfileID)
	if err != nil {
		return nil, err
	}
	if viewerProfileID > 0 {
		member, err := s.communities.GetMember(ctx, community.ID, viewerProfileID)
		if err == nil && member != nil {
			role := NormalizeRole(member.Role)
			membership = Membership{IsMember: role != model.Blocked, Role: &role, Blocked: role == model.Blocked}
			member.Role = role
			permissions = Permissions{
				CanEditCommunity:   canEditCommunity(member.Role),
				CanDeleteCommunity: member.Role == model.Owner,
				CanPost:            canPostAsMemberRole(role) || canPostAsCommunityRole(role),
				CanPostAsCommunity: canPostAsCommunityRole(role),
				CanPostAsMember:    canPostAsMemberRole(role),
				CanManageMembers:   canManageMembers(role),
				CanChangeRoles:     role == model.Owner,
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

func isValidType(value model.CommunityType) bool {
	return value == model.PublicGroup || value == model.PrivateGroup
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

func canManage(member *model.CommunityMember) bool {
	if member == nil || !member.IsActive {
		return false
	}
	role := NormalizeRole(member.Role)
	return role == model.Owner || role == model.Admin || role == model.Moderator
}

func NormalizeRole(role model.CommunityMemberRole) model.CommunityMemberRole {
	if role == model.CommunityMemberRole("manager") {
		return model.Moderator
	}
	return role
}

func canEditCommunity(role model.CommunityMemberRole) bool {
	role = NormalizeRole(role)
	return role == model.Owner || role == model.Admin
}

func canPostAsCommunityRole(role model.CommunityMemberRole) bool {
	role = NormalizeRole(role)
	return role == model.Owner || role == model.Admin || role == model.Moderator
}

func canPostAsMemberRole(role model.CommunityMemberRole) bool {
	role = NormalizeRole(role)
	return role == model.Owner || role == model.Admin || role == model.Moderator || role == model.Member
}

func canManageMembers(role model.CommunityMemberRole) bool {
	role = NormalizeRole(role)
	return role == model.Owner || role == model.Admin
}

func canRemoveMember(actorRole, targetRole model.CommunityMemberRole) bool {
	actorRole = NormalizeRole(actorRole)
	targetRole = NormalizeRole(targetRole)
	if targetRole == model.Owner {
		return false
	}
	if actorRole == model.Owner {
		return true
	}
	if actorRole == model.Admin {
		return targetRole == model.Member || targetRole == model.Moderator || targetRole == model.Blocked
	}
	return false
}

func isAssignableRole(role model.CommunityMemberRole) bool {
	role = NormalizeRole(role)
	return role == model.Admin || role == model.Moderator || role == model.Member || role == model.Blocked
}

func (s *Service) requireActorRole(ctx context.Context, communityID, userAccountID int64) (int64, model.CommunityMemberRole, error) {
	profileID, err := s.profileIDByUserAccount(ctx, userAccountID)
	if err != nil {
		return 0, "", err
	}
	member, err := s.communities.GetMember(ctx, communityID, profileID)
	if err != nil {
		return 0, "", err
	}
	role := NormalizeRole(member.Role)
	return profileID, role, nil
}

func (s *Service) viewerMembership(ctx context.Context, communityID int64, userAccountID *int64) (int64, model.CommunityMemberRole, error) {
	if userAccountID == nil || *userAccountID <= 0 {
		return 0, "", nil
	}
	profileID, err := s.profileIDByUserAccount(ctx, *userAccountID)
	if err != nil {
		return 0, "", nil
	}
	member, err := s.communities.GetMember(ctx, communityID, profileID)
	if err != nil {
		return profileID, "", nil
	}
	return profileID, NormalizeRole(member.Role), nil
}

func (s *Service) memberDetails(ctx context.Context, member model.CommunityMember, viewerProfileID int64) (*MemberDetails, error) {
	role := NormalizeRole(member.Role)
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
		Blocked:       role == model.Blocked,
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
