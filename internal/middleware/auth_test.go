package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	mock_service "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/mocks"
)

func TestAuthMiddleware_ValidSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessSvc := mock_service.NewMockSessionService(ctrl)
	middleware := AuthMiddleware(mockSessSvc)

	// Запрос с кукой
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "valid-session-id"})

	expectedUserID := uuid.New()
	mockSessSvc.EXPECT().
		Get(gomock.Any(), models.SessionID("valid-session-id")).
		Return(&models.Session{UserID: expectedUserID}, nil)

	// nextHandler проверит, что user_id добавлен в контекст
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id")
		assert.Equal(t, expectedUserID, userID)
		w.WriteHeader(http.StatusOK)
	})

	w := httptest.NewRecorder()
	middleware(nextHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_NoCookie(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSessSvc := mock_service.NewMockSessionService(ctrl)
	middleware := AuthMiddleware(mockSessSvc)

	req := httptest.NewRequest("GET", "/", nil)
	// nextHandler не должен быть вызван
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	w := httptest.NewRecorder()
	middleware(nextHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
