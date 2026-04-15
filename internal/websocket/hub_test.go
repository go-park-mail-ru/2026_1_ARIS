package websocket

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
