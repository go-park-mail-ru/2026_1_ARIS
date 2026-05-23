package websocket

import (
	"context"
	"sync"
	"time"
)

const presenceTimeout = 500 * time.Millisecond

type PresenceCache interface {
	SetOnline(ctx context.Context, userAccountID int64) error
	SetOffline(ctx context.Context, userAccountID int64) error
	Heartbeat(ctx context.Context, userAccountID int64) error
}

type Hub struct {
	register   chan *Client
	unregister chan *Client
	rooms      map[string]map[*Client]bool
	users      map[int64]int
	presence   PresenceCache
	mu         sync.Mutex
}

func NewHub(presence ...PresenceCache) *Hub {
	hub := &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		rooms:      make(map[string]map[*Client]bool),
		users:      make(map[int64]int),
	}
	if len(presence) > 0 {
		hub.presence = presence[0]
	}
	return hub
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			firstConnection := false
			h.mu.Lock()
			roomID := client.chatID
			if _, ok := h.rooms[roomID]; !ok {
				h.rooms[roomID] = make(map[*Client]bool)
			}
			h.rooms[roomID][client] = true
			if client.userID > 0 {
				firstConnection = h.users[client.userID] == 0
				h.users[client.userID]++
			}
			h.mu.Unlock()
			if firstConnection {
				h.markOnline(client.userID)
			}
		case client := <-h.unregister:
			lastConnection := false
			h.mu.Lock()
			if clients, ok := h.rooms[client.chatID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.rooms, client.chatID)
					}
					if client.userID > 0 && h.users[client.userID] > 0 {
						h.users[client.userID]--
						if h.users[client.userID] == 0 {
							delete(h.users, client.userID)
							lastConnection = true
						}
					}
				}
			}
			h.mu.Unlock()
			if lastConnection {
				h.markOffline(client.userID)
			}
		}
	}
}

func (h *Hub) BroadcastToChat(chatID string, message []byte) {
	h.mu.Lock()
	offlineUsers := make([]int64, 0)
	defer h.mu.Unlock()
	if clients, ok := h.rooms[chatID]; ok {
		for client := range clients {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(clients, client)
				if client.userID > 0 && h.users[client.userID] > 0 {
					h.users[client.userID]--
					if h.users[client.userID] == 0 {
						delete(h.users, client.userID)
						offlineUsers = append(offlineUsers, client.userID)
					}
				}
			}
		}
	}
	go func() {
		for _, userID := range offlineUsers {
			h.markOffline(userID)
		}
	}()
}

func (h *Hub) Heartbeat(userID int64) {
	if h.presence == nil || userID <= 0 {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), presenceTimeout)
		defer cancel()
		_ = h.presence.Heartbeat(ctx, userID)
	}()
}

func (h *Hub) markOnline(userID int64) {
	if h.presence == nil || userID <= 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), presenceTimeout)
	defer cancel()
	_ = h.presence.SetOnline(ctx, userID)
}

func (h *Hub) markOffline(userID int64) {
	if h.presence == nil || userID <= 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), presenceTimeout)
	defer cancel()
	_ = h.presence.SetOffline(ctx, userID)
}
