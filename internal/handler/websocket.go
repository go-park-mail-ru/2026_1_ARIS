package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/websocket"
)

type WebSocketHandler struct {
	hub         *websocket.Hub
	chatService service.ChatService
}

func NewWebSocketHandler(hub *websocket.Hub, chatService service.ChatService) *WebSocketHandler {
	return &WebSocketHandler{
		hub:         hub,
		chatService: chatService,
	}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	chatIDStr := chi.URLParam(r, "chatID")
	if chatIDStr == "" {
		utils.WriteError(w, "chatID required", http.StatusBadRequest)
		return
	}

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		utils.WriteError(w, "invalid chatID", http.StatusBadRequest)
		return
	}

	// Проверяем, что пользователь является участником чата
	if h.chatService != nil {
		allowed, err := h.chatService.CheckUserInChat(r.Context(), chatID, userID)
		if err != nil {
			utils.WriteError(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !allowed {
			utils.WriteError(w, "forbidden", http.StatusForbidden)
			return
		}
	}

	if err := websocket.ServeWebSocket(h.hub, w, r, chatIDStr, userID); err != nil {
		utils.WriteError(w, err.Error(), http.StatusInternalServerError)
	}
}
