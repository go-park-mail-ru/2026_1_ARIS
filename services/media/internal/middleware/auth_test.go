package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	authpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth"
	authmock "github.com/go-park-mail-ru/2026_1_ARIS/proto/auth/mock"
	"github.com/golang/mock/gomock"
)

func TestAuthMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := authmock.NewMockAuthServiceClient(ctrl)
	middleware := AuthMiddleware(client)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next should not be called without cookie")
	})).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized without cookie, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "bad"})
	rec = httptest.NewRecorder()
	client.EXPECT().ValidateSession(gomock.Any(), gomock.Any()).Return(nil, errors.New("invalid"))
	middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next should not be called for invalid session")
	})).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized for invalid session, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "ok"})
	rec = httptest.NewRecorder()
	client.EXPECT().ValidateSession(gomock.Any(), gomock.Any()).Return(&authpb.ValidateSessionResponse{UserAccountId: 42}, nil)
	middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Context().Value("user_id"); got != int64(42) {
			t.Fatalf("unexpected user_id in context: %#v", got)
		}
		w.WriteHeader(http.StatusAccepted)
	})).ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected next status, got %d", rec.Code)
	}
}
