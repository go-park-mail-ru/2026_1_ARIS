package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/post/service"
	"github.com/stretchr/testify/require"
)

func TestPostMappingHelpers(t *testing.T) {
	text := "<hello>"
	avatar := "https://cdn.test/a.png"
	communityID := int64(3)
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	details := &service.PostDetails{
		ID: 1, AuthorID: 2, CommunityID: &communityID, Text: &text, CreatedAt: now, UpdatedAt: now, Likes: 7, IsLiked: true,
		Author: service.Author{ID: 2, FirstName: "<Neo>", LastName: "A", Username: "neo", UserAccountID: 10, AvatarURL: &avatar},
		Media:  []service.Media{{ID: 5, UID: "uid", MimeType: "image/png", URL: "https://cdn.test/m.png"}},
	}

	resp := mapPostDetails(details)
	require.Equal(t, int64(1), resp.ID)
	require.Equal(t, "&lt;hello&gt;", *resp.Text)
	require.Equal(t, "&lt;Neo&gt;", resp.Author.FirstName)
	require.Len(t, resp.Media, 1)

	list := mapPostList([]service.PostDetails{*details})
	require.Len(t, list, 1)
	require.Equal(t, "&lt;hello&gt;", list[0].Text)
	require.NotNil(t, list[0].UpdatedAt)

	feed := mapFeed(service.FeedResult{
		Posts: []service.FeedPost{{
			ID: 1, Text: "body", Author: details.Author, CreatedAt: now, Likes: 1, Comments: 2, Reposts: 3,
			Medias: []service.Media{{ID: 5, UID: "uid", MimeType: "image/png", URL: "https://cdn.test/m.png"}},
		}},
		Cursor: "cursor", HasMore: true,
	})
	require.Len(t, feed.Items, 1)
	require.Equal(t, "2", feed.Items[0].Author.Id)
	require.Equal(t, avatar, feed.Items[0].Author.AvatarLink)
	require.Equal(t, "cursor", feed.NextCursor)

	media := []dto.MediaRequestData{{MediaID: 5, MediaURL: "url"}}
	input := createInput(PostCreationRequest{Text: &text, Media: &media, CommunityID: &communityID})
	require.Equal(t, text, *input.Text)
	require.Equal(t, media, input.Media)
	require.Equal(t, &communityID, input.CommunityID)
	require.Nil(t, escapeTextPtr(nil))
	require.Equal(t, "", derefString(nil))
	require.Len(t, popularTitles(), 5)
}

func TestPostHTTPHelpersAndErrors(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?limit=12", nil)
	rec := httptest.NewRecorder()
	limit, ok := parseLimit(rec, req)
	require.True(t, ok)
	require.Equal(t, 12, limit)

	rec = httptest.NewRecorder()
	_, ok = parseLimit(rec, httptest.NewRequest(http.MethodGet, "/?limit=bad", nil))
	require.False(t, ok)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	id, ok := parseID(rec, "10")
	require.True(t, ok)
	require.Equal(t, int64(10), id)

	rec = httptest.NewRecorder()
	_, ok = parseID(rec, "0")
	require.False(t, ok)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	userID, ok := userIDFromContext(rec, httptest.NewRequest(http.MethodGet, "/", nil).WithContext(context.WithValue(context.Background(), "user_id", int64(10))))
	require.True(t, ok)
	require.Equal(t, int64(10), userID)

	rec = httptest.NewRecorder()
	_, ok = userIDFromContext(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.False(t, ok)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	for _, tc := range []struct {
		err  error
		code int
	}{
		{service.ErrInvalidInput, http.StatusBadRequest},
		{service.ErrPostContentRequired, http.StatusBadRequest},
		{service.ErrPostNotFound, http.StatusNotFound},
		{service.ErrProfileNotFound, http.StatusNotFound},
		{service.ErrForbidden, http.StatusForbidden},
		{errors.New("boom"), http.StatusInternalServerError},
	} {
		rec = httptest.NewRecorder()
		writeServiceError(rec, tc.err)
		require.Equal(t, tc.code, rec.Code)
	}
}
