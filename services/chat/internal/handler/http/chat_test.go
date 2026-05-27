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
	userpb "github.com/go-park-mail-ru/2026_1_ARIS/proto/user"
	usermock "github.com/go-park-mail-ru/2026_1_ARIS/proto/user/mock"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/repository"
	repomocks "github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/repository/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/usecase"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestChatHTTPPresenceRoutes(t *testing.T) {
	t.Parallel()

	router, _, _ := newChatRouter(t)

	for _, path := range []string{"/presence/online", "/presence/heartbeat", "/presence/offline", "/presence/force-offline"} {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			rr := serveChat(t, router, http.MethodPost, path, nil, 5)
			require.Equal(t, http.StatusNoContent, rr.Code)
		})
	}

	rr := serveChat(t, router, http.MethodPost, "/presence/online", nil, 0)
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestChatHTTPStickerRoutes(t *testing.T) {
	t.Parallel()

	t.Run("list my packs", func(t *testing.T) {
		t.Parallel()
		router, users, repos := newChatRouter(t)
		expectChatProfile(users, 5, 10)
		repos.stickers.EXPECT().
			ListPacksByAuthorID(gomock.Any(), int64(10), 10, 0).
			Return([]model.StickerPack{chatStickerPack(20, 10)}, nil)

		rr := serveChat(t, router, http.MethodGet, "/sticker-packs?my=true&limit=10", nil, 5)

		require.Equal(t, http.StatusOK, rr.Code)
		require.Contains(t, rr.Body.String(), `"title":"pack"`)
	})

	t.Run("create pack", func(t *testing.T) {
		t.Parallel()
		router, users, repos := newChatRouter(t)
		expectChatProfile(users, 5, 10)
		repos.stickers.EXPECT().CreatePack(gomock.Any(), gomock.Any()).Return(int64(20), nil)
		repos.stickers.EXPECT().GetPack(gomock.Any(), int64(20)).Return(ptr(chatStickerPack(20, 10)), nil)

		rr := serveChat(t, router, http.MethodPost, "/sticker-packs", map[string]any{"title": " pack "}, 5)

		require.Equal(t, http.StatusCreated, rr.Code)
		require.Contains(t, rr.Body.String(), `"id":"20"`)
	})

	t.Run("list stickers", func(t *testing.T) {
		t.Parallel()
		router, users, repos := newChatRouter(t)
		expectChatProfile(users, 5, 10)
		repos.stickers.EXPECT().
			ListByPackID(gomock.Any(), int64(20), 25, 0).
			Return([]model.Sticker{chatSticker(30, 20, 99)}, nil)

		rr := serveChat(t, router, http.MethodGet, "/sticker-packs/20/stickers?limit=25", nil, 5)

		require.Equal(t, http.StatusOK, rr.Code)
		require.Contains(t, rr.Body.String(), `"id":"30"`)
	})

	t.Run("create sticker", func(t *testing.T) {
		t.Parallel()
		router, users, repos := newChatRouter(t)
		expectChatProfile(users, 5, 10)
		repos.stickers.EXPECT().GetPack(gomock.Any(), int64(20)).Return(ptr(chatStickerPack(20, 10)), nil)
		repos.stickers.EXPECT().
			GetMediaInfo(gomock.Any(), int64(99)).
			Return(&model.MediaInfo{ID: 99, AuthorID: 10, MimeType: "image/png", Link: "/m/99.png", Size: 100}, nil)
		repos.stickers.EXPECT().NextStickerOrder(gomock.Any(), int64(20)).Return(1, nil)
		repos.stickers.EXPECT().CreateSticker(gomock.Any(), gomock.Any()).Return(int64(30), nil)
		repos.stickers.EXPECT().Get(gomock.Any(), int64(30)).Return(ptr(chatSticker(30, 20, 99)), nil)

		rr := serveChat(t, router, http.MethodPost, "/sticker-packs/20/stickers", map[string]any{"mediaID": 99}, 5)

		require.Equal(t, http.StatusCreated, rr.Code)
		require.Contains(t, rr.Body.String(), `"mediaId":"99"`)
	})
}

func TestChatHTTPMessageRoutes(t *testing.T) {
	t.Parallel()

	t.Run("get messages", func(t *testing.T) {
		t.Parallel()
		router, users, repos := newChatRouter(t)
		expectChatProfile(users, 5, 10)
		expectChatProfile(users, 5, 10)
		repos.members.EXPECT().
			GetByChatID(gomock.Any(), int64(1)).
			Return([]model.ChatMember{{ChatID: 1, MemberID: 10}}, nil)
		repos.messages.EXPECT().
			GetByChatID(gomock.Any(), int64(1), 25, 0).
			Return([]model.Message{chatMessage(100, 1, 10, "hello")}, nil)
		expectMessageDecorations(users, repos, 10, []int64{100})

		rr := serveChat(t, router, http.MethodGet, "/chats/1/messages?limit=25", nil, 5)

		require.Equal(t, http.StatusOK, rr.Code)
		require.Contains(t, rr.Body.String(), `"text":"hello"`)
	})

	t.Run("send message", func(t *testing.T) {
		t.Parallel()
		router, users, repos := newChatRouter(t)
		expectChatProfile(users, 5, 10)
		expectChatProfile(users, 5, 10)
		repos.members.EXPECT().
			GetByChatID(gomock.Any(), int64(1)).
			Return([]model.ChatMember{{ChatID: 1, MemberID: 10}}, nil)
		repos.messages.EXPECT().
			Save(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, msg *model.Message) error {
				msg.ID = 101
				msg.Uid = uuid.New()
				msg.CreatedAt = time.Now()
				msg.UpdatedAt = msg.CreatedAt
				return nil
			})
		expectMessageDecorations(users, repos, 10, []int64{101})

		rr := serveChat(t, router, http.MethodPost, "/chats/1/messages", map[string]any{"text": " hello "}, 5)

		require.Equal(t, http.StatusOK, rr.Code)
		require.Contains(t, rr.Body.String(), `"text":"hello"`)
	})

	t.Run("invalid chat id", func(t *testing.T) {
		t.Parallel()
		router, _, _ := newChatRouter(t)

		rr := serveChat(t, router, http.MethodGet, "/chats/0/messages", nil, 5)

		require.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestChatHTTPHelpers(t *testing.T) {
	t.Parallel()

	require.Equal(t, 50, parseBoundedInt("", 50, 1, 100))
	require.Equal(t, 50, parseBoundedInt("bad", 50, 1, 100))
	require.Equal(t, 50, parseBoundedInt("101", 50, 1, 100))
	require.Equal(t, 25, parseBoundedInt("25", 50, 1, 100))

	now := time.Now()
	require.Nil(t, timePtrString(nil))
	require.NotNil(t, timePtrString(&now))
	require.Nil(t, int64PtrString(nil))
	id := int64(42)
	require.Equal(t, "42", *int64PtrString(&id))

	text := "hello"
	msg := mapMessage(usecase.Message{ID: 1, Text: &text, ChatID: 2, AuthorID: 3, Type: model.MessageTypeText, CreatedAt: now, UpdatedAt: now})
	require.Equal(t, "1", msg.ID)
	require.Equal(t, "2", msg.ChatID)
}

type chatRepoMocks struct {
	members      *repomocks.MockChatMemberRepo
	messages     *repomocks.MockMessageRepo
	messageMedia *repomocks.MockMessageMediaRepo
	reactions    *repomocks.MockReactionRepo
	stickers     *repomocks.MockStickerRepo
}

func newChatRouter(t *testing.T) (*chi.Mux, *usermock.MockUserServiceClient, chatRepoMocks) {
	t.Helper()
	ctrl := gomock.NewController(t)
	users := usermock.NewMockUserServiceClient(ctrl)
	members := repomocks.NewMockChatMemberRepo(ctrl)
	messages := repomocks.NewMockMessageRepo(ctrl)
	messageMedia := repomocks.NewMockMessageMediaRepo(ctrl)
	reactions := repomocks.NewMockReactionRepo(ctrl)
	stickers := repomocks.NewMockStickerRepo(ctrl)
	store := repository.Store{
		ChatMembers:  members,
		Messages:     messages,
		MessageMedia: messageMedia,
		Reactions:    reactions,
		Stickers:     stickers,
	}
	svc := usecase.New(store, users)
	router := chi.NewRouter()
	New(svc, nil).RegisterRoutes(router, nil)
	return router, users, chatRepoMocks{members: members, messages: messages, messageMedia: messageMedia, reactions: reactions, stickers: stickers}
}

func serveChat(t *testing.T, router *chi.Mux, method, path string, body any, userID int64) *httptest.ResponseRecorder {
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

func expectChatProfile(users *usermock.MockUserServiceClient, accountID, profileID int64) {
	users.EXPECT().
		GetProfileByUserAccount(gomock.Any(), &userpb.GetProfileByUserAccountRequest{UserAccountId: accountID}).
		Return(&userpb.GetProfileByUserAccountResponse{ProfileId: profileID}, nil)
}

func chatStickerPack(id, authorID int64) model.StickerPack {
	now := time.Now()
	return model.StickerPack{ID: id, Uid: uuid.New(), Title: "pack", AuthorID: &authorID, IsActive: true, CreatedAt: now, UpdatedAt: now}
}

func chatSticker(id, packID, mediaID int64) model.Sticker {
	now := time.Now()
	mime := "image/png"
	link := "/m/99.png"
	return model.Sticker{ID: id, Uid: uuid.New(), PackID: &packID, MediaID: &mediaID, MimeType: &mime, Link: &link, IsActive: true, CreatedAt: now, UpdatedAt: now}
}

func chatMessage(id, chatID, authorID int64, text string) model.Message {
	now := time.Now()
	return model.Message{ID: id, Uid: uuid.New(), Text: &text, ChatID: chatID, AuthorID: authorID, Type: model.MessageTypeText, IsActive: true, CreatedAt: now, UpdatedAt: now}
}

func expectMessageDecorations(users *usermock.MockUserServiceClient, repos chatRepoMocks, profileID int64, messageIDs []int64) {
	repos.messageMedia.EXPECT().GetByMessageIDs(gomock.Any(), messageIDs).Return(map[int64][]model.MessageMedia{}, nil)
	repos.reactions.EXPECT().GetSummaryByMessageIDs(gomock.Any(), messageIDs).Return(map[int64][]model.ReactionSummary{}, nil)
	repos.reactions.EXPECT().GetUserReactionsByMessageIDs(gomock.Any(), messageIDs, profileID).Return(map[int64]string{}, nil)
	users.EXPECT().
		GetProfileSummary(gomock.Any(), &userpb.GetProfileSummaryRequest{ProfileId: profileID}).
		Return(&userpb.GetProfileSummaryResponse{ProfileId: profileID, UserAccountId: 5, FirstName: "Neo", LastName: "Anderson", Username: "neo"}, nil)
}

func ptr[T any](value T) *T {
	return &value
}
