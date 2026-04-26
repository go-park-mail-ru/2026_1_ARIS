package websocket

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	mock_service "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/mocks"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/websocket"
)

func TestNewWebSocketHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	hub := websocket.NewHub()
	handler := NewWebSocketHandler(hub, mockChatSvc)
	assert.NotNil(t, handler)
	assert.Equal(t, hub, handler.hub)
}

func TestHandleWebSocket_NotAWebSocket(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	hub := websocket.NewHub()
	go hub.Run()

	handler := NewWebSocketHandler(hub, mockChatSvc)

	req := httptest.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()

	handler.HandleWebSocket(w, req)

	assert.NotEqual(t, http.StatusSwitchingProtocols, w.Code)
}

func TestHandleWebSocket_MissingChatID(t *testing.T) {
	handler := NewWebSocketHandler(nil, nil) // hub и chatService могут быть nil

	req := httptest.NewRequest(http.MethodGet, "/ws/", nil)
	rec := httptest.NewRecorder()

	ctx := context.WithValue(req.Context(), "user_id", int64(11))
	req = req.WithContext(ctx)

	handler.HandleWebSocket(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "required")
}

func TestHandleWebSocket_InvalidChatID_WithRouter(t *testing.T) {
	handler := NewWebSocketHandler(nil, nil)

	r := chi.NewRouter()
	r.Get("/ws/{chatID}", handler.HandleWebSocket)

	req := httptest.NewRequest(http.MethodGet, "/ws/abc", nil)
	// Добавляем user_id через контекст (иначе будет 401)
	ctx := context.WithValue(req.Context(), "user_id", int64(11))
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid chatID")
}

func TestHandleWebSocket_ChatServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	hub := websocket.NewHub()
	handler := NewWebSocketHandler(hub, mockChatSvc)

	r := chi.NewRouter()
	r.Get("/ws/{chatID}", handler.HandleWebSocket)

	mockChatSvc.EXPECT().
		CheckUserInChat(gomock.Any(), int64(15), int64(11)).
		Return(false, errors.New("boom"))

	req := httptest.NewRequest(http.MethodGet, "/ws/15", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(11)))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleWebSocket_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChatSvc := mock_service.NewMockChatService(ctrl)
	hub := websocket.NewHub()
	handler := NewWebSocketHandler(hub, mockChatSvc)

	r := chi.NewRouter()
	r.Get("/ws/{chatID}", handler.HandleWebSocket)

	mockChatSvc.EXPECT().
		CheckUserInChat(gomock.Any(), int64(15), int64(11)).
		Return(false, nil)

	req := httptest.NewRequest(http.MethodGet, "/ws/15", nil)
	req = req.WithContext(context.WithValue(req.Context(), "user_id", int64(11)))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}
