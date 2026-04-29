package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/chat/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/websocket"
)

type Handler struct {
	chat *service.Service
	hub  *websocket.Hub
}

func New(chat *service.Service, hub *websocket.Hub) *Handler {
	return &Handler{chat: chat, hub: hub}
}

func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		r.Get("/chats", h.GetChats)
		r.Post("/chats", h.CreateChat)
		r.Get("/chats/{chatID}/messages", h.GetMessages)
		r.Post("/chats/{chatID}/messages", h.SendMessage)
		r.Put("/chats/{chatID}/messages/{messageID}", h.UpdateMessage)
	})
}

func (h *Handler) RegisterWebSocketRoute(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		r.Get("/ws/{chatID}", h.HandleWebSocket)
	})
}

func (h *Handler) GetChats(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	chats, err := h.chat.GetUserChats(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	resp := make([]ChatResponse, 0, len(chats))
	for _, chat := range chats {
		resp = append(resp, mapChat(chat))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateChat(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	otherID, ok := parseQueryID(w, r, "otherUserId")
	if !ok {
		return
	}
	chat, err := h.chat.CreatePrivateChat(r.Context(), userID, otherID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, mapChat(chat))
}

func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	userID, chatID, ok := h.userAndChatID(w, r)
	if !ok {
		return
	}
	limit := parseBoundedInt(r.URL.Query().Get("limit"), 50, 1, 100)

	if rawAfter := r.URL.Query().Get("after"); rawAfter != "" {
		afterID, ok := parseNonNegativeID(w, rawAfter, "неверный ID сообщения")
		if !ok {
			return
		}
		messages, err := h.chat.GetMessagesAfter(r.Context(), userID, chatID, afterID, limit)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		resp := make([]MessageResponse, 0, len(messages))
		for _, msg := range messages {
			resp = append(resp, mapMessage(msg))
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	offset := parseBoundedInt(r.URL.Query().Get("offset"), 0, 0, 1<<30)
	messages, err := h.chat.GetMessages(r.Context(), userID, chatID, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	resp := make([]MessageResponse, 0, len(messages))
	for _, msg := range messages {
		resp = append(resp, mapMessage(msg))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, chatID, ok := h.userAndChatID(w, r)
	if !ok {
		return
	}
	var req textRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, "неверный формат запроса", http.StatusBadRequest)
		return
	}
	msg, err := h.chat.SendMessage(r.Context(), userID, chatID, req.Text)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	resp := mapMessage(msg)
	h.broadcast(chatID, resp)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) UpdateMessage(w http.ResponseWriter, r *http.Request) {
	userID, chatID, ok := h.userAndChatID(w, r)
	if !ok {
		return
	}
	messageID, ok := parsePathID(w, chi.URLParam(r, "messageID"), "неверный ID сообщения")
	if !ok {
		return
	}
	var req textRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, "неверный формат запроса", http.StatusBadRequest)
		return
	}
	msg, err := h.chat.UpdateMessage(r.Context(), userID, chatID, messageID, req.Text)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	resp := mapMessage(msg)
	h.broadcast(msg.ChatID, resp)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID, chatID, ok := h.userAndChatID(w, r)
	if !ok {
		return
	}
	allowed, err := h.chat.CheckUserInChat(r.Context(), chatID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if !allowed {
		writeServiceError(w, service.ErrForbidden)
		return
	}
	if err := websocket.ServeWebSocketWithHandler(h.hub, w, r, strconv.FormatInt(chatID, 10), userID, h.handleSocketMessage); err != nil {
		utils.WriteError(w, err.Error(), http.StatusBadRequest)
	}
}

func (h *Handler) userAndChatID(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return 0, 0, false
	}
	chatID, ok := parsePathID(w, chi.URLParam(r, "chatID"), "неверный ID чата")
	return userID, chatID, ok
}

func userIDFromContext(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok || userID <= 0 {
		utils.WriteError(w, "не авторизован", http.StatusUnauthorized)
		return 0, false
	}
	return userID, true
}

func parseQueryID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		utils.WriteError(w, "не указан ID собеседника", http.StatusBadRequest)
		return 0, false
	}
	return parsePathID(w, raw, "неверный формат ID")
}

func parsePathID(w http.ResponseWriter, raw string, message string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, message, http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func parseNonNegativeID(w http.ResponseWriter, raw string, message string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 0 {
		utils.WriteError(w, message, http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func parseBoundedInt(raw string, fallback, min, max int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < min || value > max {
		return fallback
	}
	return value
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrForbidden):
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "доступ запрещён"})
	case errors.Is(err, service.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *Handler) broadcast(chatID int64, msg MessageResponse) {
	if h.hub == nil {
		return
	}
	msgBytes, _ := json.Marshal(msg)
	h.hub.BroadcastToChat(strconv.FormatInt(chatID, 10), msgBytes)
}

func (h *Handler) handleSocketMessage(ctx context.Context, chatIDRaw string, userID int64, payload []byte) ([]byte, error) {
	chatID, err := strconv.ParseInt(chatIDRaw, 10, 64)
	if err != nil || chatID <= 0 {
		return nil, service.ErrInvalidInput
	}

	var req textRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, err
	}

	msg, err := h.chat.SendMessage(ctx, userID, chatID, req.Text)
	if err != nil {
		return nil, err
	}

	return json.Marshal(mapMessage(msg))
}

func mapChat(chat service.Chat) ChatResponse {
	return ChatResponse{
		ID:        strconv.FormatInt(chat.ID, 10),
		Uid:       chat.UID,
		Title:     chat.Title,
		AvatarID:  chat.AvatarID,
		Type:      string(chat.Type),
		IsActive:  chat.IsActive,
		CreatedAt: chat.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: chat.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func mapMessage(message service.Message) MessageResponse {
	return MessageResponse{
		ID:              strconv.FormatInt(message.ID, 10),
		Uid:             message.UID,
		Text:            message.Text,
		AuthorName:      message.AuthorName,
		ParentMessageID: int64PtrString(message.ParentMessageID),
		ChatID:          strconv.FormatInt(message.ChatID, 10),
		AuthorID:        strconv.FormatInt(message.AuthorID, 10),
		StickerID:       int64PtrString(message.StickerID),
		IsActive:        message.IsActive,
		CreatedAt:       message.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:       message.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func int64PtrString(value *int64) *string {
	if value == nil {
		return nil
	}
	out := strconv.FormatInt(*value, 10)
	return &out
}
