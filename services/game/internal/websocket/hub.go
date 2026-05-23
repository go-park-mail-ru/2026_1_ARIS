package websocket

import "sync"

type Hub struct {
	register   chan *Client
	unregister chan *Client
	rooms      map[string]map[*Client]bool
	mu         sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		unregister: make(chan *Client),
		rooms:      make(map[string]map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if _, ok := h.rooms[client.roomID]; !ok {
				h.rooms[client.roomID] = make(map[*Client]bool)
			}
			h.rooms[client.roomID][client] = true
			h.mu.Unlock()
			if len(client.initial) > 0 {
				client.send <- client.initial
			}
		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.rooms[client.roomID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.rooms, client.roomID)
					}
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) BroadcastToRoom(roomID string, message []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.rooms[roomID]; ok {
		for client := range clients {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(clients, client)
			}
		}
	}
}

func (h *Hub) BroadcastRoomFunc(roomID string, build func(userID int64) []byte) {
	h.mu.Lock()
	if clients, ok := h.rooms[roomID]; ok {
		snapshot := make([]*Client, 0, len(clients))
		for client := range clients {
			snapshot = append(snapshot, client)
		}
		h.mu.Unlock()
		for _, client := range snapshot {
			message := build(client.userID)
			h.mu.Lock()
			currentClients, ok := h.rooms[roomID]
			if !ok || !currentClients[client] || len(message) == 0 {
				h.mu.Unlock()
				continue
			}
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(currentClients, client)
			}
			h.mu.Unlock()
		}
		return
	}
	h.mu.Unlock()
}
