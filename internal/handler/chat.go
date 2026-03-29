package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/google/uuid"
)

type ChatHandler struct {
	chatService    service.ChatService
	messageService service.MessageService
}

func NewChatHandler(chatService service.ChatService, messageService service.MessageService) *ChatHandler {
	return &ChatHandler{
		chatService:    chatService,
		messageService: messageService,
	}
}

func (h *ChatHandler) GetChats(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		utils.WriteError(w, "не авторизован", http.StatusUnauthorized)
		return
	}

	chats, err := h.chatService.GetUserChats(r.Context(), userID)
	if err != nil {
		utils.WriteError(w, "ошибка загрузки чатов", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chats)
}

func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		utils.WriteError(w, "не авторизован", http.StatusUnauthorized)
		return
	}

	otherUserIDStr := r.URL.Query().Get("otherUserId")
	if otherUserIDStr == "" {
		utils.WriteError(w, "не указан ID собеседника", http.StatusBadRequest)
		return
	}
	otherUserID, err := uuid.Parse(otherUserIDStr)
	if err != nil {
		utils.WriteError(w, "неверный формат ID", http.StatusBadRequest)
		return
	}

	chat, err := h.chatService.CreatePrivateChat(r.Context(), userID, otherUserID)
	if err != nil {
		utils.WriteError(w, "ошибка создания чата", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(chat)
}

func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		utils.WriteError(w, "не авторизован", http.StatusUnauthorized)
		return
	}
	_ = userID

	chatIDStr := chi.URLParam(r, "chatID")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		utils.WriteError(w, "неверный ID чата", http.StatusBadRequest)
		return
	}

	limit := 50
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	offset := 0
	offsetStr := r.URL.Query().Get("offset")
	if offsetStr != "" {
		o, err := strconv.Atoi(offsetStr)
		if err == nil && o >= 0 {
			offset = o
		}
	}

	messages, err := h.messageService.GetMessages(r.Context(), chatID, limit, offset)
	if err != nil {
		utils.WriteError(w, "ошибка загрузки сообщений", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		utils.WriteError(w, "не авторизован", http.StatusUnauthorized)
		return
	}

	chatIDStr := chi.URLParam(r, "chatID")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		utils.WriteError(w, "неверный ID чата", http.StatusBadRequest)
		return
	}

	ok, err = h.chatService.CheckUserInChat(r.Context(), chatID, userID)
	if err != nil {
		utils.WriteError(w, "ошибка проверки чата", http.StatusInternalServerError)
		return
	}
	if !ok {
		utils.WriteError(w, "доступ запрещён", http.StatusForbidden)
		return
	}

	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, "неверный формат запроса", http.StatusBadRequest)
		return
	}
	if req.Text == "" {
		utils.WriteError(w, "текст сообщения не может быть пустым", http.StatusBadRequest)
		return
	}

	msg, err := h.messageService.SendMessage(r.Context(), chatID, userID, req.Text)
	if err != nil {
		utils.WriteError(w, "ошибка отправки сообщения", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg)
}
