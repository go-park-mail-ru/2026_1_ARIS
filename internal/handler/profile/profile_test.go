package profile

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
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	mock_service "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
)

func TestProfileHandler_GetProfileMe_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)

	handler := NewProfileHandler(mockUserSvc, mockMediaSvc, mockSessionSvc)

	sessionID := models.SessionID("sess123")
	userID := int64(1)
	profileID := int64(100)

	// Мок сессии
	mockSessionSvc.EXPECT().
		Get(gomock.Any(), sessionID).
		Return(&models.Session{UserID: userID}, nil)

	// Мок профиля пользователя
	userProfile := &models.UserProfile{
		ID:           1,
		ProfileID:    profileID,
		FirstName:    "John",
		LastName:     "Doe",
		Gender:       models.Male,
		BirthdayDate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), userID).
		Return(userProfile, nil)

	// Мок buildProfileResponse
	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), profileID).
		Return(userProfile, nil)

	mockUserSvc.EXPECT().
		GetUserAccountByProfileID(gomock.Any(), profileID).
		Return(&models.UserAccount{Phone: ptrStr("+123"), Email: ptrStr("a@b.com")}, nil)

	mockUserSvc.EXPECT().
		GetProfileByUserProfileID(gomock.Any(), userProfile.ID).
		Return(&models.Profile{AvatarID: nil}, nil)

	req := httptest.NewRequest("GET", "/profile/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: string(sessionID)})
	w := httptest.NewRecorder()

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	handler.GetProfileMe(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp GetProfileMeResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "John", resp.FirstName)
	assert.Equal(t, "Doe", resp.LastName)
}

func TestProfileHandler_GetProfileMe_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewProfileHandler(nil, nil, nil)

	req := httptest.NewRequest("GET", "/profile/me", nil)
	w := httptest.NewRecorder()

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	handler.GetProfileMe(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestProfileHandler_GetProfileByID_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)

	handler := NewProfileHandler(mockUserSvc, mockMediaSvc, mockSessionSvc)

	profileID := int64(200)
	userProfile := &models.UserProfile{
		ID:           2,
		ProfileID:    profileID,
		FirstName:    "Jane",
		LastName:     "Smith",
		Gender:       models.Female,
		BirthdayDate: time.Time{},
	}

	// Моки для buildProfileResponse
	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), profileID).
		Return(userProfile, nil)

	mockUserSvc.EXPECT().
		GetUserAccountByProfileID(gomock.Any(), profileID).
		Return(&models.UserAccount{Phone: nil, Email: nil}, nil)

	mockUserSvc.EXPECT().
		GetProfileByUserProfileID(gomock.Any(), userProfile.ID).
		Return(&models.Profile{AvatarID: nil}, nil)

	req := httptest.NewRequest("GET", "/profile/200", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "200")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetProfileByID(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp GetProfileMeResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Jane", resp.FirstName)
}

func TestProfileHandler_GetProfileByID_InvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewProfileHandler(nil, nil, nil)

	req := httptest.NewRequest("GET", "/profile/invalid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.GetProfileByID(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_EditProfileMe_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)

	handler := NewProfileHandler(mockUserSvc, mockMediaSvc, mockSessionSvc)

	sessionID := models.SessionID("sess123")
	userID := int64(1)
	profileID := int64(100)
	userProfileID := int64(10)
	userAccountID := int64(20)

	// Мок сессии
	mockSessionSvc.EXPECT().
		Get(gomock.Any(), sessionID).
		Return(&models.Session{UserID: userID}, nil)

	// Мок профиля пользователя
	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.UserProfile{
			ID:        userProfileID,
			ProfileID: profileID,
		}, nil)

	// Мок аккаунта пользователя
	mockUserSvc.EXPECT().
		GetUserAccountByUserProfileID(gomock.Any(), userProfileID).
		Return(&models.UserAccount{ID: userAccountID}, nil)

	// Мок обновления
	mockUserSvc.EXPECT().
		UpdateMe(gomock.Any(), gomock.Any()).
		Return(nil)

	updateReq := dto.UpdateFullProfileRequestDTO{
		FirstName: ptrStr("NewName"),
	}
	body, _ := json.Marshal(updateReq)

	req := httptest.NewRequest("PATCH", "/profile/me/edit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: string(sessionID)})
	w := httptest.NewRecorder()

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	handler.EditProfileMe(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestProfileHandler_EditProfileMe_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)

	handler := NewProfileHandler(mockUserSvc, nil, mockSessionSvc)

	sessionID := models.SessionID("sess123")
	userID := int64(1)

	mockSessionSvc.EXPECT().
		Get(gomock.Any(), sessionID).
		Return(&models.Session{UserID: userID}, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.UserProfile{ID: 10, ProfileID: 100}, nil)

	mockUserSvc.EXPECT().
		GetUserAccountByUserProfileID(gomock.Any(), int64(10)).
		Return(&models.UserAccount{ID: 20}, nil)

	// Невалидный email вызовет ошибку валидации
	updateReq := map[string]interface{}{
		"email": "invalid-email",
	}
	body, _ := json.Marshal(updateReq)

	req := httptest.NewRequest("PATCH", "/profile/me/edit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "session_id", Value: string(sessionID)})
	w := httptest.NewRecorder()

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	handler.EditProfileMe(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_EditProfileMe_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewProfileHandler(nil, nil, nil)

	req := httptest.NewRequest("PATCH", "/profile/me/edit", nil)
	w := httptest.NewRecorder()

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	handler.EditProfileMe(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestProfileHandler_buildProfileResponse_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := &ProfileHandler{userService: mockUserSvc}

	profileID := int64(999)
	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), profileID).
		Return(nil, errors.New("not found"))

	req := httptest.NewRequest("GET", "/", nil)

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	_, status, msg := handler.buildProfileResponse(req, profileID)

	assert.Equal(t, http.StatusNotFound, status)
	assert.Equal(t, "profile not found", msg)
}
func TestProfileHandler_buildProfileResponse_WithAvatar(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	handler := &ProfileHandler{
		userService:  mockUserSvc,
		mediaService: mockMediaSvc,
	}

	profileID := int64(300)
	avatarID := int64(999)
	avatarLink := "http://avatar.png"

	userProfile := &models.UserProfile{
		ID:           3,
		ProfileID:    profileID,
		FirstName:    "Avatar",
		LastName:     "User",
		Gender:       models.Female,
		BirthdayDate: time.Now(),
	}

	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), profileID).
		Return(userProfile, nil)

	mockUserSvc.EXPECT().
		GetUserAccountByProfileID(gomock.Any(), profileID).
		Return(&models.UserAccount{Phone: nil, Email: nil}, nil)

	mockUserSvc.EXPECT().
		GetProfileByUserProfileID(gomock.Any(), userProfile.ID).
		Return(&models.Profile{AvatarID: &avatarID}, nil)

	mockMediaSvc.EXPECT().
		GetAvatarByID(gomock.Any(), &avatarID).
		Return(&models.Media{Link: avatarLink}, nil)

	req := httptest.NewRequest("GET", "/", nil)

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	resp, status, msg := handler.buildProfileResponse(req, profileID)

	assert.Equal(t, http.StatusOK, status)
	assert.Empty(t, msg)
	assert.NotNil(t, resp.ImageLink)
	assert.Equal(t, avatarLink, *resp.ImageLink)
}

func TestProfileHandler_buildProfileResponse_UserAccountError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := &ProfileHandler{userService: mockUserSvc}

	profileID := int64(400)

	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), profileID).
		Return(&models.UserProfile{ID: 4}, nil)

	mockUserSvc.EXPECT().
		GetUserAccountByProfileID(gomock.Any(), profileID).
		Return(nil, errors.New("db error"))

	req := httptest.NewRequest("GET", "/", nil)

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	_, status, msg := handler.buildProfileResponse(req, profileID)

	assert.Equal(t, http.StatusInternalServerError, status)
	assert.Equal(t, "internal server error", msg)
}

func TestProfileHandler_GetProfileMe_SessionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	handler := NewProfileHandler(nil, nil, mockSessionSvc)

	req := httptest.NewRequest("GET", "/profile/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "badsession"})

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	mockSessionSvc.EXPECT().
		Get(gomock.Any(), models.SessionID("badsession")).
		Return(nil, errors.New("not found"))

	w := httptest.NewRecorder()
	handler.GetProfileMe(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestProfileHandler_GetProfileMe_UserProfileError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	handler := NewProfileHandler(mockUserSvc, nil, mockSessionSvc)

	req := httptest.NewRequest("GET", "/profile/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess123"})

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	mockSessionSvc.EXPECT().
		Get(gomock.Any(), models.SessionID("sess123")).
		Return(&models.Session{UserID: 42}, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), int64(42)).
		Return(nil, errors.New("boom"))

	w := httptest.NewRecorder()
	handler.GetProfileMe(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestProfileHandler_GetProfileMe_BuildResponseFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	handler := NewProfileHandler(mockUserSvc, nil, mockSessionSvc)

	req := httptest.NewRequest("GET", "/profile/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess123"})

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	mockSessionSvc.EXPECT().
		Get(gomock.Any(), models.SessionID("sess123")).
		Return(&models.Session{UserID: 42}, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), int64(42)).
		Return(&models.UserProfile{ID: 5, ProfileID: 77}, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), int64(77)).
		Return(nil, errors.New("not found"))

	w := httptest.NewRecorder()
	handler.GetProfileMe(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProfileHandler_GetProfileByID_BuildResponseFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := NewProfileHandler(mockUserSvc, nil, nil)

	req := httptest.NewRequest("GET", "/profile/300", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "300")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), int64(300)).
		Return(nil, errors.New("not found"))

	w := httptest.NewRecorder()
	handler.GetProfileByID(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProfileHandler_buildProfileResponse_NoAvatarAndBirthday(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	handler := &ProfileHandler{
		userService:  mockUserSvc,
		mediaService: mockMediaSvc,
	}

	profileID := int64(301)
	bio := "<b>bio</b>"
	town := "<town>"
	institution := "<uni>"

	userProfile := &models.UserProfile{
		ID:           33,
		ProfileID:    profileID,
		FirstName:    "<Jane>",
		LastName:     "<Smith>",
		Bio:          &bio,
		Town:         &town,
		Institution:  &institution,
		Gender:       models.Female,
		BirthdayDate: time.Date(2000, 2, 3, 0, 0, 0, 0, time.UTC),
	}

	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), profileID).
		Return(userProfile, nil)

	mockUserSvc.EXPECT().
		GetUserAccountByProfileID(gomock.Any(), profileID).
		Return(&models.UserAccount{}, nil)

	mockUserSvc.EXPECT().
		GetProfileByUserProfileID(gomock.Any(), userProfile.ID).
		Return(nil, errors.New("missing profile"))

	req := httptest.NewRequest("GET", "/", nil)
	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	resp, status, msg := handler.buildProfileResponse(req, profileID)

	assert.Equal(t, http.StatusOK, status)
	assert.Empty(t, msg)
	assert.Equal(t, "&lt;Jane&gt;", resp.FirstName)
	assert.Equal(t, "&lt;Smith&gt;", resp.LastName)
	assert.Equal(t, "2000-02-03", resp.BirthdayDate)
	assert.Nil(t, resp.ImageLink)
	assert.NotNil(t, resp.Bio)
	assert.Equal(t, "<b>bio</b>", *resp.Bio)
	assert.Len(t, resp.Education, 1)
	assert.NotNil(t, resp.Education[0].Institution)
	assert.Equal(t, "<uni>", *resp.Education[0].Institution)
}

func TestProfileHandler_EditProfileMe_InvalidSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	handler := NewProfileHandler(nil, nil, mockSessionSvc)

	req := httptest.NewRequest("PATCH", "/profile/me/edit", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "bad"})
	w := httptest.NewRecorder()

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	mockSessionSvc.EXPECT().
		Get(gomock.Any(), models.SessionID("bad")).
		Return(nil, errors.New("bad session"))

	handler.EditProfileMe(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestProfileHandler_EditProfileMe_GetUserProfileError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	handler := NewProfileHandler(mockUserSvc, nil, mockSessionSvc)

	req := httptest.NewRequest("PATCH", "/profile/me/edit", bytes.NewReader([]byte(`{}`)))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess123"})
	w := httptest.NewRecorder()

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	mockSessionSvc.EXPECT().
		Get(gomock.Any(), models.SessionID("sess123")).
		Return(&models.Session{UserID: 7}, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), int64(7)).
		Return(nil, errors.New("boom"))

	handler.EditProfileMe(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestProfileHandler_EditProfileMe_GetUserAccountError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	handler := NewProfileHandler(mockUserSvc, nil, mockSessionSvc)

	req := httptest.NewRequest("PATCH", "/profile/me/edit", bytes.NewReader([]byte(`{}`)))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess123"})
	w := httptest.NewRecorder()

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	mockSessionSvc.EXPECT().
		Get(gomock.Any(), models.SessionID("sess123")).
		Return(&models.Session{UserID: 7}, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), int64(7)).
		Return(&models.UserProfile{ID: 70, ProfileID: 700}, nil)

	mockUserSvc.EXPECT().
		GetUserAccountByUserProfileID(gomock.Any(), int64(70)).
		Return(nil, errors.New("boom"))

	handler.EditProfileMe(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestProfileHandler_EditProfileMe_InvalidBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	handler := NewProfileHandler(mockUserSvc, nil, mockSessionSvc)

	req := httptest.NewRequest("PATCH", "/profile/me/edit", bytes.NewReader([]byte(`{`)))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess123"})
	w := httptest.NewRecorder()

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	mockSessionSvc.EXPECT().
		Get(gomock.Any(), models.SessionID("sess123")).
		Return(&models.Session{UserID: 7}, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), int64(7)).
		Return(&models.UserProfile{ID: 70, ProfileID: 700}, nil)

	mockUserSvc.EXPECT().
		GetUserAccountByUserProfileID(gomock.Any(), int64(70)).
		Return(&models.UserAccount{ID: 88}, nil)

	handler.EditProfileMe(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProfileHandler_EditProfileMe_UpdateError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	handler := NewProfileHandler(mockUserSvc, nil, mockSessionSvc)

	req := httptest.NewRequest("PATCH", "/profile/me/edit", bytes.NewReader([]byte(`{}`)))
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess123"})
	w := httptest.NewRecorder()

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	mockSessionSvc.EXPECT().
		Get(gomock.Any(), models.SessionID("sess123")).
		Return(&models.Session{UserID: 7}, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), int64(7)).
		Return(&models.UserProfile{ID: 70, ProfileID: 700}, nil)

	mockUserSvc.EXPECT().
		GetUserAccountByUserProfileID(gomock.Any(), int64(70)).
		Return(&models.UserAccount{ID: 88}, nil)

	mockUserSvc.EXPECT().
		UpdateMe(gomock.Any(), gomock.Any()).
		Return(errors.New("update failed"))

	handler.EditProfileMe(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestNewProfileHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)

	handler := NewProfileHandler(mockUserSvc, mockMediaSvc, mockSessionSvc)

	assert.NotNil(t, handler)
}

func ptrStr(s string) *string { return &s }
