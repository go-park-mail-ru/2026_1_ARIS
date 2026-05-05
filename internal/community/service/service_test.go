package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/community/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	communitymock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/community/mock"
	mediapb "github.com/go-park-mail-ru/2026_1_ARIS/proto/media"
	mediamock "github.com/go-park-mail-ru/2026_1_ARIS/proto/media/mock"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type communityMocks struct {
	communities *communitymock.MockCommunityRepo
	userClient  *usermock.MockUserServiceClient
	mediaClient *mediamock.MockMediaServiceClient
	service     *Service
}

func newCommunityMocks(t *testing.T) (*gomock.Controller, communityMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	m := communityMocks{
		communities: communitymock.NewMockCommunityRepo(ctrl),
		userClient:  usermock.NewMockUserServiceClient(ctrl),
		mediaClient: mediamock.NewMockMediaServiceClient(ctrl),
	}
	m.service = New(repository.NewStore(m.communities), m.userClient, m.mediaClient)
	return ctrl, m
}

func expectCommunityProfile(m communityMocks, ctx context.Context, accountID, profileID int64) {
	m.userClient.EXPECT().
		GetProfileByUserAccount(ctx, &userpb.GetProfileByUserAccountRequest{UserAccountId: accountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)
}

func expectCommunitySummary(m communityMocks, ctx context.Context, profileID, accountID int64, roleName string) {
	avatarID := int64(55)
	m.userClient.EXPECT().
		GetProfileSummary(ctx, &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{
			ProfileId: profileID, UserAccountId: accountID, FirstName: roleName, LastName: "User", Username: roleName, AvatarId: &avatarID,
		}, nil)
	m.mediaClient.EXPECT().GetMediaURL(ctx, &mediapb.GetMediaURLRequest{MediaId: avatarID}).Return(&mediapb.GetMediaURLResponse{Url: "https://cdn.test/avatar.png"}, nil)
}

func communityFixture() models.Community {
	coverID := int64(88)
	return models.Community{
		ID: 1, Uid: uuid.New(), Title: "Community", Type: models.PublicGroup,
		ProfileID: 100, Username: "community", CoverMediaID: &coverID, IsActive: true,
	}
}

func expectDecorate(m communityMocks, ctx context.Context, community models.Community, viewerProfileID int64, role models.CommunityMemberRole) {
	avatarID := int64(77)
	m.communities.EXPECT().GetAvatarID(ctx, community.ProfileID).Return(&avatarID, nil)
	if viewerProfileID > 0 {
		m.communities.EXPECT().GetMember(ctx, community.ID, viewerProfileID).Return(&models.CommunityMember{
			CommunityID: community.ID, MemberID: viewerProfileID, Role: role, IsActive: true,
		}, nil)
	}
	m.mediaClient.EXPECT().GetMediaURL(ctx, &mediapb.GetMediaURLRequest{MediaId: avatarID}).Return(&mediapb.GetMediaURLResponse{Url: "https://cdn.test/avatar.png"}, nil)
	if community.CoverMediaID != nil {
		m.mediaClient.EXPECT().GetMediaURL(ctx, &mediapb.GetMediaURLRequest{MediaId: *community.CoverMediaID}).Return(&mediapb.GetMediaURLResponse{Url: "https://cdn.test/cover.png"}, nil)
	}
}

func TestCreateDecoratesCommunity(t *testing.T) {
	ctrl, m := newCommunityMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	inputAvatarID := int64(10)
	inputCoverID := int64(11)
	bio := "  bio  "
	created := communityFixture()

	expectCommunityProfile(m, ctx, 10, 20)
	m.communities.EXPECT().Create(ctx, gomock.Any(), int64(20), &inputAvatarID).DoAndReturn(func(_ context.Context, community models.Community, ownerProfileID int64, avatarID *int64) (*models.Community, error) {
		require.Equal(t, "Title", community.Title)
		require.Equal(t, "bio", *community.Bio)
		require.Equal(t, "myteam", community.Username)
		require.Equal(t, &inputCoverID, community.CoverMediaID)
		require.Equal(t, int64(20), ownerProfileID)
		require.Equal(t, &inputAvatarID, avatarID)
		return &created, nil
	})
	expectDecorate(m, ctx, created, 20, models.Owner)

	details, err := m.service.Create(ctx, 10, CreateInput{
		Title: " Title ", Bio: &bio, Type: models.PublicGroup, Username: " MyTeam ", AvatarID: &inputAvatarID, CoverMediaID: &inputCoverID,
	})

	require.NoError(t, err)
	require.Equal(t, created.ID, details.Community.ID)
	require.True(t, details.Membership.IsMember)
	require.True(t, details.Permission.CanDeleteCommunity)
	require.Equal(t, "https://cdn.test/avatar.png", *details.AvatarURL)
}

func TestGetListAndUpdate(t *testing.T) {
	ctrl, m := newCommunityMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	community := communityFixture()
	viewerAccountID := int64(10)
	viewerProfileID := int64(20)

	m.communities.EXPECT().Get(ctx, community.ID).Return(&community, nil)
	expectCommunityProfile(m, ctx, viewerAccountID, viewerProfileID)
	expectDecorate(m, ctx, community, viewerProfileID, models.Admin)
	details, err := m.service.Get(ctx, community.ID, &viewerAccountID)
	require.NoError(t, err)
	require.True(t, details.Permission.CanEditCommunity)

	m.communities.EXPECT().List(ctx, 20, 0).Return([]models.Community{community}, nil)
	expectDecorate(m, ctx, community, 0, "")
	list, err := m.service.List(ctx, 0, -1, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)

	title := " New Title "
	username := " NewName "
	removeCover := true
	removeAvatar := true
	updated := community
	updated.Title = "New Title"
	updated.Username = "newname"
	updated.CoverMediaID = nil
	expectCommunityProfile(m, ctx, viewerAccountID, viewerProfileID)
	m.communities.EXPECT().Get(ctx, community.ID).Return(&community, nil)
	m.communities.EXPECT().GetMember(ctx, community.ID, viewerProfileID).Return(&models.CommunityMember{Role: models.Owner, IsActive: true}, nil)
	m.communities.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, next models.Community) (*models.Community, error) {
		require.Equal(t, "New Title", next.Title)
		require.Equal(t, "newname", next.Username)
		require.Nil(t, next.CoverMediaID)
		return &updated, nil
	})
	m.communities.EXPECT().UpdateAvatar(ctx, updated.ProfileID, nil).Return(nil)
	expectDecorate(m, ctx, updated, viewerProfileID, models.Owner)

	details, err = m.service.Update(ctx, viewerAccountID, community.ID, UpdateInput{
		Title: &title, Username: &username, RemoveCover: &removeCover, RemoveAvatar: &removeAvatar,
	})

	require.NoError(t, err)
	require.Equal(t, "New Title", details.Community.Title)
}

func TestGetByProfileIDAndDelete(t *testing.T) {
	ctrl, m := newCommunityMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	community := communityFixture()
	viewerAccountID := int64(10)
	viewerProfileID := int64(20)

	m.communities.EXPECT().GetByProfileID(ctx, community.ProfileID).Return(&community, nil)
	expectCommunityProfile(m, ctx, viewerAccountID, viewerProfileID)
	expectDecorate(m, ctx, community, viewerProfileID, models.Owner)

	details, err := m.service.GetByProfileID(ctx, community.ProfileID, &viewerAccountID)

	require.NoError(t, err)
	require.Equal(t, community.ProfileID, details.Community.ProfileID)

	expectCommunityProfile(m, ctx, viewerAccountID, viewerProfileID)
	m.communities.EXPECT().GetMember(ctx, community.ID, viewerProfileID).Return(&models.CommunityMember{
		CommunityID: community.ID, MemberID: viewerProfileID, Role: models.Owner, IsActive: true,
	}, nil)
	m.communities.EXPECT().Delete(ctx, community.ID).Return(nil)

	require.NoError(t, m.service.Delete(ctx, viewerAccountID, community.ID))
}

func TestDeleteRejectsNonOwner(t *testing.T) {
	ctrl, m := newCommunityMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	expectCommunityProfile(m, ctx, 10, 20)
	m.communities.EXPECT().GetMember(ctx, int64(1), int64(20)).Return(&models.CommunityMember{Role: models.Admin, IsActive: true}, nil)

	require.ErrorIs(t, m.service.Delete(ctx, 10, 1), ErrForbidden)
}

func TestMemberFlows(t *testing.T) {
	ctrl, m := newCommunityMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	communityID := int64(1)
	actorAccountID := int64(10)
	actorProfileID := int64(20)
	targetProfileID := int64(30)
	joined := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)

	expectCommunityProfile(m, ctx, actorAccountID, actorProfileID)
	m.communities.EXPECT().GetMember(ctx, communityID, actorProfileID).Return(nil, errors.New("not member"))
	m.communities.EXPECT().UpsertMemberRole(ctx, communityID, actorProfileID, models.Member).Return(&models.CommunityMember{
		CommunityID: communityID, MemberID: actorProfileID, Role: models.Member, JoinedAt: joined, IsActive: true,
	}, nil)
	expectCommunitySummary(m, ctx, actorProfileID, actorAccountID, "Actor")
	member, err := m.service.Join(ctx, actorAccountID, communityID)
	require.NoError(t, err)
	require.True(t, member.IsSelf)
	require.Equal(t, models.Member, member.Role)

	expectCommunityProfile(m, ctx, actorAccountID, actorProfileID)
	m.communities.EXPECT().GetMember(ctx, communityID, actorProfileID).Return(&models.CommunityMember{Role: models.Member}, nil)
	m.communities.EXPECT().DeactivateMember(ctx, communityID, actorProfileID).Return(nil)
	require.NoError(t, m.service.Leave(ctx, actorAccountID, communityID))

	expectCommunityProfile(m, ctx, actorAccountID, actorProfileID)
	m.communities.EXPECT().GetMember(ctx, communityID, actorProfileID).Return(&models.CommunityMember{Role: models.Admin}, nil)
	m.communities.EXPECT().GetMember(ctx, communityID, targetProfileID).Return(&models.CommunityMember{Role: models.Moderator}, nil)
	m.communities.EXPECT().DeactivateMember(ctx, communityID, targetProfileID).Return(nil)
	require.NoError(t, m.service.RemoveMember(ctx, actorAccountID, communityID, targetProfileID))

	expectCommunityProfile(m, ctx, actorAccountID, actorProfileID)
	m.communities.EXPECT().GetMember(ctx, communityID, actorProfileID).Return(&models.CommunityMember{Role: models.Owner}, nil)
	m.communities.EXPECT().GetMember(ctx, communityID, targetProfileID).Return(&models.CommunityMember{Role: models.Member}, nil)
	m.communities.EXPECT().UpsertMemberRole(ctx, communityID, targetProfileID, models.Moderator).Return(&models.CommunityMember{
		CommunityID: communityID, MemberID: targetProfileID, Role: models.Moderator, JoinedAt: joined, IsActive: true,
	}, nil)
	expectCommunitySummary(m, ctx, targetProfileID, 11, "Target")
	changed, err := m.service.ChangeMemberRole(ctx, actorAccountID, communityID, targetProfileID, models.CommunityMemberRole("manager"))
	require.NoError(t, err)
	require.Equal(t, models.Moderator, changed.Role)
}

func TestListMembersAndPostingChecks(t *testing.T) {
	ctrl, m := newCommunityMocks(t)
	defer ctrl.Finish()

	ctx := context.Background()
	communityID := int64(1)
	viewerAccountID := int64(10)
	viewerProfileID := int64(20)
	targetProfileID := int64(30)
	community := communityFixture()
	joined := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)

	expectCommunityProfile(m, ctx, viewerAccountID, viewerProfileID)
	m.communities.EXPECT().GetMember(ctx, communityID, viewerProfileID).Return(&models.CommunityMember{Role: models.Admin}, nil)
	m.communities.EXPECT().ListMembers(ctx, communityID, true).Return([]models.CommunityMember{
		{CommunityID: communityID, MemberID: viewerProfileID, Role: models.Admin, JoinedAt: joined, IsActive: true},
		{CommunityID: communityID, MemberID: targetProfileID, Role: models.Blocked, JoinedAt: joined, IsActive: true},
	}, nil)
	expectCommunitySummary(m, ctx, viewerProfileID, viewerAccountID, "Viewer")
	expectCommunitySummary(m, ctx, targetProfileID, 11, "Target")
	members, err := m.service.ListMembers(ctx, communityID, &viewerAccountID, true)
	require.NoError(t, err)
	require.Len(t, members, 2)
	require.True(t, members[0].IsSelf)
	require.True(t, members[1].Blocked)

	m.communities.EXPECT().GetByProfileID(ctx, community.ProfileID).Return(&community, nil)
	m.communities.EXPECT().GetMember(ctx, community.ID, viewerProfileID).Return(&models.CommunityMember{Role: models.Moderator}, nil)
	ok, err := m.service.CanPostByProfile(ctx, community.ProfileID, viewerProfileID)
	require.NoError(t, err)
	require.True(t, ok)

	m.communities.EXPECT().GetMember(ctx, community.ID, targetProfileID).Return(&models.CommunityMember{Role: models.Member}, nil)
	ok, err = m.service.CanPostAsMember(ctx, community.ID, targetProfileID)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestValidationHelpers(t *testing.T) {
	require.NoError(t, validateCreate(CreateInput{Title: "T", Type: models.PublicGroup, Username: "abc"}))
	require.ErrorIs(t, validateCreate(CreateInput{Title: "", Type: models.PublicGroup, Username: "abc"}), ErrInvalidInput)
	require.ErrorIs(t, validateCreate(CreateInput{Title: "T", Type: "bad", Username: "abc"}), ErrInvalidInput)
	require.Equal(t, "abc", normalizeUsername(" AbC "))
	require.Nil(t, trimPtr(nil))
	blank := " "
	require.Nil(t, trimPtr(&blank))
	value := " x "
	require.Equal(t, "x", *trimPtr(&value))

	require.True(t, canEditCommunity(models.Owner))
	require.True(t, canPostAsCommunityRole(models.CommunityMemberRole("manager")))
	require.True(t, canPostAsMemberRole(models.Member))
	require.True(t, canManageMembers(models.Admin))
	require.True(t, canRemoveMember(models.Admin, models.Moderator))
	require.False(t, canRemoveMember(models.Moderator, models.Member))
	require.True(t, isAssignableRole(models.Blocked))
	require.False(t, isAssignableRole(models.Owner))
}
