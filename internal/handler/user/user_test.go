package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	mock_service "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
)

// Вспомогательная функция для создания контекста с user_id
func contextWithUserID(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func TestUserHandler_GetSuggestedUsers_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	mockSettingsSvc := mock_service.NewMockUserSettingsService(ctrl)

	handler := NewUserHandler(mockUserSvc, mockMediaSvc, mockSettingsSvc)

	userID := int64(1)
	profileID1 := int64(100)
	profileID2 := int64(200)

	// Мок получения рекомендованных пользователей
	profiles := []models.Profile{
		{ID: profileID1, AvatarID: nil},
		{ID: profileID2, AvatarID: ptrInt64(5)},
	}
	mockUserSvc.EXPECT().
		GetSuggestedUsers(gomock.Any(), userID).
		Return(profiles, nil)

	// Для первого профиля: получение UserProfile и UserAccount
	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), profileID1).
		Return(&models.UserProfile{ID: 1, FirstName: "John", LastName: "Doe"}, nil)
	mockUserSvc.EXPECT().
		GetUserAccountByUserProfileID(gomock.Any(), int64(1)).
		Return(&models.UserAccount{Username: "johndoe"}, nil)

	// Для второго профиля: аватар, профиль, аккаунт
	mockMediaSvc.EXPECT().
		GetAvatarByID(gomock.Any(), ptrInt64(5)).
		Return(&models.Media{Link: "http://avatar.png"}, nil)
	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), profileID2).
		Return(&models.UserProfile{ID: 2, FirstName: "Jane", LastName: "Smith"}, nil)
	mockUserSvc.EXPECT().
		GetUserAccountByUserProfileID(gomock.Any(), int64(2)).
		Return(&models.UserAccount{Username: "janesmith"}, nil)

	req := httptest.NewRequest("GET", "/users/suggested", nil)
	req = req.WithContext(contextWithUserID(userID))
	w := httptest.NewRecorder()

	handler.GetSuggestedUsers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp suggestedUsersResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, "100", resp.Items[0].Id)
	assert.Equal(t, "John", resp.Items[0].FirstName)
	assert.Equal(t, "johndoe", resp.Items[0].Username)
	assert.Equal(t, "200", resp.Items[1].Id)
	assert.Equal(t, "Jane", resp.Items[1].FirstName)
	assert.Equal(t, "http://avatar.png", resp.Items[1].AvatarLink)
}

func TestUserHandler_GetSuggestedUsers_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewUserHandler(nil, nil, nil)

	req := httptest.NewRequest("GET", "/users/suggested", nil)
	// Нет user_id в контексте
	w := httptest.NewRecorder()

	handler.GetSuggestedUsers(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserHandler_GetSuggestedUsers_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := NewUserHandler(mockUserSvc, nil, nil)

	userID := int64(1)
	mockUserSvc.EXPECT().
		GetSuggestedUsers(gomock.Any(), userID).
		Return(nil, errors.New("db error"))

	req := httptest.NewRequest("GET", "/users/suggested", nil)
	req = req.WithContext(contextWithUserID(userID))
	w := httptest.NewRecorder()

	handler.GetSuggestedUsers(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUserHandler_GetPublicPopularUsers_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	mockSettingsSvc := mock_service.NewMockUserSettingsService(ctrl)

	handler := NewUserHandler(mockUserSvc, mockMediaSvc, mockSettingsSvc)

	profiles := []models.Profile{
		{ID: 300, AvatarID: nil},
	}
	mockUserSvc.EXPECT().
		GetPublicPopularUsers(gomock.Any()).
		Return(profiles, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), int64(300)).
		Return(&models.UserProfile{ID: 3, FirstName: "Public", LastName: "User"}, nil)
	mockUserSvc.EXPECT().
		GetUserAccountByUserProfileID(gomock.Any(), int64(3)).
		Return(&models.UserAccount{Username: "publicuser"}, nil)

	req := httptest.NewRequest("GET", "/public/popular-users", nil)
	w := httptest.NewRecorder()

	handler.GetPublicPopularUsers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp suggestedUsersResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "300", resp.Items[0].Id)
	assert.Equal(t, "publicuser", resp.Items[0].Username)
}

func TestUserHandler_GetPublicPopularUsers_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := NewUserHandler(mockUserSvc, nil, nil)

	mockUserSvc.EXPECT().
		GetPublicPopularUsers(gomock.Any()).
		Return(nil, errors.New("service error"))

	req := httptest.NewRequest("GET", "/public/popular-users", nil)
	w := httptest.NewRecorder()

	handler.GetPublicPopularUsers(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUserHandler_GetLatestEvents_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	mockSettingsSvc := mock_service.NewMockUserSettingsService(ctrl)

	handler := NewUserHandler(mockUserSvc, mockMediaSvc, mockSettingsSvc)

	events := []user.LatestEvent{
		{
			Profile: models.Profile{ID: 400, AvatarID: nil},
			Type:    1,
		},
	}
	mockUserSvc.EXPECT().
		GetLatestEvents(gomock.Any()).
		Return(events, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), int64(400)).
		Return(&models.UserProfile{ID: 4, FirstName: "Event", LastName: "User"}, nil)
	mockUserSvc.EXPECT().
		GetUserAccountByUserProfileID(gomock.Any(), int64(4)).
		Return(&models.UserAccount{Username: "eventuser"}, nil)

	req := httptest.NewRequest("GET", "/users/latest-events", nil)
	w := httptest.NewRecorder()

	handler.GetLatestEvents(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp latestEventsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "400", resp.Items[0].Id)
	assert.Equal(t, 1, resp.Items[0].Type)
}

func TestUserHandler_GetLatestEvents_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	handler := NewUserHandler(mockUserSvc, nil, nil)

	mockUserSvc.EXPECT().
		GetLatestEvents(gomock.Any()).
		Return(nil, errors.New("service error"))

	req := httptest.NewRequest("GET", "/users/latest-events", nil)
	w := httptest.NewRecorder()

	handler.GetLatestEvents(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUserHandler_SetSettings_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	mockSettingsSvc := mock_service.NewMockUserSettingsService(ctrl)

	handler := NewUserHandler(mockUserSvc, mockMediaSvc, mockSettingsSvc)

	userID := int64(1)
	lang := models.LanguageRU
	theme := models.ThemeDark
	updateReq := dto.UserSettingsUpdate{
		Language: &lang,
		Theme:    &theme,
	}
	expectedSettings := &models.UserSettings{
		UserAccountID: userID,
		Language:      lang,
		Theme:         theme,
	}

	mockSettingsSvc.EXPECT().
		Update(gomock.Any(), userID, updateReq).
		Return(expectedSettings, nil)

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("POST", "/api/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithUserID(userID))
	w := httptest.NewRecorder()

	handler.SetSettings(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.UserSettings
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, lang, resp.Language)
	assert.Equal(t, theme, resp.Theme)
}

func TestUserHandler_SetSettings_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewUserHandler(nil, nil, nil)

	userID := int64(1)
	req := httptest.NewRequest("POST", "/api/settings", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithUserID(userID))
	w := httptest.NewRecorder()

	handler.SetSettings(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_SetSettings_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewUserHandler(nil, nil, nil)

	userID := int64(1)
	// Передаём невалидные значения (пусть Language будет некорректным)
	invalidReq := map[string]string{
		"language": "INVALID",
	}
	body, _ := json.Marshal(invalidReq)
	req := httptest.NewRequest("POST", "/api/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithUserID(userID))
	w := httptest.NewRecorder()

	handler.SetSettings(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_SetSettings_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	mockSettingsSvc := mock_service.NewMockUserSettingsService(ctrl)

	handler := NewUserHandler(mockUserSvc, mockMediaSvc, mockSettingsSvc)

	userID := int64(1)
	lang := models.LanguageEN
	updateReq := dto.UserSettingsUpdate{
		Language: &lang,
	}

	mockSettingsSvc.EXPECT().
		Update(gomock.Any(), userID, updateReq).
		Return(nil, errors.New("db error"))

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("POST", "/api/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithUserID(userID))
	w := httptest.NewRecorder()

	handler.SetSettings(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUserHandler_GetSettings_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	mockSettingsSvc := mock_service.NewMockUserSettingsService(ctrl)

	handler := NewUserHandler(mockUserSvc, mockMediaSvc, mockSettingsSvc)

	userID := int64(1)
	expectedSettings := &models.UserSettings{
		UserAccountID: userID,
		Language:      models.LanguageRU,
		Theme:         models.ThemeLight,
	}

	mockSettingsSvc.EXPECT().
		GetByUserID(gomock.Any(), userID).
		Return(expectedSettings, nil)

	req := httptest.NewRequest("GET", "/api/settings", nil)
	req = req.WithContext(contextWithUserID(userID))
	w := httptest.NewRecorder()

	handler.GetSettings(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.UserSettings
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, models.LanguageRU, resp.Language)
}

func TestUserHandler_GetSettings_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	handler := NewUserHandler(nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/settings", nil)
	// Нет user_id в контексте
	w := httptest.NewRecorder()

	handler.GetSettings(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserHandler_GetSettings_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	mockSettingsSvc := mock_service.NewMockUserSettingsService(ctrl)

	handler := NewUserHandler(mockUserSvc, mockMediaSvc, mockSettingsSvc)

	userID := int64(1)
	mockSettingsSvc.EXPECT().
		GetByUserID(gomock.Any(), userID).
		Return(nil, errors.New("db error"))

	req := httptest.NewRequest("GET", "/api/settings", nil)
	req = req.WithContext(contextWithUserID(userID))
	w := httptest.NewRecorder()

	handler.GetSettings(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
func TestUserHandler_GetSuggestedUsers_ProfileError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserSvc := mock_service.NewMockUserService(ctrl)
	mockMediaSvc := mock_service.NewMockMediaService(ctrl)
	mockSettingsSvc := mock_service.NewMockUserSettingsService(ctrl)

	handler := NewUserHandler(mockUserSvc, mockMediaSvc, mockSettingsSvc)

	userID := int64(1)
	profileID := int64(100)

	profiles := []models.Profile{
		{ID: profileID, AvatarID: nil},
	}
	mockUserSvc.EXPECT().
		GetSuggestedUsers(gomock.Any(), userID).
		Return(profiles, nil)

	// Ошибка при получении UserProfile
	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), profileID).
		Return(nil, errors.New("not found"))

	req := httptest.NewRequest("GET", "/users/suggested", nil)
	req = req.WithContext(contextWithUserID(userID))
	w := httptest.NewRecorder()

	handler.GetSuggestedUsers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp suggestedUsersResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Empty(t, resp.Items) // пропущен из-за ошибки
}

// Вспомогательная функция для указателя на int64
func ptrInt64(i int64) *int64 {
	return &i
}
