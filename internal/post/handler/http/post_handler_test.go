package http

import (
	"bytes"
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	postrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/post/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/post/service"
	commentmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/comment/mock"
	communitymock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/community/mock"
	likemock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/like/mock"
	postmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/post/mock"
	repostmock "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/repost/mock"
	mediamock "github.com/go-park-mail-ru/2026_1_ARIS/proto/media/mock"
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type postHTTPMocks struct {
	posts         *postmock.MockPostRepo
	postWithMedia *postmock.MockPostWithMediaRepo
	comments      *commentmock.MockCommentRepo
	likes         *likemock.MockLikeRepo
	reposts       *repostmock.MockRepostRepo
	communities   *communitymock.MockCommunityRepo
	userClient    *usermock.MockUserServiceClient
	mediaClient   *mediamock.MockMediaServiceClient
	handler       *Handler
}

func newPostHTTPMocks(t *testing.T) (*gomock.Controller, postHTTPMocks) {
	t.Helper()

	ctrl := gomock.NewController(t)
	m := postHTTPMocks{
		posts:         postmock.NewMockPostRepo(ctrl),
		postWithMedia: postmock.NewMockPostWithMediaRepo(ctrl),
		comments:      commentmock.NewMockCommentRepo(ctrl),
		likes:         likemock.NewMockLikeRepo(ctrl),
		reposts:       repostmock.NewMockRepostRepo(ctrl),
		communities:   communitymock.NewMockCommunityRepo(ctrl),
		userClient:    usermock.NewMockUserServiceClient(ctrl),
		mediaClient:   mediamock.NewMockMediaServiceClient(ctrl),
	}
	m.handler = New(service.New(postrepo.NewStore(m.posts, m.postWithMedia, m.comments, m.likes, m.reposts, m.communities), m.userClient, m.mediaClient))
	return ctrl, m
}

func postHTTPRequest(method, target, body string, userAccountID int64, params map[string]string) *stdhttp.Request {
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

func expectPostHTTPProfile(m postHTTPMocks, accountID, profileID int64) {
	m.userClient.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: accountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)
}

func expectPostHTTPAuthor(m postHTTPMocks, profileID, accountID int64) {
	m.userClient.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{
			ProfileId: profileID, UserAccountId: accountID, FirstName: "Neo", LastName: "Anderson", Username: "neo",
		}, nil)
}

func expectPostHTTPDetails(m postHTTPMocks, postID, authorID, viewerProfileID int64) {
	expectPostHTTPAuthor(m, authorID, 10)
	m.postWithMedia.EXPECT().GetMediaByPostID(gomock.Any(), postID).Return(nil)
	m.likes.EXPECT().GetLikeCountOnPost(gomock.Any(), postID).Return(3)
	if viewerProfileID > 0 {
		m.likes.EXPECT().HasActivePostLike(gomock.Any(), postID, viewerProfileID).Return(true)
	}
}

func TestPostHandlerFeedAndPopularRoutes(t *testing.T) {
	ctrl, m := newPostHTTPMocks(t)
	defer ctrl.Finish()

	router := chi.NewRouter()
	m.handler.RegisterRoutes(router, nil)
	require.NotNil(t, router)

	now := time.Now()
	text := "<hello>"
	posts := []models.Post{
		{ID: 1, Uid: uuid.New(), Text: &text, AuthorID: 20, CreatedAt: now, IsPublicDemo: false},
		{ID: 2, Uid: uuid.New(), Text: &text, AuthorID: 20, CreatedAt: now.Add(time.Minute), IsPublicDemo: true},
	}
	m.posts.EXPECT().GetAll(gomock.Any()).Return(posts, nil).Times(2)
	for i := 0; i < 2; i++ {
		expectPostHTTPAuthor(m, 20, 10)
		m.likes.EXPECT().GetLikeCountOnPost(gomock.Any(), gomock.Any()).Return(1)
		m.comments.EXPECT().GetCommentCount(gomock.Any(), gomock.Any()).Return(2)
		m.reposts.EXPECT().GetRepostCount(gomock.Any(), gomock.Any()).Return(3)
		m.postWithMedia.EXPECT().GetMediaByPostID(gomock.Any(), gomock.Any()).Return(nil)
	}

	rec := httptest.NewRecorder()
	m.handler.GetFeed(rec, postHTTPRequest(stdhttp.MethodGet, "/feed?limit=10", "", 0, nil))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"posts"`)

	rec = httptest.NewRecorder()
	m.handler.GetPublicFeed(rec, postHTTPRequest(stdhttp.MethodGet, "/public/feed?limit=10", "", 0, nil))
	require.Equal(t, stdhttp.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	m.handler.GetPopularPosts(rec, postHTTPRequest(stdhttp.MethodGet, "/posts/popular", "", 10, nil))
	require.Equal(t, stdhttp.StatusOK, rec.Code)

	rec = httptest.NewRecorder()
	m.handler.GetPublicPopularPosts(rec, postHTTPRequest(stdhttp.MethodGet, "/public/popular-posts", "", 0, nil))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
}

func TestPostHandlerCRUDAndLikes(t *testing.T) {
	ctrl, m := newPostHTTPMocks(t)
	defer ctrl.Finish()

	text := "hello"
	updatedText := "updated"
	postID := int64(99)
	userAccountID := int64(10)
	authorID := int64(20)
	post := models.Post{ID: postID, Uid: uuid.New(), Text: &text, AuthorID: authorID, CreatedAt: time.Now(), UpdatedAt: time.Now()}

	expectPostHTTPProfile(m, userAccountID, authorID)
	m.posts.EXPECT().Save(gomock.Any(), gomock.Any()).Return(postID, nil)
	expectPostHTTPProfile(m, userAccountID, authorID)
	m.posts.EXPECT().Get(gomock.Any(), postID).Return(&post, nil)
	expectPostHTTPDetails(m, postID, authorID, authorID)
	rec := httptest.NewRecorder()
	m.handler.CreatePost(rec, postHTTPRequest(stdhttp.MethodPost, "/post/upload", `{"text":"hello"}`, userAccountID, nil))
	require.Equal(t, stdhttp.StatusCreated, rec.Code)

	expectPostHTTPProfile(m, userAccountID, authorID)
	m.posts.EXPECT().Get(gomock.Any(), postID).Return(&post, nil)
	expectPostHTTPDetails(m, postID, authorID, authorID)
	rec = httptest.NewRecorder()
	m.handler.GetPost(rec, postHTTPRequest(stdhttp.MethodGet, "/post/99", "", userAccountID, map[string]string{"id": "99"}))
	require.Equal(t, stdhttp.StatusOK, rec.Code)

	expectPostHTTPProfile(m, userAccountID, authorID)
	m.posts.EXPECT().Get(gomock.Any(), postID).Return(&post, nil)
	m.posts.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	expectPostHTTPProfile(m, userAccountID, authorID)
	updatedPost := post
	updatedPost.Text = &updatedText
	m.posts.EXPECT().Get(gomock.Any(), postID).Return(&updatedPost, nil)
	expectPostHTTPDetails(m, postID, authorID, authorID)
	rec = httptest.NewRecorder()
	m.handler.UpdatePost(rec, postHTTPRequest(stdhttp.MethodPatch, "/post/99", `{"text":"updated"}`, userAccountID, map[string]string{"id": "99"}))
	require.Equal(t, stdhttp.StatusOK, rec.Code)

	expectPostHTTPProfile(m, userAccountID, authorID)
	m.posts.EXPECT().Get(gomock.Any(), postID).Return(&post, nil)
	m.likes.EXPECT().GetPostLikeByAuthor(gomock.Any(), postID, authorID).Return(&models.Like{ID: 5, IsActive: false}, nil)
	m.likes.EXPECT().SetActive(gomock.Any(), int64(5), true).Return(nil)
	expectPostHTTPProfile(m, userAccountID, authorID)
	m.posts.EXPECT().Get(gomock.Any(), postID).Return(&post, nil)
	expectPostHTTPDetails(m, postID, authorID, authorID)
	rec = httptest.NewRecorder()
	m.handler.LikePost(rec, postHTTPRequest(stdhttp.MethodPost, "/post/99/likes", "", userAccountID, map[string]string{"id": "99"}))
	require.Equal(t, stdhttp.StatusOK, rec.Code)

	expectPostHTTPProfile(m, userAccountID, authorID)
	m.posts.EXPECT().Get(gomock.Any(), postID).Return(&post, nil)
	m.likes.EXPECT().GetPostLikeByAuthor(gomock.Any(), postID, authorID).Return(&models.Like{ID: 5, IsActive: true}, nil)
	m.likes.EXPECT().SetActive(gomock.Any(), int64(5), false).Return(nil)
	expectPostHTTPProfile(m, userAccountID, authorID)
	m.posts.EXPECT().Get(gomock.Any(), postID).Return(&post, nil)
	expectPostHTTPDetails(m, postID, authorID, authorID)
	rec = httptest.NewRecorder()
	m.handler.UnlikePost(rec, postHTTPRequest(stdhttp.MethodDelete, "/post/99/likes", "", userAccountID, map[string]string{"id": "99"}))
	require.Equal(t, stdhttp.StatusOK, rec.Code)

	expectPostHTTPProfile(m, userAccountID, authorID)
	m.posts.EXPECT().Get(gomock.Any(), postID).Return(&post, nil)
	m.posts.EXPECT().Delete(gomock.Any(), postID).Return(nil)
	rec = httptest.NewRecorder()
	m.handler.DeletePost(rec, postHTTPRequest(stdhttp.MethodDelete, "/post/99", "", userAccountID, map[string]string{"id": "99"}))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
}

func TestPostHandlerLists(t *testing.T) {
	ctrl, m := newPostHTTPMocks(t)
	defer ctrl.Finish()

	text := "hello"
	postID := int64(99)
	userAccountID := int64(10)
	authorID := int64(20)
	communityID := int64(7)
	post := models.Post{ID: postID, Uid: uuid.New(), Text: &text, AuthorID: authorID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	communityPost := post
	communityPost.CommunityID = &communityID

	expectPostHTTPProfile(m, userAccountID, authorID)
	m.posts.EXPECT().GetByAuthorID(gomock.Any(), authorID).Return([]models.Post{post}, nil)
	expectPostHTTPDetails(m, postID, authorID, 0)
	rec := httptest.NewRecorder()
	m.handler.GetMyPosts(rec, postHTTPRequest(stdhttp.MethodGet, "/post/me", "", userAccountID, nil))
	require.Equal(t, stdhttp.StatusOK, rec.Code)

	m.posts.EXPECT().GetByAuthorID(gomock.Any(), authorID).Return([]models.Post{post}, nil)
	expectPostHTTPDetails(m, postID, authorID, 0)
	rec = httptest.NewRecorder()
	m.handler.GetProfilePosts(rec, postHTTPRequest(stdhttp.MethodGet, "/post/profile/20", "", 0, map[string]string{"profileID": "20"}))
	require.Equal(t, stdhttp.StatusOK, rec.Code)

	expectPostHTTPProfile(m, userAccountID, authorID)
	m.posts.EXPECT().GetByCommunityID(gomock.Any(), communityID).Return([]models.Post{communityPost}, nil)
	expectPostHTTPDetails(m, postID, authorID, authorID)
	rec = httptest.NewRecorder()
	m.handler.GetCommunityPosts(rec, postHTTPRequest(stdhttp.MethodGet, "/post/community/7", "", userAccountID, map[string]string{"communityID": "7"}))
	require.Equal(t, stdhttp.StatusOK, rec.Code)

	m.communities.EXPECT().Get(gomock.Any(), communityID).Return(&models.Community{ID: communityID, ProfileID: authorID}, nil)
	m.posts.EXPECT().GetByCommunityID(gomock.Any(), communityID).Return([]models.Post{communityPost}, nil)
	expectPostHTTPDetails(m, postID, authorID, 0)
	rec = httptest.NewRecorder()
	m.handler.GetCommunityOfficialPosts(rec, postHTTPRequest(stdhttp.MethodGet, "/post/community/7/official", "", 0, map[string]string{"communityID": "7"}))
	require.Equal(t, stdhttp.StatusOK, rec.Code)
}

func TestPostHandlerValidationBranches(t *testing.T) {
	_, m := newPostHTTPMocks(t)

	rec := httptest.NewRecorder()
	m.handler.GetFeed(rec, postHTTPRequest(stdhttp.MethodGet, "/feed?limit=bad", "", 0, nil))
	require.Equal(t, stdhttp.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	m.handler.CreatePost(rec, postHTTPRequest(stdhttp.MethodPost, "/post/upload", `{"text":`, 10, nil))
	require.Equal(t, stdhttp.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	m.handler.GetPost(rec, postHTTPRequest(stdhttp.MethodGet, "/post/bad", "", 10, map[string]string{"id": "bad"}))
	require.Equal(t, stdhttp.StatusBadRequest, rec.Code)
}
