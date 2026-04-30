package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/mocks"
	mock_service "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type validationResponse struct {
	Ok     bool              `json:"ok"`
	Errors map[string]string `json:"errors"`
}

func TestValidateRegisterStepOne_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	reqBody := `{"login": "test", "password1": "123", "password2": }` // broken JSON
	req := httptest.NewRequest(http.MethodPost, "/validate-step1", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.ValidateRegisterStepOne(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid request body")
}

func TestValidateRegisterStepOne_ShortPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	reqBody := `{"login": "validlogin", "password1": "123", "password2": "123"}`
	req := httptest.NewRequest(http.MethodPost, "/validate-step1", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.ValidateRegisterStepOne(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp validationResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.False(t, resp.Ok)
	assert.Equal(t, "Пароль слишком короткий (мин. 7 символов)", resp.Errors["password1"])
}

func TestValidateRegisterStepOne_LongPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	longPass := "thisisaverylongpasswordexceeding20chars"
	reqBody := map[string]string{
		"login":     "validlogin",
		"password1": longPass,
		"password2": longPass,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/validate-step1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.ValidateRegisterStepOne(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp validationResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.False(t, resp.Ok)
	assert.Equal(t, "Пароль может содержать максимум 20 символов", resp.Errors["password1"])
}

func TestValidateRegisterStepOne_PasswordMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	reqBody := `{"login": "validlogin", "password1": "password123", "password2": "password456"}`
	req := httptest.NewRequest(http.MethodPost, "/validate-step1", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.ValidateRegisterStepOne(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp validationResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.False(t, resp.Ok)
	assert.Equal(t, "Пароли не совпадают", resp.Errors["password2"])
	// password1 не должен содержать ошибку (пароль достаточной длины)
	assert.NotContains(t, resp.Errors, "password1")
}

func TestValidateRegisterStepOne_ShortLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	reqBody := `{"login": "short", "password1": "password123", "password2": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/validate-step1", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.ValidateRegisterStepOne(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp validationResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.False(t, resp.Ok)
	assert.Equal(t, "Логин слишком короткий (мин. 6 символов)", resp.Errors["login"])
}

func TestValidateRegisterStepOne_LongLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	reqBody := `{"login": "thisloginistoolong", "password1": "password123", "password2": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/validate-step1", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.ValidateRegisterStepOne(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp validationResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.False(t, resp.Ok)
	assert.Equal(t, "Логин может содержать максимум 12 символов", resp.Errors["login"])
}

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
	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	profileUid := uuid.New()
	expectedProfile := &models.Profile{
		Uid: profileUid,
	}

	expectedUserProfile := &models.UserProfile{
		Uid:           uuid.Nil,
		UserAccountID: 112,
	}
	userAccountUid := uuid.New()
	userAccountID := int64(43)
	expectedUserAccount := &models.UserAccount{
		ID:  userAccountID,
		Uid: userAccountUid,
	}

	sessionID := "sess123"
	expectedSession := &models.Session{
		SessionID: models.SessionID(sessionID),
		UserID:    expectedUserAccount.ID,
		ExpiredAt: time.Now().Add(24 * time.Hour),
	}

	mockAuthSvc.EXPECT().
		Register(gomock.Any(), "Ivan", "Petrov", "ivan123", "qwerty123", "24/02/2005", models.Gender("male")).
		Return(expectedProfile, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), int64(0)).
		Return(expectedUserProfile, nil)

	mockUserSvc.EXPECT().
		GetUserAccountByUserProfileID(gomock.Any(), int64(0)).
		Return(expectedUserAccount, nil)

	mockSessionSvc.EXPECT().
		Create(gomock.Any(), userAccountID).
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
	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Register_PasswordMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthService := mocks.NewMockAuthService(ctrl)
	mockSessionService := mocks.NewMockSessionService(ctrl)
	mockUserService := mocks.NewMockUserService(ctrl)
	// mediaService не используется в этом тесте, можно nil
	handler := NewAuthHandler(mockAuthService, mockSessionService, mockUserService, nil)

	requestBody := `{
		"firstName": "Ivan",
		"lastName": "Petrov",
		"login": "ivan123",
		"password1": "qwerty123",
		"password2": "different",
		"birthday": "24/02/2005",
		"gender": 1
	}`

	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBufferString(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	handler.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	// Хендлер возвращает "validation failed: ..." при ошибке валидации
	assert.Contains(t, response["error"], "validation failed")
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

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	mockAuthSvc.EXPECT().
		Register(gomock.Any(), "Ivan", "Petrov", "ivan123", "qwerty123", "24/02/2005", models.Gender("male")).
		Return(nil, errors.New("пользователь с таким login уже существует"))

	w := httptest.NewRecorder()
	handler.Register(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "login already exists")
}

func TestAuthHandler_Register_InvalidBirthday(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthService := mocks.NewMockAuthService(ctrl)
	mockSessionService := mocks.NewMockSessionService(ctrl)
	mockUserService := mocks.NewMockUserService(ctrl)
	handler := NewAuthHandler(mockAuthService, mockSessionService, mockUserService, nil)

	requestBody := `{
		"firstName": "Ivan",
		"lastName": "Petrov",
		"login": "ivan123",
		"password1": "qwerty123",
		"password2": "qwerty123",
		"birthday": "invalid-date",
		"gender": 1
	}`

	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBufferString(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	handler.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "validation failed")
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

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	mockAuthSvc.EXPECT().
		Register(gomock.Any(), "Ivan", "Petrov", "ivan123", "qwerty123", "24/02/2020", models.Gender("male")).
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

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	profileUid := uuid.New()
	expectedProfile := &models.Profile{Uid: profileUid}

	userProfileUid := uuid.New()
	expectedUserProfile := &models.UserProfile{Uid: userProfileUid, ID: 0}

	userAccountUid := uuid.New()
	userAccountID := int64(22)
	expectedUserAccount := &models.UserAccount{ID: userAccountID, Uid: userAccountUid}

	mockAuthSvc.EXPECT().
		Register(gomock.Any(), "Ivan", "Petrov", "ivan123", "qwerty123", "24/02/2005", models.Gender("male")).
		Return(expectedProfile, nil)

	mockUserSvc.EXPECT().
		GetUserProfileByProfileID(gomock.Any(), int64(0)).
		Return(expectedUserProfile, nil)

	mockUserSvc.EXPECT().
		GetUserAccountByUserProfileID(gomock.Any(), int64(0)).
		Return(expectedUserAccount, nil)

	mockSessionSvc.EXPECT().
		Create(gomock.Any(), userAccountID).
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

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	userID := uuid.New()
	userAccountID := int64(33)
	expectedUserAccount := &models.UserAccount{ID: userAccountID, Uid: userID, Username: "ivan123"}
	sessionID := "sess456"
	expectedSession := &models.Session{
		SessionID: models.SessionID(sessionID),
		UserID:    userAccountID,
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
		Create(gomock.Any(), userAccountID).
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

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

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

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	userUid := uuid.New()
	userAccountID := int64(23)
	expectedUser := &models.UserAccount{Uid: userUid, ID: userAccountID}

	expectedUserProfile := &models.UserProfile{ID: 1}

	mockAuthSvc.EXPECT().
		Login(gomock.Any(), "ivan123", "qwerty123").
		Return(expectedUser, nil)

	mockSessionSvc.EXPECT().
		Create(gomock.Any(), userAccountID).
		Return(nil, errors.New("session error"))

	mockUserSvc.EXPECT().
		GetUserProfileByUserID(gomock.Any(), userAccountID).
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

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

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

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

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

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

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

	mockAuthService := mocks.NewMockAuthService(ctrl)
	mockSessionService := mocks.NewMockSessionService(ctrl)
	mockUserService := mocks.NewMockUserService(ctrl)
	mockMediaService := mocks.NewMockMediaService(ctrl) // может быть nil, если не проверяем аватар

	handler := NewAuthHandler(mockAuthService, mockSessionService, mockUserService, mockMediaService)

	userID := int64(2)
	profileID := int64(1)

	expectedUserProfile := &models.UserProfile{
		ProfileID: profileID,
		FirstName: "Ivan",
		LastName:  "Petrov",
	}

	// Хендлер вызывает GetUserProfileByUserAccountID
	mockUserService.EXPECT().
		GetUserProfileByUserAccountID(gomock.Any(), userID).
		Return(expectedUserProfile, nil)

	// Может вызываться GetProfileByUserAccountID для аватара
	mockUserService.EXPECT().
		GetProfileByUserAccountID(gomock.Any(), userID).
		Return(&models.Profile{AvatarID: nil}, nil).AnyTimes()

	req := httptest.NewRequest("GET", "/api/me", nil)
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	mockLogger := zap.NewNop()
	ctx = logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.Me(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, strconv.FormatInt(userID, 10), response.ID)
	assert.Equal(t, "Ivan", response.FirstName)
	assert.Equal(t, "Petrov", response.LastName)
}

func TestAuthHandler_Me_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthSvc := mock_service.NewMockAuthService(ctrl)
	mockSessionSvc := mock_service.NewMockSessionService(ctrl)
	mockUserSvc := mock_service.NewMockUserService(ctrl)

	handler := NewAuthHandler(mockAuthSvc, mockSessionSvc, mockUserSvc)

	req := httptest.NewRequest("GET", "/api/auth/me", nil)

	mockLogger := zap.NewNop()
	ctx := logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

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

	userID := int64(44)
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	ctx := context.WithValue(req.Context(), "user_id", userID)
	req = req.WithContext(ctx)

	mockLogger := zap.NewNop()
	ctx = logger.WithLogger(req.Context(), mockLogger)
	req = req.WithContext(ctx)

	mockUserSvc.EXPECT().
		GetUserProfileByUserAccountID(ctx, userID).
		Return(nil, errors.New("not found"))

	w := httptest.NewRecorder()
	handler.Me(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not found")
}
