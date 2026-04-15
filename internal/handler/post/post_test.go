package post

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	mock_repo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/mocks"
	mock_service "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/post"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
)

func strPtr(s string) *string { return &s }

type contextKey string

const LoggerKey contextKey = "logger"

func contextWithObservedLogger() (context.Context, *observer.ObservedLogs) {
	core, recorded := observer.New(zap.DebugLevel)
	loggerObserved := zap.New(core)
	ctx := logger.WithLogger(context.Background(), loggerObserved)
	return ctx, recorded
}

// createTestPostService создаёт реальный postService с замоканными репозиториями.
func createTestPostService(ctrl *gomock.Controller) (post.PostService, *mock_repo.MockPostRepo, *mock_repo.MockPostWithMediaRepo) {
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
	return svc, postRepo, postWithMediaRepo
}

func TestPostHandler_CreatePost_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	postSvc, postRepo, postWithMediaRepo := createTestPostService(ctrl)

	handler := NewPostHandler(mockUserSvc, postSvc, mockMediaSvc)

	userAccountID := int64(1)
	profileID := int64(100)
	postText := "Hello, world!"

	// Мок получения профиля
	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userAccountID).
		Return(&models.Profile{ID: profileID}, nil)

	// Мок сохранения поста
	postRepo.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(models.Post{})).
		DoAndReturn(func(ctx context.Context, p models.Post) (int64, error) {
			assert.Equal(t, profileID, p.AuthorID)
			assert.Equal(t, postText, *p.Text)
			return 123, nil
		})

	// Мок прикрепления медиа (пустой список)
	// В AttachMedia вызывается Save для каждого элемента MediaRequestData
	// Если Media == nil, то AttachMedia не вызывается вовсе, поэтому мокать не нужно.
	// Но для надёжности можно оставить AnyTimes
	postWithMediaRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	// Мок получения профиля пользователя для ответа
	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), profileID).
		Return(&models.UserProfile{FirstName: "John", LastName: "Doe"}, nil)

	reqBody := PostCreationRequest{
		Text:  &postText,
		Media: nil,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", userAccountID)
	req = req.WithContext(ctx)

	mockLogger := zap.NewNop()
	ctx = logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.CreatePost(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp PostCreationResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, int64(123), resp.ID)
	assert.Equal(t, profileID, resp.ProfileID)
	assert.Equal(t, "John", resp.FirstName)
	assert.Equal(t, "Doe", resp.LastName)
}

func TestPostHandler_CreatePost_Unauthorized(t *testing.T) {
	handler := NewPostHandler(nil, nil, nil)
	req := httptest.NewRequest("POST", "/api/posts", nil)

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.CreatePost(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPostHandler_CreatePost_NoContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	postSvc, _, _ := createTestPostService(ctrl)

	handler := NewPostHandler(mockUserSvc, postSvc, mockMediaSvc)

	userAccountID := int64(1)
	profileID := int64(100)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userAccountID).
		Return(&models.Profile{ID: profileID}, nil)

	reqBody := PostCreationRequest{} // пустой
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/posts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", userAccountID)
	req = req.WithContext(ctx)

	mockLogger := zap.NewNop()
	ctx = logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.CreatePost(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), xerrors.PostContentRequired)
}

func TestPostHandler_GetMyPosts_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	postSvc, postRepo, _ := createTestPostService(ctrl)

	handler := NewPostHandler(mockUserSvc, postSvc, mockMediaSvc)

	userAccountID := int64(1)
	profileID := int64(100)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userAccountID).
		Return(&models.Profile{ID: profileID}, nil)

	posts := []models.Post{
		{ID: 10, Text: strPtr("Post 1"), AuthorID: profileID, CreatedAt: time.Now()},
		{ID: 11, Text: strPtr("Post 2"), AuthorID: profileID, CreatedAt: time.Now()},
	}
	postRepo.EXPECT().
		GetByAuthorID(gomock.Any(), profileID).
		Return(posts, nil)

	// Медиа для постов
	mockMediaSvc.EXPECT().
		GetMediasByPostID(gomock.Any(), int64(10)).
		Return([]models.Media{}).AnyTimes()
	mockMediaSvc.EXPECT().
		GetMediasByPostID(gomock.Any(), int64(11)).
		Return([]models.Media{}).AnyTimes()

	req := httptest.NewRequest("GET", "/api/posts/my", nil)
	ctx := context.WithValue(req.Context(), "user_id", userAccountID)
	req = req.WithContext(ctx)

	mockLogger := zap.NewNop()
	ctx = logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetMyPosts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []PostListItemResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp, 2)
}

func TestPostHandler_GetProfilePosts_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	postSvc, postRepo, _ := createTestPostService(ctrl)

	handler := NewPostHandler(nil, postSvc, mockMediaSvc)

	targetProfileID := int64(200)
	posts := []models.Post{
		{ID: 30, Text: strPtr("Profile Post"), AuthorID: targetProfileID},
	}
	postRepo.EXPECT().
		GetByAuthorID(gomock.Any(), targetProfileID).
		Return(posts, nil)

	mockMediaSvc.EXPECT().
		GetMediasByPostID(gomock.Any(), int64(30)).
		Return([]models.Media{}).AnyTimes()

	req := httptest.NewRequest("GET", "/api/profiles/200/posts", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("profileID", "200")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetProfilePosts(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []PostListItemResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp, 1)
}

func TestPostHandler_DeletePost_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	postSvc, postRepo, _ := createTestPostService(ctrl)

	handler := NewPostHandler(mockUserSvc, postSvc, nil)

	userAccountID := int64(1)
	profileID := int64(100)
	postID := int64(55)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userAccountID).
		Return(&models.Profile{ID: profileID}, nil)

	// Важно: вернуть пост с заполненным ID
	postRepo.EXPECT().
		Get(gomock.Any(), postID).
		Return(&models.Post{ID: postID, AuthorID: profileID}, nil)

	postRepo.EXPECT().
		Delete(gomock.Any(), postID).
		Return(nil)

	req := httptest.NewRequest("DELETE", "/api/posts/55", nil)
	ctx := context.WithValue(req.Context(), "user_id", userAccountID)
	req = req.WithContext(ctx)

	mockLogger := zap.NewNop()
	ctx = logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "55")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.DeletePost(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPostHandler_DeletePost_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	postSvc, postRepo, _ := createTestPostService(ctrl)

	handler := NewPostHandler(mockUserSvc, postSvc, nil)

	userAccountID := int64(1)
	profileID := int64(100)
	postID := int64(55)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userAccountID).
		Return(&models.Profile{ID: profileID}, nil)

	postRepo.EXPECT().
		Get(gomock.Any(), postID).
		Return(&models.Post{AuthorID: 999}, nil)

	req := httptest.NewRequest("DELETE", "/api/posts/55", nil)
	ctx := context.WithValue(req.Context(), "user_id", userAccountID)
	req = req.WithContext(ctx)

	mockLogger := zap.NewNop()
	ctx = logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "55")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.DeletePost(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestPostHandler_GetPost_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	postSvc, postRepo, _ := createTestPostService(ctrl)

	handler := NewPostHandler(mockUserSvc, postSvc, mockMediaSvc)

	postID := int64(77)
	profileID := int64(200)
	userAccountID := int64(10)

	postRepo.EXPECT().
		Get(gomock.Any(), postID).
		Return(&models.Post{
			ID:        postID,
			Text:      strPtr("Some content"),
			AuthorID:  profileID,
			CreatedAt: time.Now(),
		}, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), profileID).
		Return(&models.UserProfile{
			FirstName:     "Alice",
			LastName:      "Smith",
			ProfileID:     profileID,
			UserAccountID: userAccountID,
		}, nil)

	mockUserSvc.EXPECT().
		GetProfileByProfileID(gomock.Any(), profileID).
		Return(&models.Profile{ID: profileID}, nil)

	// GetMediasByPostID возвращает только []models.Media, без ошибки
	mockMediaSvc.EXPECT().
		GetMediasByPostID(gomock.Any(), postID).
		Return([]models.Media{})

	req := httptest.NewRequest("GET", "/api/posts/77", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "77")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetPost(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp PostCreationResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, postID, resp.ID)
	assert.Equal(t, "Alice", resp.FirstName)
	assert.Equal(t, "Smith", resp.LastName)
}

func TestPostHandler_UpdatePost_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	postSvc, postRepo, _ := createTestPostService(ctrl) // postWithMediaRepo не используется в этом тесте

	handler := NewPostHandler(mockUserSvc, postSvc, mockMediaSvc)

	userAccountID := int64(1)
	profileID := int64(100)
	postID := int64(88)
	newText := "Updated text"

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userAccountID).
		Return(&models.Profile{ID: profileID}, nil)

	// Первый вызов Get для проверки прав
	postRepo.EXPECT().
		Get(gomock.Any(), postID).
		Return(&models.Post{ID: postID, AuthorID: profileID}, nil)

	// Обновление поста
	postRepo.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(models.Post{})).
		DoAndReturn(func(ctx context.Context, p models.Post) error {
			assert.Equal(t, postID, p.ID)
			assert.Equal(t, newText, *p.Text)
			return nil
		})

	// Так как request.Media == nil, ReplaceMedia не вызывается,
	// поэтому ожиданий для DeleteByPostID нет.

	// Второй вызов Get для получения обновлённого поста
	postRepo.EXPECT().
		Get(gomock.Any(), postID).
		Return(&models.Post{
			ID:        postID,
			Text:      &newText,
			AuthorID:  profileID,
			CreatedAt: time.Now(),
		}, nil)

	mockMediaSvc.EXPECT().
		GetMediasByPostID(gomock.Any(), postID).
		Return([]models.Media{})

	reqBody := PostCreationRequest{Text: &newText}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PUT", "/api/posts/88", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_id", userAccountID)
	req = req.WithContext(ctx)

	mockLogger := zap.NewNop()
	ctx = logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "88")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	handler.UpdatePost(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp PostCreationResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, postID, resp.ID)
	assert.Equal(t, newText, *resp.Text)
}

func TestCreatePost_MissingUserID_LogsWarning(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	postSvc, _, _ := createTestPostService(ctrl) // postWithMediaRepo не используется в этом тесте

	handler := NewPostHandler(mockUserSvc, postSvc, mockMediaSvc)

	ctx, observed := contextWithObservedLogger()
	req := httptest.NewRequest(http.MethodPost, "/posts", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.CreatePost(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	logs := observed.All()
	require.Equal(t, 1, len(logs))
	entry := logs[0]
	assert.Equal(t, zap.WarnLevel, entry.Level)
	assert.Equal(t, "cannot_create_post_missing_user", entry.Message)
	assert.Equal(t, "/posts", entry.ContextMap()["path"])
}

func TestCreatePost_ProfileNotFound_LogsWarning(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	postSvc, _, _ := createTestPostService(ctrl) // postWithMediaRepo не используется в этом тесте

	handler := NewPostHandler(mockUserSvc, postSvc, mockMediaSvc)

	userID := int64(123)
	ctx, observed := contextWithObservedLogger()
	ctx = context.WithValue(ctx, "user_id", userID)
	req := httptest.NewRequest(http.MethodPost, "/posts", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(nil, xerrors.ProfileNotFound)

	handler.CreatePost(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	logs := observed.All()
	require.Equal(t, 1, len(logs))
	entry := logs[0]
	assert.Equal(t, zap.WarnLevel, entry.Level)
	assert.Equal(t, "cannot_create_post_profile_not_found", entry.Message)
	assert.Equal(t, userID, entry.ContextMap()["userAccount_id"])
}

func TestCreatePost_GetProfileError_LogsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	postSvc, _, _ := createTestPostService(ctrl) // postWithMediaRepo не используется в этом тесте

	handler := NewPostHandler(mockUserSvc, postSvc, mockMediaSvc)

	userID := int64(123)
	ctx, observed := contextWithObservedLogger()
	ctx = context.WithValue(ctx, "user_id", userID)
	req := httptest.NewRequest(http.MethodPost, "/posts", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	dbErr := errors.New("db connection error")
	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(nil, dbErr)

	handler.CreatePost(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	logs := observed.All()
	require.Equal(t, 1, len(logs))
	entry := logs[0]
	assert.Equal(t, zap.ErrorLevel, entry.Level)
	assert.Equal(t, "failed_to_get_profile", entry.Message)
	assert.Equal(t, userID, entry.ContextMap()["userAccount_id"])
	assert.Contains(t, entry.ContextMap()["error"], dbErr.Error())
}

func TestCreatePost_InvalidJSONBody_LogsWarning(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	postSvc, _, _ := createTestPostService(ctrl) // postWithMediaRepo не используется в этом тесте

	handler := NewPostHandler(mockUserSvc, postSvc, mockMediaSvc)

	userID := int64(123)
	profile := &models.Profile{ID: 456}
	ctx, observed := contextWithObservedLogger()
	ctx = context.WithValue(ctx, "user_id", userID)

	body := bytes.NewBufferString(`{"text": "hello", "media": }`) // broken JSON
	req := httptest.NewRequest(http.MethodPost, "/posts", body).WithContext(ctx)
	rec := httptest.NewRecorder()

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(profile, nil)

	handler.CreatePost(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	logs := observed.All()
	// Ищем нужное сообщение среди всех логов
	var found bool
	for _, entry := range logs {
		if entry.Message == "cannot_create_post_invalid_body" {
			found = true
			assert.Equal(t, zap.WarnLevel, entry.Level)
			assert.Equal(t, "/posts", entry.ContextMap()["path"])
			assert.NotNil(t, entry.ContextMap()["error"])
			break
		}
	}
	assert.True(t, found, "expected log not found")
}

func TestCreatePost_EmptyContent_LogsWarning(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	postSvc, _, _ := createTestPostService(ctrl) // postWithMediaRepo не используется в этом тесте

	handler := NewPostHandler(mockUserSvc, postSvc, mockMediaSvc)

	userID := int64(123)
	profile := &models.Profile{ID: 456}
	ctx, observed := contextWithObservedLogger()
	ctx = context.WithValue(ctx, "user_id", userID)

	requestBody := PostCreationRequest{Text: nil, Media: nil}
	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/posts", bytes.NewBuffer(bodyBytes)).WithContext(ctx)
	rec := httptest.NewRecorder()

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(profile, nil)

	handler.CreatePost(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	logs := observed.All()
	require.Equal(t, 1, len(logs))
	entry := logs[0]
	assert.Equal(t, zap.WarnLevel, entry.Level)
	assert.Equal(t, "cannot_create_post_empty_content", entry.Message)
	assert.Equal(t, "/posts", entry.ContextMap()["path"])
}

func TestPostHandler_buildPostListItemResponse_WithMediaAndUpdatedAt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	handler := NewPostHandler(nil, nil, mockMediaSvc)

	text := "<hello>"
	updatedAt := time.Now()
	postModel := models.Post{
		ID:        12,
		AuthorID:  34,
		Text:      &text,
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: updatedAt,
	}

	mockMediaSvc.EXPECT().
		GetMediasByPostID(gomock.Any(), int64(12)).
		Return([]models.Media{{ID: 1, Link: "a.png"}, {ID: 2, Link: "b.png"}})

	resp := handler.buildPostListItemResponse(context.Background(), postModel)

	require.Equal(t, int64(12), resp.ID)
	require.Equal(t, int64(34), resp.ProfileID)
	require.Equal(t, "&lt;hello&gt;", resp.Text)
	require.Len(t, resp.Media, 2)
	require.Equal(t, []string{"a.png", "b.png"}, resp.MediaURL)
	require.NotNil(t, resp.UpdatedAt)
}

func TestPostHandler_CreatePost_ProfileNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := NewPostHandler(mockUserSvc, nil, nil)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), int64(1)).
		Return(nil, xerrors.ProfileNotFound)

	req := httptest.NewRequest("POST", "/api/posts", bytes.NewReader([]byte(`{}`)))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(1)))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.CreatePost(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPostHandler_CreatePost_SaveError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	postSvc, postRepo, _ := createTestPostService(ctrl)
	handler := NewPostHandler(mockUserSvc, postSvc, mockMediaSvc)

	text := "hello"
	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), int64(1)).
		Return(&models.Profile{ID: 100}, nil)

	postRepo.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(models.Post{})).
		Return(int64(0), errors.New("save failed"))

	reqBody := PostCreationRequest{Text: &text}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/posts", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(1)))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.CreatePost(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPostHandler_CreatePost_UserProfileErrorAfterSave(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	postSvc, postRepo, _ := createTestPostService(ctrl)
	handler := NewPostHandler(mockUserSvc, postSvc, mockMediaSvc)

	text := "hello"
	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), int64(1)).
		Return(&models.Profile{ID: 100}, nil)

	postRepo.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(models.Post{})).
		Return(int64(123), nil)

	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), int64(100)).
		Return(nil, errors.New("missing user profile"))

	reqBody := PostCreationRequest{Text: &text}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/posts", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(1)))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.CreatePost(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPostHandler_GetMyPosts_PostServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	postSvc, postRepo, _ := createTestPostService(ctrl)
	handler := NewPostHandler(mockUserSvc, postSvc, nil)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), int64(1)).
		Return(&models.Profile{ID: 100}, nil)

	postRepo.EXPECT().
		GetByAuthorID(gomock.Any(), int64(100)).
		Return(nil, errors.New("boom"))

	req := httptest.NewRequest("GET", "/api/posts/my", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(1)))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.GetMyPosts(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPostHandler_GetProfilePosts_InvalidID(t *testing.T) {
	handler := NewPostHandler(nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/profiles/bad/posts", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("profileID", "bad")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.GetProfilePosts(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostHandler_GetPost_InvalidID(t *testing.T) {
	handler := NewPostHandler(nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/posts/bad", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.GetPost(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostHandler_UpdatePost_InvalidBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	postSvc, postRepo, _ := createTestPostService(ctrl)
	handler := NewPostHandler(mockUserSvc, postSvc, nil)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), int64(1)).
		Return(&models.Profile{ID: 100}, nil)

	postRepo.EXPECT().
		Get(gomock.Any(), int64(5)).
		Return(&models.Post{ID: 5, AuthorID: 100}, nil)

	req := httptest.NewRequest("PATCH", "/api/posts/5", bytes.NewReader([]byte(`{`)))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "5")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(1)))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.UpdatePost(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNewPostHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)

	handler := NewPostHandler(mockUserSvc, nil, mockMediaSvc)

	require.NotNil(t, handler)
}

func TestPostHandler_DeletePost_Unauthorized(t *testing.T) {
	handler := NewPostHandler(nil, nil, nil)

	req := httptest.NewRequest("DELETE", "/api/posts/55", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.DeletePost(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPostHandler_DeletePost_InvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := NewPostHandler(mockUserSvc, nil, nil)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), int64(1)).
		Return(&models.Profile{ID: 100}, nil)

	req := httptest.NewRequest("DELETE", "/api/posts/bad", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(1)))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.DeletePost(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostHandler_DeletePost_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	postSvc, postRepo, _ := createTestPostService(ctrl)
	handler := NewPostHandler(mockUserSvc, postSvc, nil)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), int64(1)).
		Return(&models.Profile{ID: 100}, nil)

	postRepo.EXPECT().
		Get(gomock.Any(), int64(55)).
		Return(nil, xerrors.PostNotFound)

	req := httptest.NewRequest("DELETE", "/api/posts/55", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "55")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(1)))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.DeletePost(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPostHandler_GetPost_PostNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	postSvc, postRepo, _ := createTestPostService(ctrl)
	handler := NewPostHandler(nil, postSvc, nil)

	postRepo.EXPECT().
		Get(gomock.Any(), int64(77)).
		Return(nil, xerrors.PostNotFound)

	req := httptest.NewRequest("GET", "/api/posts/77", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "77")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.GetPost(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPostHandler_GetPost_UserProfileNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	postSvc, postRepo, _ := createTestPostService(ctrl)
	handler := NewPostHandler(mockUserSvc, postSvc, nil)

	postRepo.EXPECT().
		Get(gomock.Any(), int64(77)).
		Return(&models.Post{ID: 77, AuthorID: 200}, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), int64(200)).
		Return(nil, xerrors.UserProfileNotFound)

	req := httptest.NewRequest("GET", "/api/posts/77", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "77")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.GetPost(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPostHandler_GetPost_ProfileLookupNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	postSvc, postRepo, _ := createTestPostService(ctrl)
	handler := NewPostHandler(mockUserSvc, postSvc, nil)

	postRepo.EXPECT().
		Get(gomock.Any(), int64(77)).
		Return(&models.Post{ID: 77, AuthorID: 200}, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), int64(200)).
		Return(&models.UserProfile{
			FirstName:     "Alice",
			LastName:      "Smith",
			ProfileID:     200,
			UserAccountID: 10,
		}, nil)

	mockUserSvc.EXPECT().
		GetProfileByProfileID(gomock.Any(), int64(200)).
		Return(nil, xerrors.ProfileNotFound)

	req := httptest.NewRequest("GET", "/api/posts/77", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "77")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.GetPost(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestPostHandler_UpdatePost_Unauthorized(t *testing.T) {
	handler := NewPostHandler(nil, nil, nil)

	req := httptest.NewRequest("PATCH", "/api/posts/5", nil)
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.UpdatePost(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPostHandler_UpdatePost_InvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := NewPostHandler(mockUserSvc, nil, nil)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), int64(1)).
		Return(&models.Profile{ID: 100}, nil)

	req := httptest.NewRequest("PATCH", "/api/posts/bad", bytes.NewReader([]byte(`{}`)))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(1)))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.UpdatePost(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostHandler_UpdatePost_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	postSvc, postRepo, _ := createTestPostService(ctrl)
	handler := NewPostHandler(mockUserSvc, postSvc, nil)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), int64(1)).
		Return(&models.Profile{ID: 100}, nil)

	postRepo.EXPECT().
		Get(gomock.Any(), int64(5)).
		Return(&models.Post{ID: 5, AuthorID: 101}, nil)

	req := httptest.NewRequest("PATCH", "/api/posts/5", bytes.NewReader([]byte(`{}`)))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "5")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(1)))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.UpdatePost(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestPostHandler_UpdatePost_NoText(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	postSvc, postRepo, _ := createTestPostService(ctrl)
	handler := NewPostHandler(mockUserSvc, postSvc, nil)

	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), int64(1)).
		Return(&models.Profile{ID: 100}, nil)

	postRepo.EXPECT().
		Get(gomock.Any(), int64(5)).
		Return(&models.Post{ID: 5, AuthorID: 100}, nil)

	req := httptest.NewRequest("PATCH", "/api/posts/5", bytes.NewReader([]byte(`{}`)))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "5")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(1)))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.UpdatePost(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostHandler_UpdatePost_UpdateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	postSvc, postRepo, _ := createTestPostService(ctrl)
	handler := NewPostHandler(mockUserSvc, postSvc, nil)

	newText := "Updated text"
	mockUserSvc.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), int64(1)).
		Return(&models.Profile{ID: 100}, nil)

	postRepo.EXPECT().
		Get(gomock.Any(), int64(5)).
		Return(&models.Post{ID: 5, AuthorID: 100}, nil)

	postRepo.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(models.Post{})).
		Return(errors.New("boom"))

	reqBody := PostCreationRequest{Text: &newText}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PATCH", "/api/posts/5", bytes.NewReader(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "5")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(1)))
	req = req.WithContext(logger.WithLogger(req.Context(), zap.NewNop()))

	w := httptest.NewRecorder()
	handler.UpdatePost(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
