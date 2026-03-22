package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	mock_service "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/mocks"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuthHandler_Register_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	reqBody := RegisterRequest{
		FirstName: "Ivan",
		LastName:  "Petrov",
		Birthday:  "24/02/2005",
		Login:     "ivan123",
		Gender:    1,
		Password1: "qwerty123",
		Password2: "qwerty123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	profileUid := uuid.New()
	expectedProfile := &models.Profile{
		Uid: profileUid,
	}

	expectedUserProfile := &models.UserProfile{
		Uid:           uuid.Nil,
		UserAccountID: 112,
	}
	sessionID := "sess123"
	expectedSession := &models.Session{
		SessionID: models.SessionID(sessionID),
		UserID:    expectedUserProfile.Uid,
		ExpiredAt: time.Now().Add(24 * time.Hour),
	}

	userAccountUid := uuid.New()
	expectedUserAccount := &models.UserAccount{
		ID:  0,
		Uid: userAccountUid,
	}

	mockAuthSvc.EXPECT().
		Register(gomock.Any(), "Ivan", "Petrov", "ivan123", "qwerty123", "24/02/2005", models.Gender(0)).
		Return(expectedProfile, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), int64(0)).
		Return(expectedUserProfile, nil)

	mockUserSvc.EXPECT().
		GetUserAccountByUserProfileID(gomock.Any(), int64(0)).
		Return(expectedUserAccount, nil)

	mockSessionSvc.EXPECT().
		Create(gomock.Any(), userAccountUid).
		Return(expectedSession, nil)

	// Выполнение
	w := httptest.NewRecorder()
	handler.Register(w, req)

	// Проверки
	assert.Equal(t, http.StatusCreated, w.Code, w)

	// Проверка куки
	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "session_id" {
			assert.Equal(t, sessionID, c.Value)
			assert.True(t, c.HttpOnly)
			assert.Equal(t, "/", c.Path)
			found = true
			break
		}
	}
	assert.True(t, found, "cookie session_id not set")

	// Проверка тела ответа
	var resp models.Profile
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, expectedProfile.ID, resp.ID)
	//assert.Equal(t, expectedProfile.Username, resp.Username)
}

func TestAuthHandler_Register_ValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	reqBody := RegisterRequest{
		FirstName: "Ivan",
		LastName:  "Petrov",
		Birthday:  "24/02/2005",
		Login:     "ivan123",
		Password1: "123",
		Password2: "123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Register_PasswordMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	reqBody := RegisterRequest{
		FirstName: "Ivan",
		LastName:  "Petrov",
		Birthday:  "24/02/2005",
		Gender:    1,
		Login:     "ivan123",
		Password1: "qwerty123",
		Password2: "qwerty456",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "passwords do not match")
}

func TestAuthHandler_Register_LoginAlreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	reqBody := RegisterRequest{
		FirstName: "Ivan",
		LastName:  "Petrov",
		Birthday:  "24/02/2005",
		Gender:    1,
		Login:     "ivan123",
		Password1: "qwerty123",
		Password2: "qwerty123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	mockAuthSvc.EXPECT().
		Register(gomock.Any(), "Ivan", "Petrov", "ivan123", "qwerty123", "24/02/2005", models.Gender(0)).
		Return(nil, errors.New("пользователь с таким login уже существует"))

	w := httptest.NewRecorder()
	handler.Register(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "login already exists")
}

func TestAuthHandler_Register_InvalidBirthday(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	reqBody := RegisterRequest{
		FirstName: "Ivan",
		LastName:  "Petrov",
		Birthday:  "2005-02-24", // неправильный формат
		Gender:    1,
		Login:     "ivan123",
		Password1: "qwerty123",
		Password2: "qwerty123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	mockAuthSvc.EXPECT().
		Register(gomock.Any(), "Ivan", "Petrov", "ivan123", "qwerty123", "2005-02-24", models.Gender(0)).
		Return(nil, errors.New("invalid birthday date"))

	w := httptest.NewRecorder()
	handler.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid birthday date")
}

func TestAuthHandler_Register_TooYoung(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	reqBody := RegisterRequest{
		FirstName: "Ivan",
		LastName:  "Petrov",
		Birthday:  "24/02/2020", // моложе 12
		Gender:    1,
		Login:     "ivan123",
		Password1: "qwerty123",
		Password2: "qwerty123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	mockAuthSvc.EXPECT().
		Register(gomock.Any(), "Ivan", "Petrov", "ivan123", "qwerty123", "24/02/2020", models.Gender(0)).
		Return(nil, errors.New("you are too young, buddy"))

	w := httptest.NewRecorder()
	handler.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "user is too young")
}

func TestAuthHandler_Register_SessionCreationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	reqBody := RegisterRequest{
		FirstName: "Ivan",
		LastName:  "Petrov",
		Birthday:  "24/02/2005",
		Gender:    1,
		Login:     "ivan123",
		Password1: "qwerty123",
		Password2: "qwerty123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	profileUid := uuid.New()
	expectedProfile := &models.Profile{Uid: profileUid}

	userProfileUid := uuid.New()
	expectedUserProfile := &models.UserProfile{Uid: userProfileUid, ID: 0}

	userAccountUid := uuid.New()
	expectedUserAccount := &models.UserAccount{ID: 22, Uid: userAccountUid}

	mockAuthSvc.EXPECT().
		Register(gomock.Any(), "Ivan", "Petrov", "ivan123", "qwerty123", "24/02/2005", models.Gender(0)).
		Return(expectedProfile, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), int64(0)).
		Return(expectedUserProfile, nil)

	mockUserSvc.EXPECT().
		GetUserAccountByUserProfileID(gomock.Any(), int64(0)).
		Return(expectedUserAccount, nil)

	mockSessionSvc.EXPECT().
		Create(gomock.Any(), userAccountUid).
		Return(nil, errors.New("session error"))

	w := httptest.NewRecorder()
	handler.Register(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to create session")
}

func TestAuthHandler_Login_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	reqBody := LoginRequest{
		Login:    "ivan123",
		Password: "qwerty123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	userID := uuid.New()
	userAccountID := int64(33)
	expectedUserAccount := &models.UserAccount{ID: userAccountID, Uid: userID, Username: "ivan123"}
	sessionID := "sess456"
	expectedSession := &models.Session{
		SessionID: models.SessionID(sessionID),
		UserID:    userID,
		ExpiredAt: time.Now().Add(24 * time.Hour),
	}

	var userProfileID int64 = 44

	expectedUserProfile := &models.UserProfile{
		ID:            userProfileID,
		Uid:           uuid.New(),
		UserAccountID: userAccountID,
		FirstName:     "Ivan",
		LastName:      "Petrov",
		CreatedAt:     time.Now(),
	}

	mockAuthSvc.EXPECT().
		Login(gomock.Any(), "ivan123", "qwerty123").
		Return(expectedUserAccount, nil)

	mockSessionSvc.EXPECT().
		Create(gomock.Any(), userID).
		Return(expectedSession, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByUserID(gomock.Any(), userAccountID).
		Return(expectedUserProfile, nil)

	w := httptest.NewRecorder()
	handler.Login(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "session_id" {
			assert.Equal(t, sessionID, c.Value)
			assert.True(t, c.HttpOnly)
			assert.Equal(t, "/", c.Path)
			found = true
			break
		}
	}
	assert.True(t, found, "cookie session_id not set")

	var resp LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

}
func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	reqBody := LoginRequest{
		Login:    "ivan123",
		Password: "wrong",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	mockAuthSvc.EXPECT().
		Login(gomock.Any(), "ivan123", "wrong").
		Return(nil, errors.New("invalid credentials"))

	w := httptest.NewRecorder()
	handler.Login(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid credentials")
}

func TestAuthHandler_Login_SessionCreationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	reqBody := LoginRequest{
		Login:    "ivan123",
		Password: "qwerty123",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	userID := uuid.New()
	expectedUser := &models.UserAccount{Uid: userID, ID: 0}

	expectedUserProfile := &models.UserProfile{ID: 1}

	mockAuthSvc.EXPECT().
		Login(gomock.Any(), "ivan123", "qwerty123").
		Return(expectedUser, nil)

	mockSessionSvc.EXPECT().
		Create(gomock.Any(), userID).
		Return(nil, errors.New("session error"))

	mockUserSvc.EXPECT().
		GetUserProfileByUserID(gomock.Any(), int64(0)).
		Return(expectedUserProfile, nil)

	w := httptest.NewRecorder()
	handler.Login(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "failed to create session")
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	// Запрос с кукой
	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: "sess123",
	})

	mockSessionSvc.EXPECT().
		Delete(gomock.Any(), models.SessionID("sess123")).
		Return(nil)

	w := httptest.NewRecorder()
	handler.Logout(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "session_id" {
			assert.Equal(t, "", c.Value)
			assert.True(t, c.MaxAge < 0) // MaxAge -1 означает удаление
			found = true
			break
		}
	}
	assert.True(t, found, "cookie not cleared")

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "successfully logged out", resp["message"])
}

func TestAuthHandler_Logout_NoCookie(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)

	w := httptest.NewRecorder()
	handler.Logout(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "already logged out", resp["message"])
}

func TestAuthHandler_Logout_DeleteError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: "sess123",
	})

	mockSessionSvc.EXPECT().
		Delete(gomock.Any(), models.SessionID("sess123")).
		Return(errors.New("delete error"))

	w := httptest.NewRecorder()
	handler.Logout(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Кука должна быть сброшена
	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "session_id" {
			assert.Equal(t, "", c.Value)
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestAuthHandler_Me_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	//var expectedID int64 = 1
	userUid := uuid.New()
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	ctx := context.WithValue(req.Context(), "user_id", userUid)
	req = req.WithContext(ctx)

	expectedUserProfile := &models.UserProfile{
		ID:            2,
		Uid:           uuid.New(),
		UserAccountID: 1,
		FirstName:     "Ivan",
		LastName:      "Petrov",
	}

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountUid(ctx, userUid).
		Return(expectedUserProfile, nil)

	w := httptest.NewRecorder()
	handler.Me(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.UserProfile
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, expectedUserProfile.ID, resp.ID)
	assert.Equal(t, expectedUserProfile.FirstName, resp.FirstName)
}

func TestAuthHandler_Me_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	req := httptest.NewRequest("GET", "/api/auth/me", nil)

	w := httptest.NewRecorder()
	handler.Me(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unauthorized")
}

func TestAuthHandler_Me_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	userID := uuid.New()
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountUid(ctx, userID).
		Return(nil, errors.New("not found"))

	w := httptest.NewRecorder()
	handler.Me(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
}
