package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/usecase"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPostHTTPPublicAndFeedEvents(t *testing.T) {
	t.Parallel()

	router := newPostRouter()

	rr := servePost(t, router, http.MethodGet, "/public/popular-posts", nil, 0)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"items"`)

	rr = servePost(t, router, http.MethodGet, "/posts/popular", nil, 5)
	require.Equal(t, http.StatusOK, rr.Code)

	rr = servePost(t, router, http.MethodPost, "/feed/events", map[string]any{
		"events": []map[string]any{{"postId": 1, "type": "view", "dwellMs": 100, "position": 2, "source": "feed"}},
	}, 5)
	require.Equal(t, http.StatusNoContent, rr.Code)

	tooMany := make([]map[string]any, 51)
	rr = servePost(t, router, http.MethodPost, "/feed/events", map[string]any{"events": tooMany}, 5)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestPostHTTPHelpers(t *testing.T) {
	t.Parallel()

	events := mapFeedEvents([]feedEventItem{
		{PostID: 1, Type: "view"},
		{PostID: 2, Type: "hide"},
		{PostID: 3, Type: "report"},
		{PostID: 4, Type: "unknown"},
	})
	require.Len(t, events, 3)

	w := httptest.NewRecorder()
	ids, ok := parseParentIDs(w, "1, 2,3")
	require.True(t, ok)
	require.Equal(t, []int64{1, 2, 3}, ids)
	w = httptest.NewRecorder()
	_, ok = parseParentIDs(w, "")
	require.False(t, ok)

	req := httptest.NewRequest(http.MethodGet, "/?limit=25&offset=3", nil)
	require.Equal(t, 25, parseBoundedQueryInt(req, "limit", 50, 1, 100))
	require.Equal(t, 3, parseBoundedQueryInt(req, "offset", 0, 0, 100))
	require.Equal(t, 50, parseBoundedQueryInt(httptest.NewRequest(http.MethodGet, "/?limit=bad", nil), "limit", 50, 1, 100))

	text := "<b>hello</b>"
	details := mapPostDetails(&usecase.PostDetails{
		ID: 1,
		Text: &text,
		Author: usecase.Author{ID: 10, FirstName: "<Neo>", LastName: "Anderson", Username: "neo", UserAccountID: 5},
		Media: []usecase.Media{{ID: 7, Name: "img", URL: "/m/7.png"}},
	})
	require.Equal(t, int64(1), details.ID)
	require.Equal(t, "&lt;b&gt;hello&lt;/b&gt;", *details.Text)

	now := time.Now()
	comment := mapComment(usecase.Comment{
		ID: 1, UID: uuid.New(), PostID: 2, Text: &text,
		Author: usecase.Author{ID: 10, FirstName: "Neo", Username: "neo"},
		CreatedAt: now, UpdatedAt: now,
	})
	require.Equal(t, "1", comment.ID)
	require.Equal(t, "2", comment.PostID)
	require.Equal(t, "", derefString(nil))
	require.Equal(t, "x", derefString(stringPtr("x")))
	require.Nil(t, int64PtrString(nil))
	id := int64(42)
	require.Equal(t, "42", *int64PtrString(&id))
	require.NotEmpty(t, popularTitles())
}

func newPostRouter() *chi.Mux {
	router := chi.NewRouter()
	New(usecase.New(repository.Store{}, nil, nil, nil)).RegisterRoutes(router, nil)
	return router
}

func servePost(t *testing.T, router *chi.Mux, method, path string, body any, userID int64) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if userID > 0 {
		req = req.WithContext(context.WithValue(req.Context(), "user_id", userID))
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func stringPtr(value string) *string {
	return &value
}
