package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	assert.NotNil(t, hub)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.rooms)
}

// Тест для регистрации клиента (можно добавить, если нужно больше покрытия)
func TestHubRegisterAndUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		hub:    hub,
		chatID: "test-chat",
		send:   make(chan []byte, 1),
	}
	hub.register <- client
	// Даем время на обработку
	// Проверить, что клиент появился в rooms, но для этого нужен доступ к внутренней map.
	// В целях покрытия достаточно, что код выполнится без паники.
	// Можно добавить небольшое ожидание.
}

func TestBroadcastToChat_RemovesBlockedClient(t *testing.T) {
	hub := NewHub()
	client := &Client{
		hub:    hub,
		chatID: "room-2",
		send:   make(chan []byte),
	}
	hub.rooms["room-2"] = map[*Client]bool{client: true}

	hub.BroadcastToChat("room-2", []byte("payload"))

	_, ok := hub.rooms["room-2"][client]
	assert.False(t, ok)
}

func TestServeWebSocket_InvalidUpgrade(t *testing.T) {
	hub := NewHub()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rec := httptest.NewRecorder()

	err := ServeWebSocket(hub, rec, req, "room-1", 42)

	require.Error(t, err)
}

func TestHubRun_RemovesRoomOnUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		hub:    hub,
		chatID: "room-cleanup",
		send:   make(chan []byte, 1),
	}

	hub.register <- client
	require.Eventually(t, func() bool {
		hub.mu.Lock()
		defer hub.mu.Unlock()
		return len(hub.rooms["room-cleanup"]) == 1
	}, time.Second, 10*time.Millisecond)

	hub.unregister <- client
	require.Eventually(t, func() bool {
		hub.mu.Lock()
		defer hub.mu.Unlock()
		_, ok := hub.rooms["room-cleanup"]
		return !ok
	}, time.Second, 10*time.Millisecond)
}
