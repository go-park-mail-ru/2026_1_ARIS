package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/search/internal/usecase"
	"go.uber.org/zap"
)

func TestHandlerSearchInvalidInput(t *testing.T) {
	handler := New(usecase.New(nil, nil), zap.NewNop())
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/search?limit=bad", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMapResultEscapesFields(t *testing.T) {
	avatarID := int64(1)
	url := "https://cdn/avatar"
	bio := "<bio>"
	coverID := int64(2)
	communityID := int64(3)

	resp := mapResult(&usecase.Result{
		Users: []usecase.UserResult{{
			ProfileID: 10, UserAccountID: 20, Username: "<ann>", FirstName: "<Ann>", LastName: "User", AvatarID: &avatarID, AvatarURL: &url,
		}},
		Communities: []usecase.CommunityResult{{
			ID: 1, ProfileID: 30, Username: "<community>", Title: "<Title>", Bio: &bio, Type: "<public>", AvatarID: &avatarID, AvatarURL: &url, CoverMediaID: &coverID, CoverURL: &url,
		}},
		Posts: []usecase.PostResult{{
			ID: 2, Text: "<post>", AuthorID: 40, AuthorProfileID: 50, AuthorUsername: "<bob>", AuthorFirstName: "<Bob>", AuthorLastName: "Writer", AuthorAvatarID: &avatarID, AuthorAvatarURL: &url, CommunityID: &communityID, CreatedAt: time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC),
		}},
	})

	if resp.Users[0].Username != "&lt;ann&gt;" || resp.Communities[0].Title != "&lt;Title&gt;" || resp.Posts[0].Text != "&lt;post&gt;" {
		t.Fatalf("expected escaped fields, got %+v", resp)
	}
	if resp.Communities[0].Bio == nil || *resp.Communities[0].Bio != "&lt;bio&gt;" {
		t.Fatalf("expected escaped bio, got %+v", resp.Communities[0].Bio)
	}
}

func TestParseLimitAndServiceError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/search?limit=17", nil)
	if got := parseLimit(req, "limit", 10); got != 17 {
		t.Fatalf("parseLimit() = %d", got)
	}
	req = httptest.NewRequest(http.MethodGet, "/search?limit=bad", nil)
	if got := parseLimit(req, "limit", 10); got != 10 {
		t.Fatalf("parseLimit fallback = %d", got)
	}

	rec := httptest.NewRecorder()
	writeServiceError(rec, usecase.ErrInvalidInput)
	if rec.Code != http.StatusBadRequest || !bytes.Contains(rec.Body.Bytes(), []byte("invalid input")) {
		t.Fatalf("unexpected invalid input response: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	writeServiceError(rec, errUnexpected{})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

type errUnexpected struct{}

func (errUnexpected) Error() string { return "unexpected" }
