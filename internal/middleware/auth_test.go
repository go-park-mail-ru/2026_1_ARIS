package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type fakeSessionService struct {
	session *models.Session
	err     error
}

func (f fakeSessionService) Create(context.Context, int64) (*models.Session, error) {
	return nil, errors.New("not implemented")
}

func (f fakeSessionService) Get(context.Context, models.SessionID) (*models.Session, error) {
	return f.session, f.err
}

func (f fakeSessionService) Delete(context.Context, models.SessionID) error {
	return errors.New("not implemented")
}

func TestAuthMiddleware_ValidSession(t *testing.T) {
	expectedUserID := int64(12)
	middleware := AuthMiddleware(fakeSessionService{session: &models.Session{UserID: expectedUserID}})

	// Запрос с кукой
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "valid-session-id"})

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
	middleware := AuthMiddleware(fakeSessionService{})

	req := httptest.NewRequest("GET", "/", nil)
	// nextHandler не должен быть вызван
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	})

	w := httptest.NewRecorder()
	middleware(nextHandler).ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
