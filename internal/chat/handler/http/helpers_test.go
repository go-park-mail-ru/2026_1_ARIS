package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/chat/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/stretchr/testify/require"
)

func TestChatHTTPHelpers(t *testing.T) {
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	avatarID := int64(7)
	chatResp := mapChat(service.Chat{
		ID: 1, UID: "uid", Title: "Chat", AvatarID: &avatarID, Type: models.PrivateChat, IsActive: true, CreatedAt: now, UpdatedAt: now,
	})
	require.Equal(t, "1", chatResp.ID)
	require.Equal(t, &avatarID, chatResp.AvatarID)
	require.Equal(t, string(models.PrivateChat), chatResp.Type)

	text := "hello"
	parentID := int64(2)
	stickerID := int64(3)
	msgResp := mapMessage(service.Message{
		ID: 4, UID: "muid", Text: &text, AuthorName: "Neo", ParentMessageID: &parentID, ChatID: 1, AuthorID: 10,
		StickerID: &stickerID, IsActive: true, CreatedAt: now, UpdatedAt: now,
	})
	require.Equal(t, "4", msgResp.ID)
	require.Equal(t, "2", *msgResp.ParentMessageID)
	require.Equal(t, "3", *msgResp.StickerID)
	require.Nil(t, int64PtrString(nil))

	require.Equal(t, 10, parseBoundedInt("", 10, 1, 20))
	require.Equal(t, 12, parseBoundedInt("12", 10, 1, 20))
	require.Equal(t, 10, parseBoundedInt("bad", 10, 1, 20))
	require.Equal(t, 10, parseBoundedInt("50", 10, 1, 20))
}

func TestChatHTTPParseAndErrorHelpers(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?otherID=10", nil)
	id, ok := parseQueryID(rec, req, "otherID")
	require.True(t, ok)
	require.Equal(t, int64(10), id)

	rec = httptest.NewRecorder()
	_, ok = parseQueryID(rec, httptest.NewRequest(http.MethodGet, "/", nil), "otherID")
	require.False(t, ok)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	id, ok = parsePathID(rec, "11", "bad")
	require.True(t, ok)
	require.Equal(t, int64(11), id)

	rec = httptest.NewRecorder()
	_, ok = parseNonNegativeID(rec, "-1", "bad")
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
		{service.ErrForbidden, http.StatusForbidden},
		{service.ErrNotFound, http.StatusNotFound},
		{errors.New("boom"), http.StatusInternalServerError},
	} {
		rec = httptest.NewRecorder()
		writeServiceError(rec, tc.err)
		require.Equal(t, tc.code, rec.Code)
	}

	rec = httptest.NewRecorder()
	writeJSON(rec, http.StatusCreated, map[string]bool{"ok": true})
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}
