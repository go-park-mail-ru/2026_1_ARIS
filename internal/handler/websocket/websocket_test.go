package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
