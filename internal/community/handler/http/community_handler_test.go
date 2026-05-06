package http

import (
	"bytes"
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	communityrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/community/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/community/service"
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

type communityHTTPMocks struct {
	communities *communitymock.MockCommunityRepo
	userClient  *usermock.MockUserServiceClient
	mediaClient *mediamock.MockMediaServiceClient
	handler     *Handler
}

func newCommunityHTTPMocks(t *testing.T) (*gomock.Controller, communityHTTPMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	m := communityHTTPMocks{
		communities: communitymock.NewMockCommunityRepo(ctrl),
		userClient:  usermock.NewMockUserServiceClient(ctrl),
		mediaClient: mediamock.NewMockMediaServiceClient(ctrl),
	}
	m.handler = New(service.New(communityrepo.NewStore(m.communities), m.userClient, m.mediaClient))
	return ctrl, m
}

func communityHTTPFixture() models.Community {
	coverID := int64(88)
	return models.Community{
		ID: 1, Uid: uuid.New(), ProfileID: 100, Username: "team", Title: "Team",
		Type: models.PublicGroup, CoverMediaID: &coverID, IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

func communityHTTPRequest(method, target, body string, userAccountID int64, params map[string]string) *stdhttp.Request {
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	if userAccountID > 0 {
		req = req.WithContext(context.WithValue(req.Context(), "user_id", userAccountID))
	}
	if len(params) > 0 {
		rctx := chi.NewRouteContext()
		for key, value := range params {
			rctx.URLParams.Add(key, value)
		}
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}
	return req
}

func expectCommunityProfile(m communityHTTPMocks, accountID, profileID int64) {
	m.userClient.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: accountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)
}

func expectCommunityMediaURL(m communityHTTPMocks, mediaID int64, url string) {
	m.mediaClient.EXPECT().
		GetMediaURL(gomock.Any(), &mediapb.GetMediaURLRequest{MediaId: mediaID}).
		Return(&mediapb.GetMediaURLResponse{Url: url}, nil)
}

func expectCommunityDecorate(m communityHTTPMocks, community models.Community, viewerProfileID int64, role models.CommunityMemberRole) {
	avatarID := int64(77)
	m.communities.EXPECT().GetAvatarID(gomock.Any(), community.ProfileID).Return(&avatarID, nil)
	if viewerProfileID > 0 {
		m.communities.EXPECT().GetMember(gomock.Any(), community.ID, viewerProfileID).Return(&models.CommunityMember{
			CommunityID: community.ID, MemberID: viewerProfileID, Role: role, IsActive: true,
		}, nil)
	}
	expectCommunityMediaURL(m, avatarID, "https://cdn.test/avatar.png")
	if community.CoverMediaID != nil {
		expectCommunityMediaURL(m, *community.CoverMediaID, "https://cdn.test/cover.png")
	}
}

func expectCommunitySummary(m communityHTTPMocks, profileID, userAccountID int64, firstName string) {
	avatarID := int64(55)
	m.userClient.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{
			ProfileId: profileID, UserAccountId: userAccountID, FirstName: firstName, LastName: "User", Username: firstName, AvatarId: &avatarID,
		}, nil)
	expectCommunityMediaURL(m, avatarID, "https://cdn.test/member.png")
}

func TestCommunityHandlerCRUDAndLookup(t *testing.T) {
	ctrl, m := newCommunityHTTPMocks(t)
	defer ctrl.Finish()

	community := communityHTTPFixture()
	viewerAccountID := int64(10)
	viewerProfileID := int64(20)
	avatarID := int64(10)
	coverID := int64(11)
	bio := "  bio  "

	expectCommunityProfile(m, viewerAccountID, viewerProfileID)
	m.communities.EXPECT().Create(gomock.Any(), gomock.Any(), viewerProfileID, &avatarID).DoAndReturn(func(_ context.Context, input models.Community, ownerProfileID int64, gotAvatarID *int64) (*models.Community, error) {
		require.Equal(t, "Team", input.Title)
		require.Equal(t, "teamname", input.Username)
		require.Equal(t, "bio", *input.Bio)
		require.Equal(t, &coverID, input.CoverMediaID)
		return &community, nil
	})
	expectCommunityDecorate(m, community, viewerProfileID, models.Owner)

	rec := httptest.NewRecorder()
	m.handler.Create(rec, communityHTTPRequest(stdhttp.MethodPost, "/communities", `{"title":" Team ","bio":"`+bio+`","type":"public","username":" TeamName ","avatarId":10,"coverId":11}`, viewerAccountID, nil))
	require.Equal(t, stdhttp.StatusCreated, rec.Code)
	require.Contains(t, rec.Body.String(), `"username":"team"`)

	m.communities.EXPECT().List(gomock.Any(), 5, 1).Return([]models.Community{community}, nil)
	expectCommunityDecorate(m, community, 0, "")
	rec = httptest.NewRecorder()
	m.handler.List(rec, communityHTTPRequest(stdhttp.MethodGet, "/communities?limit=5&offset=1", "", 0, nil))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
	var list communityListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	require.Len(t, list.Items, 1)

	m.communities.EXPECT().Get(gomock.Any(), community.ID).Return(&community, nil)
	expectCommunityProfile(m, viewerAccountID, viewerProfileID)
	expectCommunityDecorate(m, community, viewerProfileID, models.Admin)
	rec = httptest.NewRecorder()
	m.handler.Get(rec, communityHTTPRequest(stdhttp.MethodGet, "/communities/1", "", viewerAccountID, map[string]string{"id": "1"}))
	require.Equal(t, stdhttp.StatusOK, rec.Code)

	m.communities.EXPECT().GetByProfileID(gomock.Any(), community.ProfileID).Return(&community, nil)
	expectCommunityProfile(m, viewerAccountID, viewerProfileID)
	expectCommunityDecorate(m, community, viewerProfileID, models.Owner)
	rec = httptest.NewRecorder()
	m.handler.GetByProfileID(rec, communityHTTPRequest(stdhttp.MethodGet, "/communities/by-profile/100", "", viewerAccountID, map[string]string{"profileID": "100"}))
	require.Equal(t, stdhttp.StatusOK, rec.Code)

	title := "New Title"
	username := "NewTeam"
	updated := community
	updated.Title = title
	updated.Username = "newteam"
	updated.CoverMediaID = nil
	expectCommunityProfile(m, viewerAccountID, viewerProfileID)
	m.communities.EXPECT().Get(gomock.Any(), community.ID).Return(&community, nil)
	m.communities.EXPECT().GetMember(gomock.Any(), community.ID, viewerProfileID).Return(&models.CommunityMember{Role: models.Owner, IsActive: true}, nil)
	m.communities.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, next models.Community) (*models.Community, error) {
		require.Equal(t, title, next.Title)
		require.Equal(t, "newteam", next.Username)
		require.Nil(t, next.CoverMediaID)
		return &updated, nil
	})
	m.communities.EXPECT().UpdateAvatar(gomock.Any(), updated.ProfileID, nil).Return(nil)
	expectCommunityDecorate(m, updated, viewerProfileID, models.Owner)
	rec = httptest.NewRecorder()
	m.handler.Update(rec, communityHTTPRequest(stdhttp.MethodPatch, "/communities/1", `{"title":"`+title+`","username":"`+username+`","removeCover":true,"removeAvatar":true}`, viewerAccountID, map[string]string{"id": "1"}))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"title":"New Title"`)

	expectCommunityProfile(m, viewerAccountID, viewerProfileID)
	m.communities.EXPECT().GetMember(gomock.Any(), community.ID, viewerProfileID).Return(&models.CommunityMember{Role: models.Owner, IsActive: true}, nil)
	m.communities.EXPECT().Delete(gomock.Any(), community.ID).Return(nil)
	rec = httptest.NewRecorder()
	m.handler.Delete(rec, communityHTTPRequest(stdhttp.MethodDelete, "/communities/1", "", viewerAccountID, map[string]string{"id": "1"}))
	require.Equal(t, stdhttp.StatusNoContent, rec.Code)
}

func TestCommunityHandlerMemberFlows(t *testing.T) {
	ctrl, m := newCommunityHTTPMocks(t)
	defer ctrl.Finish()

	communityID := int64(1)
	actorAccountID := int64(10)
	actorProfileID := int64(20)
	targetProfileID := int64(30)
	joined := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)

	expectCommunityProfile(m, actorAccountID, actorProfileID)
	m.communities.EXPECT().GetMember(gomock.Any(), communityID, actorProfileID).Return(&models.CommunityMember{Role: models.Admin, IsActive: true}, nil)
	m.communities.EXPECT().ListMembers(gomock.Any(), communityID, true).Return([]models.CommunityMember{
		{CommunityID: communityID, MemberID: actorProfileID, Role: models.Admin, JoinedAt: joined, IsActive: true},
		{CommunityID: communityID, MemberID: targetProfileID, Role: models.Member, JoinedAt: joined, IsActive: true},
	}, nil)
	expectCommunitySummary(m, actorProfileID, actorAccountID, "Actor")
	expectCommunitySummary(m, targetProfileID, 11, "Target")
	rec := httptest.NewRecorder()
	m.handler.ListMembers(rec, communityHTTPRequest(stdhttp.MethodGet, "/communities/1/members?includeBlocked=true", "", actorAccountID, map[string]string{"id": "1"}))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
	var members communityMembersResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &members))
	require.Len(t, members.Items, 2)

	expectCommunityProfile(m, actorAccountID, actorProfileID)
	m.communities.EXPECT().GetMember(gomock.Any(), communityID, actorProfileID).Return(nil, service.ErrCommunityMemberNotFound)
	m.communities.EXPECT().UpsertMemberRole(gomock.Any(), communityID, actorProfileID, models.Member).Return(&models.CommunityMember{
		CommunityID: communityID, MemberID: actorProfileID, Role: models.Member, JoinedAt: joined, IsActive: true,
	}, nil)
	expectCommunitySummary(m, actorProfileID, actorAccountID, "Actor")
	rec = httptest.NewRecorder()
	m.handler.Join(rec, communityHTTPRequest(stdhttp.MethodPost, "/communities/1/join", "", actorAccountID, map[string]string{"id": "1"}))
	require.Equal(t, stdhttp.StatusOK, rec.Code)

	expectCommunityProfile(m, actorAccountID, actorProfileID)
	m.communities.EXPECT().GetMember(gomock.Any(), communityID, actorProfileID).Return(&models.CommunityMember{Role: models.Member, IsActive: true}, nil)
	m.communities.EXPECT().DeactivateMember(gomock.Any(), communityID, actorProfileID).Return(nil)
	rec = httptest.NewRecorder()
	m.handler.Leave(rec, communityHTTPRequest(stdhttp.MethodPost, "/communities/1/leave", "", actorAccountID, map[string]string{"id": "1"}))
	require.Equal(t, stdhttp.StatusNoContent, rec.Code)

	expectCommunityProfile(m, actorAccountID, actorProfileID)
	m.communities.EXPECT().GetMember(gomock.Any(), communityID, actorProfileID).Return(&models.CommunityMember{Role: models.Admin, IsActive: true}, nil)
	m.communities.EXPECT().GetMember(gomock.Any(), communityID, targetProfileID).Return(&models.CommunityMember{Role: models.Member, IsActive: true}, nil)
	m.communities.EXPECT().DeactivateMember(gomock.Any(), communityID, targetProfileID).Return(nil)
	rec = httptest.NewRecorder()
	m.handler.RemoveMember(rec, communityHTTPRequest(stdhttp.MethodDelete, "/communities/1/members/30", "", actorAccountID, map[string]string{"id": "1", "profileID": "30"}))
	require.Equal(t, stdhttp.StatusNoContent, rec.Code)

	expectCommunityProfile(m, actorAccountID, actorProfileID)
	m.communities.EXPECT().GetMember(gomock.Any(), communityID, actorProfileID).Return(&models.CommunityMember{Role: models.Owner, IsActive: true}, nil)
	m.communities.EXPECT().GetMember(gomock.Any(), communityID, targetProfileID).Return(&models.CommunityMember{Role: models.Member, IsActive: true}, nil)
	m.communities.EXPECT().UpsertMemberRole(gomock.Any(), communityID, targetProfileID, models.Moderator).Return(&models.CommunityMember{
		CommunityID: communityID, MemberID: targetProfileID, Role: models.Moderator, JoinedAt: joined, IsActive: true,
	}, nil)
	expectCommunitySummary(m, targetProfileID, 11, "Target")
	rec = httptest.NewRecorder()
	m.handler.ChangeMemberRole(rec, communityHTTPRequest(stdhttp.MethodPatch, "/communities/1/members/30/role", `{"role":"manager"}`, actorAccountID, map[string]string{"id": "1", "profileID": "30"}))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"role":"moderator"`)
}

func TestCommunityHandlerRouteAndValidationBranches(t *testing.T) {
	ctrl, m := newCommunityHTTPMocks(t)
	defer ctrl.Finish()

	router := chi.NewRouter()
	m.handler.RegisterRoutes(router, nil)

	rec := httptest.NewRecorder()
	m.handler.Create(rec, communityHTTPRequest(stdhttp.MethodPost, "/communities", `{"title":"Team"`, 10, nil))
	require.Equal(t, stdhttp.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	m.handler.Get(rec, communityHTTPRequest(stdhttp.MethodGet, "/communities/bad", "", 0, map[string]string{"id": "bad"}))
	require.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}
