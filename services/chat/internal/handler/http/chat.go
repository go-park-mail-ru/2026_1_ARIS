package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/websocket"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
)

type Handler struct {
	chat *usecase.Service
	hub  *websocket.Hub
}

func New(chat *usecase.Service, hub *websocket.Hub) *Handler {
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
		r.Put("/chats/{chatID}/messages/{messageID}/reaction", h.SetMessageReaction)
		r.Delete("/chats/{chatID}/messages/{messageID}/reaction", h.DeleteMessageReaction)
		r.Post("/presence/online", h.SetPresenceOnline)
		r.Post("/presence/heartbeat", h.HeartbeatPresence)
		r.Post("/presence/offline", h.SetPresenceOffline)
		r.Post("/presence/force-offline", h.ForcePresenceOffline)
		r.Get("/sticker-packs", h.GetStickerPacks)
		r.Post("/sticker-packs", h.CreateStickerPack)
		r.Get("/sticker-packs/{packID}/stickers", h.GetStickersByPack)
		r.Post("/sticker-packs/{packID}/stickers", h.CreateSticker)
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
	utils.WriteJSON(w, http.StatusOK, resp)
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
	utils.WriteJSON(w, http.StatusCreated, mapChat(chat))
}

func (h *Handler) SetPresenceOnline(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	if err := h.chat.SetPresenceOnline(r.Context(), userID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) HeartbeatPresence(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	if err := h.chat.HeartbeatPresence(r.Context(), userID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetPresenceOffline(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	if err := h.chat.SetPresenceOffline(r.Context(), userID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ForcePresenceOffline(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	if err := h.chat.ForcePresenceOffline(r.Context(), userID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
		writeMessages(w, messages)
		return
	}

	offset := parseBoundedInt(r.URL.Query().Get("offset"), 0, 0, 1<<30)
	messages, err := h.chat.GetMessages(r.Context(), userID, chatID, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeMessages(w, messages)
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, chatID, ok := h.userAndChatID(w, r)
	if !ok {
		return
	}
	var req messageRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	msg, err := h.chat.SendMessage(r.Context(), userID, chatID, mapMessageInput(req))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	resp := mapMessage(msg)
	h.broadcast(chatID, resp)
	utils.WriteJSON(w, http.StatusOK, resp)
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
	var req messageRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	msg, err := h.chat.UpdateMessage(r.Context(), userID, chatID, messageID, req.Text)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	resp := mapMessage(msg)
	h.broadcast(msg.ChatID, resp)
	utils.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetStickerPacks(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	limit := parseBoundedInt(r.URL.Query().Get("limit"), 50, 1, 100)
	offset := parseBoundedInt(r.URL.Query().Get("offset"), 0, 0, 1<<30)
	myOnly := r.URL.Query().Get("my") == "true"
	packs, err := h.chat.GetStickerPacks(r.Context(), userID, r.URL.Query().Get("search"), myOnly, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	resp := make([]StickerPackResponse, 0, len(packs))
	for _, pack := range packs {
		resp = append(resp, mapStickerPack(pack))
	}
	utils.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateStickerPack(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	var req stickerPackRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	pack, err := h.chat.CreateStickerPack(r.Context(), userID, usecase.StickerPackInput{Title: req.Title})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, mapStickerPack(pack))
}

func (h *Handler) SetMessageReaction(w http.ResponseWriter, r *http.Request) {
	userID, chatID, ok := h.userAndChatID(w, r)
	if !ok {
		return
	}
	messageID, ok := parsePathID(w, chi.URLParam(r, "messageID"), "неверный ID сообщения")
	if !ok {
		return
	}
	var req reactionRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	msg, err := h.chat.SetMessageReaction(r.Context(), userID, chatID, messageID, req.Type)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	resp := mapMessage(msg)
	h.broadcast(chatID, resp)
	utils.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) DeleteMessageReaction(w http.ResponseWriter, r *http.Request) {
	userID, chatID, ok := h.userAndChatID(w, r)
	if !ok {
		return
	}

	messageID, ok := parsePathID(w, chi.URLParam(r, "messageID"), "неверный ID сообщения")
	if !ok {
		return
	}

	msg, err := h.chat.DeleteMessageReaction(r.Context(), userID, chatID, messageID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := mapMessage(msg)
	h.broadcast(chatID, resp)
	utils.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetStickersByPack(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	packID, ok := parsePathID(w, chi.URLParam(r, "packID"), "неверный ID набора")
	if !ok {
		return
	}
	limit := parseBoundedInt(r.URL.Query().Get("limit"), 100, 1, 200)
	offset := parseBoundedInt(r.URL.Query().Get("offset"), 0, 0, 1<<30)
	stickers, err := h.chat.GetStickersByPack(r.Context(), userID, packID, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	resp := make([]StickerResponse, 0, len(stickers))
	for _, sticker := range stickers {
		resp = append(resp, mapSticker(sticker))
	}
	utils.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateSticker(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	packID, ok := parsePathID(w, chi.URLParam(r, "packID"), "неверный ID набора")
	if !ok {
		return
	}
	var req stickerRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	sticker, err := h.chat.CreateSticker(r.Context(), userID, packID, usecase.StickerInput{MediaID: req.MediaID, SortOrder: req.SortOrder})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, mapSticker(sticker))
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
		writeServiceError(w, usecase.ErrForbidden)
		return
	}
	if err := websocket.ServeWebSocketWithHandler(h.hub, w, r, strconv.FormatInt(chatID, 10), userID, h.handleSocketMessage); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
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
		writeError(w, "не авторизован", http.StatusUnauthorized)
		return 0, false
	}
	return userID, true
}

func parseQueryID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		writeError(w, "не указан ID собеседника", http.StatusBadRequest)
		return 0, false
	}
	return parsePathID(w, raw, "неверный формат ID")
}

func parsePathID(w http.ResponseWriter, raw string, message string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, message, http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func parseNonNegativeID(w http.ResponseWriter, raw string, message string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 0 {
		writeError(w, message, http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func parseBoundedInt(raw string, fallback, minValue, maxValue int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minValue || value > maxValue {
		return fallback
	}
	return value
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidInput):
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, usecase.ErrForbidden):
		utils.WriteJSON(w, http.StatusForbidden, errorResponse{Error: "доступ запрещён"})
	case errors.Is(err, usecase.ErrNotFound):
		utils.WriteJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	default:
		utils.WriteJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func writeError(w http.ResponseWriter, message string, status int) {
	utils.WriteJSON(w, status, errorResponse{Error: message})
}

func writeMessages(w http.ResponseWriter, messages []usecase.Message) {
	resp := make([]MessageResponse, 0, len(messages))
	for _, msg := range messages {
		resp = append(resp, mapMessage(msg))
	}
	utils.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) broadcast(chatID int64, msg MessageResponse) {
	if h.hub == nil {
		return
	}
	msgBytes, _ := utils.MarshalJSON(msg)
	h.hub.BroadcastToChat(strconv.FormatInt(chatID, 10), msgBytes)
}

func (h *Handler) handleSocketMessage(ctx context.Context, chatIDRaw string, userID int64, payload []byte) ([]byte, error) {
	chatID, err := strconv.ParseInt(chatIDRaw, 10, 64)
	if err != nil || chatID <= 0 {
		return nil, usecase.ErrInvalidInput
	}

	var req messageRequest
	if err := utils.UnmarshalJSON(payload, &req); err != nil {
		return nil, err
	}

	msg, err := h.chat.SendMessage(ctx, userID, chatID, mapMessageInput(req))
	if err != nil {
		return nil, err
	}

	return utils.MarshalJSON(mapMessage(msg))
}

func mapChat(chat usecase.Chat) ChatResponse {
	return ChatResponse{
		ID:                        strconv.FormatInt(chat.ID, 10),
		Uid:                       chat.UID,
		Title:                     chat.Title,
		AvatarID:                  chat.AvatarID,
		AvatarLink:                chat.AvatarLink,
		Type:                      string(chat.Type),
		IsActive:                  chat.IsActive,
		InterlocutorProfileID:     int64PtrString(chat.InterlocutorProfileID),
		InterlocutorUserAccountID: int64PtrString(chat.InterlocutorUserAccountID),
		IsOnline:                  chat.IsOnline,
		LastSeenAt:                timePtrString(chat.LastSeenAt),
		CreatedAt:                 chat.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:                 chat.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func mapMessage(message usecase.Message) MessageResponse {
	return MessageResponse{
		ID:              strconv.FormatInt(message.ID, 10),
		Uid:             message.UID,
		Text:            message.Text,
		AuthorName:      message.AuthorName,
		ParentMessageID: int64PtrString(message.ParentMessageID),
		ChatID:          strconv.FormatInt(message.ChatID, 10),
		AuthorID:        strconv.FormatInt(message.AuthorID, 10),
		StickerID:       int64PtrString(message.StickerID),
		Sticker:         mapStickerPtr(message.Sticker),
		Media:           mapAttachments(message.Media),
		Files:           mapAttachments(message.Files),
		Reactions:       mapReactions(message.Reactions),
		MyReaction:      message.MyReaction,
		Type:            string(message.Type),
		IsActive:        message.IsActive,
		CreatedAt:       message.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:       message.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func mapMessageInput(req messageRequest) usecase.MessageInput {
	return usecase.MessageInput{
		Text:            req.Text,
		ParentMessageID: req.ParentMessageID,
		StickerID:       req.StickerID,
		Media:           mapAttachmentInputs(req.Media),
		Files:           mapAttachmentInputs(req.Files),
		Type:            model.MessageType(req.Type),
	}
}

func mapAttachmentInputs(items []attachmentRequest) []usecase.AttachmentInput {
	result := make([]usecase.AttachmentInput, 0, len(items))
	for _, item := range items {
		result = append(result, usecase.AttachmentInput{MediaID: item.MediaID})
	}
	return result
}

func mapAttachments(items []usecase.Attachment) []AttachmentResponse {
	result := make([]AttachmentResponse, 0, len(items))
	for _, item := range items {
		result = append(result, AttachmentResponse{
			ID:       strconv.FormatInt(item.ID, 10),
			Uid:      item.UID,
			Name:     item.Name,
			MimeType: item.MimeType,
			URL:      item.URL,
		})
	}
	return result
}

func mapStickerPack(pack usecase.StickerPack) StickerPackResponse {
	return StickerPackResponse{
		ID:        strconv.FormatInt(pack.ID, 10),
		Uid:       pack.UID,
		Title:     pack.Title,
		AuthorID:  int64PtrString(pack.AuthorID),
		CreatedAt: pack.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: pack.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func mapStickerPtr(sticker *usecase.Sticker) *StickerResponse {
	if sticker == nil {
		return nil
	}
	resp := mapSticker(*sticker)
	return &resp
}

func mapSticker(sticker usecase.Sticker) StickerResponse {
	return StickerResponse{
		ID:       strconv.FormatInt(sticker.ID, 10),
		Uid:      sticker.UID,
		PackID:   int64PtrString(sticker.PackID),
		MediaID:  int64PtrString(sticker.MediaID),
		MimeType: sticker.MimeType,
		URL:      sticker.URL,
	}
}

func mapReactions(items []usecase.ReactionSummary) []ReactionResponse {
	result := make([]ReactionResponse, 0, len(items))
	for _, item := range items {
		result = append(result, ReactionResponse{Type: item.Type, Count: item.Count})
	}
	return result
}

func int64PtrString(value *int64) *string {
	if value == nil {
		return nil
	}
	out := strconv.FormatInt(*value, 10)
	return &out
}

func timePtrString(value *time.Time) *string {
	if value == nil || value.IsZero() {
		return nil
	}
	out := value.Format(time.RFC3339Nano)
	return &out
}
