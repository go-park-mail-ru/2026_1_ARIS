package websocket

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait       = 10 * time.Second
	pongWait        = 60 * time.Second
	pingPeriod      = (pongWait * 9) / 10
	heartbeatPeriod = 5 * time.Second
	maxMessageSize  = 4096
	messageWait     = 5 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type MessageHandler func(ctx context.Context, roomID string, userID int64, payload []byte) ([]byte, error)
type ConnectionHandler func(ctx context.Context, roomID string, userID int64)

type Client struct {
	hub          *Hub
	conn         *websocket.Conn
	send         chan []byte
	roomID       string
	userID       int64
	initial      []byte
	onMessage    MessageHandler
	onDisconnect ConnectionHandler
	onHeartbeat  ConnectionHandler
}

func Serve(
	hub *Hub,
	w http.ResponseWriter,
	r *http.Request,
	roomID string,
	userID int64,
	initial []byte,
	onMessage MessageHandler,
	onDisconnect ConnectionHandler,
	onHeartbeat ConnectionHandler,
) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	client := &Client{
		hub:          hub,
		conn:         conn,
		send:         make(chan []byte, 256),
		roomID:       roomID,
		userID:       userID,
		initial:      initial,
		onMessage:    onMessage,
		onDisconnect: onDisconnect,
		onHeartbeat:  onHeartbeat,
	}
	client.hub.register <- client
	go client.writePump()
	go client.readPump()
	return nil
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		if c.onDisconnect != nil {
			ctx, cancel := context.WithTimeout(context.Background(), messageWait)
			c.onDisconnect(ctx, c.roomID, c.userID)
			cancel()
		}
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("game websocket error: %v", err)
			}
			break
		}
		if messageType != websocket.TextMessage || c.onMessage == nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), messageWait)
		broadcast, err := c.onMessage(ctx, c.roomID, c.userID, message)
		cancel()
		if err != nil || len(broadcast) == 0 {
			continue
		}
		c.send <- broadcast
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	heartbeatTicker := time.NewTicker(heartbeatPeriod)
	defer func() {
		ticker.Stop()
		heartbeatTicker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-heartbeatTicker.C:
			if c.onHeartbeat != nil {
				ctx, cancel := context.WithTimeout(context.Background(), messageWait)
				c.onHeartbeat(ctx, c.roomID, c.userID)
				cancel()
			}
		}
	}
}
