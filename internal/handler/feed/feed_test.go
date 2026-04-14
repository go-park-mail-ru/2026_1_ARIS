package feed

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	mock_repo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/mocks"
	mock_service "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/post"
)

func strPtr(s string) *string { return &s }

// createTestPostService создаёт реальный postService со всеми необходимыми моками репозиториев.
func createTestPostService(ctrl *gomock.Controller) (
	post.PostService,
	*mock_repo.MockPostRepo,
	*mock_repo.MockProfileRepo,
	*mock_repo.MockLikeRepo,
	*mock_repo.MockCommentRepo,
	*mock_repo.MockRepostRepo,
) {
	postRepo := mock_repo.NewMockPostRepo(ctrl)
	postWithMediaRepo := mock_repo.NewMockPostWithMediaRepo(ctrl)
	profileRepo := mock_repo.NewMockProfileRepo(ctrl)
	commentRepo := mock_repo.NewMockCommentRepo(ctrl)
	repostRepo := mock_repo.NewMockRepostRepo(ctrl)
	likeRepo := mock_repo.NewMockLikeRepo(ctrl)

	svc := post.NewPostService(
		postRepo,
		postWithMediaRepo,
		profileRepo,
		commentRepo,
		repostRepo,
		likeRepo,
	)
	return svc, postRepo, profileRepo, likeRepo, commentRepo, repostRepo
}

func TestFeedHandler_GetPopularPosts_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewFeedHandler(nil, nil, nil)

	req := httptest.NewRequest("GET", "/posts/popular", nil)
	w := httptest.NewRecorder()

	handler.GetPopularPosts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp popularPostsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Items, 3)
}

func TestFeedHandler_GetPublicPopularPosts_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewFeedHandler(nil, nil, nil)

	req := httptest.NewRequest("GET", "/public/popular-posts", nil)
	w := httptest.NewRecorder()

	handler.GetPublicPopularPosts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp popularPostsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Items, 3)
}

func TestFeedHandler_GetFeed_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	postSvc, postRepo, profileRepo, likeRepo, commentRepo, repostRepo := createTestPostService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewFeedHandler(postSvc, mockMediaSvc, mockUserSvc)

	postID := int64(1)
	postUID := uuid.New()
	authorProfileID := int64(100)
	authorUserProfileID := int64(200)

	// Мок постов для GetFeed (репозиторий возвращает все посты)
	allPosts := []models.Post{
		{
			ID:           postID,
			Uid:          postUID,
			Text:         strPtr("Hello world"),
			AuthorID:     authorProfileID,
			CreatedAt:    time.Now(),
			IsPublicDemo: false,
		},
	}
	postRepo.EXPECT().
		GetAll(gomock.Any()).
		Return(allPosts, nil)

	// Для GetPostAuthor
	postRepo.EXPECT().
		Get(gomock.Any(), postID).
		Return(&allPosts[0], nil).AnyTimes()
	profileRepo.EXPECT().
		Get(gomock.Any(), authorProfileID).
		Return(&models.Profile{ID: authorProfileID}, nil).AnyTimes()

	// Мок профиля пользователя
	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), authorProfileID).
		Return(&models.UserProfile{
			ID:        authorUserProfileID,
			FirstName: "John",
			LastName:  "Doe",
		}, nil)
	mockUserSvc.EXPECT().
		GetUserAccountByUserProfileID(gomock.Any(), authorUserProfileID).
		Return(&models.UserAccount{Username: "johndoe"}, nil)

	// Мок медиа
	mockMediaSvc.EXPECT().
		GetMediasByPostID(gomock.Any(), postID).
		Return([]models.Media{})

	// Моки счётчиков
	likeRepo.EXPECT().
		GetLikeCountOnPost(gomock.Any(), postID).
		Return(5)
	commentRepo.EXPECT().
		GetCommentCount(gomock.Any(), postID).
		Return(3)
	repostRepo.EXPECT().
		GetRepostCount(gomock.Any(), postID).
		Return(2)

	req := httptest.NewRequest("GET", "/feed", nil)
	w := httptest.NewRecorder()
	handler.GetFeed(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp FeedResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, postUID, resp.Items[0].Id)
	assert.Equal(t, "Hello world", resp.Items[0].Text)
	assert.Equal(t, "John", resp.Items[0].Author.FirstName)
	assert.Equal(t, "johndoe", resp.Items[0].Author.Username)
	assert.Equal(t, 5, resp.Items[0].Likes)
	assert.Equal(t, 3, resp.Items[0].Comments)
	assert.Equal(t, 2, resp.Items[0].Reposts)
}

func TestFeedHandler_GetFeed_InvalidLimit(t *testing.T) {
	handler := NewFeedHandler(nil, nil, nil)
	req := httptest.NewRequest("GET", "/feed?limit=invalid", nil)
	w := httptest.NewRecorder()
	handler.GetFeed(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFeedHandler_GetFeed_MethodNotAllowed(t *testing.T) {
	handler := NewFeedHandler(nil, nil, nil)
	req := httptest.NewRequest("POST", "/feed", nil)
	w := httptest.NewRecorder()
	handler.GetFeed(w, req)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestFeedHandler_GetFeed_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	postSvc, postRepo, _, _, _, _ := createTestPostService(ctrl)
	handler := NewFeedHandler(postSvc, nil, nil)

	postRepo.EXPECT().
		GetAll(gomock.Any()).
		Return(nil, errors.New("db error"))

	req := httptest.NewRequest("GET", "/feed", nil)
	w := httptest.NewRecorder()
	handler.GetFeed(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestFeedHandler_GetPublicFeed_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	postSvc, postRepo, profileRepo, likeRepo, commentRepo, repostRepo := createTestPostService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewFeedHandler(postSvc, mockMediaSvc, mockUserSvc)

	postID := int64(2)
	postUID := uuid.New()
	authorProfileID := int64(101)
	authorUserProfileID := int64(201)

	allPosts := []models.Post{
		{
			ID:           postID,
			Uid:          postUID,
			Text:         strPtr("Public post"),
			AuthorID:     authorProfileID,
			CreatedAt:    time.Now(),
			IsPublicDemo: true,
		},
	}
	postRepo.EXPECT().
		GetAll(gomock.Any()).
		Return(allPosts, nil)

	postRepo.EXPECT().
		Get(gomock.Any(), postID).
		Return(&allPosts[0], nil).AnyTimes()
	profileRepo.EXPECT().
		Get(gomock.Any(), authorProfileID).
		Return(&models.Profile{ID: authorProfileID}, nil).AnyTimes()

	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), authorProfileID).
		Return(&models.UserProfile{ID: authorUserProfileID, FirstName: "Jane", LastName: "Smith"}, nil)
	mockUserSvc.EXPECT().
		GetUserAccountByUserProfileID(gomock.Any(), authorUserProfileID).
		Return(&models.UserAccount{Username: "janesmith"}, nil)

	mockMediaSvc.EXPECT().
		GetMediasByPostID(gomock.Any(), postID).
		Return([]models.Media{})

	// Счётчики
	likeRepo.EXPECT().
		GetLikeCountOnPost(gomock.Any(), postID).
		Return(10)
	commentRepo.EXPECT().
		GetCommentCount(gomock.Any(), postID).
		Return(5)
	repostRepo.EXPECT().
		GetRepostCount(gomock.Any(), postID).
		Return(1)

	req := httptest.NewRequest("GET", "/public/feed", nil)
	w := httptest.NewRecorder()
	handler.GetPublicFeed(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp FeedResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, postUID, resp.Items[0].Id)
	assert.Equal(t, 10, resp.Items[0].Likes)
	assert.Equal(t, 5, resp.Items[0].Comments)
	assert.Equal(t, 1, resp.Items[0].Reposts)
}

func TestFeedHandler_GetPublicFeed_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	postSvc, postRepo, _, _, _, _ := createTestPostService(ctrl)
	handler := NewFeedHandler(postSvc, nil, nil)

	postRepo.EXPECT().
		GetAll(gomock.Any()).
		Return(nil, errors.New("error"))

	req := httptest.NewRequest("GET", "/public/feed", nil)
	w := httptest.NewRecorder()
	handler.GetPublicFeed(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
