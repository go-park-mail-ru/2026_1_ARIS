package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	useraccount "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/user_account"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
	userservice "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/websocket"
)

type ChatHandler struct {
	chatService     service.ChatService
	messageService  service.MessageService
	userAccountRepo useraccount.UserAccountRepo
	userService     userservice.UserService
	hub             *websocket.Hub
}

func NewChatHandler(
	chatService service.ChatService,
	messageService service.MessageService,
	userAccountRepo useraccount.UserAccountRepo,
	userService userservice.UserService,
	hub *websocket.Hub,
) *ChatHandler {
	return &ChatHandler{
		chatService:     chatService,
		messageService:  messageService,
		userAccountRepo: userAccountRepo,
		userService:     userService,
		hub:             hub,
	}
}

type ChatResponse struct {
	ID        string `json:"id"`
	Uid       string `json:"uid"`
	Title     string `json:"title"`
	AvatarID  *int64 `json:"avatarId,omitempty"`
	Type      string `json:"type"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type MessageResponse struct {
	ID              string  `json:"id"`
	Uid             string  `json:"uid"`
	Text            *string `json:"text,omitempty"`
	AuthorName      string  `json:"authorName"`
	ParentMessageID *string `json:"parentMessage,omitempty"`
	ChatID          string  `json:"chat"`
	AuthorID        string  `json:"authorId"`
	StickerID       *string `json:"sticker,omitempty"`
	IsActive        bool    `json:"isActive"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

func (h *ChatHandler) GetChats(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		utils.WriteError(w, "не авторизован", http.StatusUnauthorized)
		return
	}

	chats, err := h.chatService.GetUserChats(r.Context(), userID)
	if err != nil {
		utils.WriteError(w, "ошибка загрузки чатов", http.StatusInternalServerError)
		return
	}

	if len(chats) == 0 {
		h.ensureGuaranteedDialog(r.Context(), userID)
		chats, err = h.chatService.GetUserChats(r.Context(), userID)
		if err != nil {
			utils.WriteError(w, "ошибка загрузки чатов", http.StatusInternalServerError)
			return
		}
	}

	chats = h.filterChatsForList(r.Context(), chats, userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.mapChatsResponse(r.Context(), chats, userID))
}

func (h *ChatHandler) ensureGuaranteedDialog(ctx context.Context, userID int64) {
	if h.userAccountRepo == nil {
		return
	}

	currentUser, err := h.userAccountRepo.Get(ctx, userID)
	if err != nil {
		return
	}

	targetUsernames := []string{"sergeyshulginenko", "ffffff"}

	for _, username := range targetUsernames {
		targetUser, err := h.userAccountRepo.GetByUsername(ctx, username)
		if err != nil || targetUser == nil || targetUser.ID == currentUser.ID {
			continue
		}
		_, _ = h.chatService.CreatePrivateChat(ctx, currentUser.ID, targetUser.ID)
		return
	}

	users, err := h.userAccountRepo.List(ctx, 0, 20)
	if err != nil {
		return
	}
	for _, user := range users {
		if user.ID == currentUser.ID {
			continue
		}
		_, _ = h.chatService.CreatePrivateChat(ctx, currentUser.ID, user.ID)
		return
	}
}

func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		utils.WriteError(w, "не авторизован", http.StatusUnauthorized)
		return
	}

	otherUserIDStr := r.URL.Query().Get("otherUserId")
	if otherUserIDStr == "" {
		utils.WriteError(w, "не указан ID собеседника", http.StatusBadRequest)
		return
	}

	otherUserID, err := strconv.ParseInt(otherUserIDStr, 10, 64)
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
	json.NewEncoder(w).Encode(h.mapChatResponse(r.Context(), *chat, userID))
}

func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		utils.WriteError(w, "не авторизован", http.StatusUnauthorized)
		return
	}

	chatIDStr := chi.URLParam(r, "chatID")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		utils.WriteError(w, "неверный ID чата", http.StatusBadRequest)
		return
	}

	allowed, err := h.chatService.CheckUserInChat(r.Context(), chatID, userID)
	if err != nil {
		utils.WriteError(w, "ошибка проверки чата", http.StatusInternalServerError)
		return
	}
	if !allowed {
		utils.WriteError(w, "доступ запрещён", http.StatusForbidden)
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
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
	json.NewEncoder(w).Encode(h.mapMessagesResponse(r.Context(), messages, userID))
}

func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		utils.WriteError(w, "не авторизован", http.StatusUnauthorized)
		return
	}

	chatIDStr := chi.URLParam(r, "chatID")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		utils.WriteError(w, "неверный ID чата", http.StatusBadRequest)
		return
	}

	allowed, err := h.chatService.CheckUserInChat(r.Context(), chatID, userID)
	if err != nil {
		utils.WriteError(w, "ошибка проверки чата", http.StatusInternalServerError)
		return
	}
	if !allowed {
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

	// Отправка через сокеты всем участникам чата
	msgResponse := h.mapMessageResponse(r.Context(), *msg)
	msgBytes, _ := json.Marshal(msgResponse)
	h.hub.BroadcastToChat(strconv.FormatInt(chatID, 10), msgBytes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgResponse)
}

func (h *ChatHandler) mapChatsResponse(ctx context.Context, chats []models.Chat, viewerUserID int64) []ChatResponse {
	result := make([]ChatResponse, 0, len(chats))
	for _, chat := range chats {
		result = append(result, h.mapChatResponse(ctx, chat, viewerUserID))
	}
	return result
}

func (h *ChatHandler) mapChatResponse(ctx context.Context, chat models.Chat, viewerUserID int64) ChatResponse {
	title := h.resolveChatTitle(ctx, chat, viewerUserID)
	return ChatResponse{
		ID:        strconv.FormatInt(chat.ID, 10),
		Uid:       chat.Uid.String(),
		Title:     title,
		AvatarID:  chat.AvatarID,
		Type:      string(chat.Type),
		IsActive:  chat.IsActive,
		CreatedAt: chat.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: chat.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func (h *ChatHandler) mapMessagesResponse(ctx context.Context, messages []models.Message, viewerUserID int64) []MessageResponse {
	result := make([]MessageResponse, 0, len(messages))
	for _, message := range messages {
		result = append(result, h.mapMessageResponse(ctx, message))
	}
	return result
}

func (h *ChatHandler) mapMessageResponse(ctx context.Context, message models.Message) MessageResponse {
	var parentMessageID *string
	if message.ParentMessageID != nil {
		value := strconv.FormatInt(*message.ParentMessageID, 10)
		parentMessageID = &value
	}
	var stickerID *string
	if message.StickerID != nil {
		value := strconv.FormatInt(*message.StickerID, 10)
		stickerID = &value
	}
	return MessageResponse{
		ID:              strconv.FormatInt(message.ID, 10),
		Uid:             message.Uid.String(),
		Text:            message.Text,
		AuthorName:      h.resolveUserDisplayName(ctx, message.AuthorID),
		ParentMessageID: parentMessageID,
		ChatID:          strconv.FormatInt(message.ChatID, 10),
		AuthorID:        strconv.FormatInt(message.AuthorID, 10),
		StickerID:       stickerID,
		IsActive:        message.IsActive,
		CreatedAt:       message.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:       message.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func (h *ChatHandler) filterChatsForList(ctx context.Context, chats []models.Chat, viewerUserID int64) []models.Chat {
	result := make([]models.Chat, 0, len(chats))
	for _, chat := range chats {
		if h.chatHasMessages(ctx, chat.ID) || h.isSergeySupportChat(ctx, chat, viewerUserID) {
			result = append(result, chat)
		}
	}
	return result
}

func (h *ChatHandler) chatHasMessages(ctx context.Context, chatID int64) bool {
	if h.messageService == nil {
		return false
	}
	messages, err := h.messageService.GetMessages(ctx, chatID, 1, 0)
	return err == nil && len(messages) > 0
}

func (h *ChatHandler) isSergeySupportChat(ctx context.Context, chat models.Chat, viewerUserID int64) bool {
	if h.chatService == nil || h.userAccountRepo == nil {
		return false
	}
	members, err := h.chatService.GetChatMembers(ctx, chat.ID)
	if err != nil {
		return false
	}
	for _, member := range members {
		if member.MemberID == viewerUserID {
			continue
		}
		account, err := h.userAccountRepo.Get(ctx, member.MemberID)
		if err != nil || account == nil {
			continue
		}
		return account.Username == "sergeyshulginenko"
	}
	return false
}

func (h *ChatHandler) resolveChatTitle(ctx context.Context, chat models.Chat, viewerUserID int64) string {
	if chat.Type != models.PrivateChat || h.chatService == nil {
		return chat.Title
	}
	members, err := h.chatService.GetChatMembers(ctx, chat.ID)
	if err != nil {
		return chat.Title
	}
	for _, member := range members {
		if member.MemberID == viewerUserID {
			continue
		}
		return h.resolveUserDisplayName(ctx, member.MemberID)
	}
	return chat.Title
}

func (h *ChatHandler) resolveUserDisplayName(ctx context.Context, userID int64) string {
	if h.userService == nil {
		return "Пользователь"
	}
	profile, err := h.userService.GetUserProfileByUserAccountID(ctx, userID)
	if err != nil || profile == nil {
		return "Пользователь"
	}
	fullName := profile.FirstName
	if profile.LastName != "" {
		if fullName != "" {
			fullName += " "
		}
		fullName += profile.LastName
	}
	if fullName == "" {
		return "Пользователь"
	}
	return fullName
}
