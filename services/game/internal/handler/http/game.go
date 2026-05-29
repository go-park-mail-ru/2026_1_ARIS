package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/game/internal/websocket"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
)

type Handler struct {
	game GameService
	hub  *websocket.Hub
}

const waitingRoomDisconnectGrace = 3 * time.Second

func New(game GameService, hub *websocket.Hub) *Handler {
	h := &Handler{game: game, hub: hub}
	game.SetNotifier(h.broadcastRoom)
	return h
}

func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		r.Get("/games/rooms", h.ListRooms)
		r.Post("/games/rooms", h.CreateRoom)
		r.Post("/games/rooms/join", h.JoinRoom)
		r.Get("/games/rooms/{roomID}", h.GetRoom)
		r.Delete("/games/rooms/{roomID}", h.DisbandRoom)
		r.Delete("/games/rooms/{roomID}/members/me", h.LeaveRoom)
		r.Delete("/games/rooms/{roomID}/members/{profileID}", h.KickPlayer)
		r.Patch("/games/rooms/{roomID}/ready", h.SetReady)
		r.Patch("/games/rooms/{roomID}/replay", h.SetReplayReady)
		r.Patch("/games/rooms/{roomID}/password", h.UpdateRoomPassword)
		r.Patch("/games/rooms/{roomID}/title", h.UpdateRoomTitle)
		r.Patch("/games/rooms/{roomID}/ranked", h.UpdateRoomRanked)
		r.Patch("/games/rooms/{roomID}/admin", h.AssignAdmin)
		r.Post("/games/rooms/{roomID}/start", h.StartRoom)
		r.Post("/games/rooms/{roomID}/answers", h.SubmitAnswer)
		r.Post("/games/rooms/{roomID}/pause", h.PauseRoom)
		r.Post("/games/rooms/{roomID}/force-resume", h.ForceResumeRoom)
		r.Get("/games/rooms/{roomID}/messages", h.ListRoomMessages)
		r.Post("/games/rooms/{roomID}/messages", h.SendRoomMessage)
		r.Get("/games/ratings/{gameType}/leaderboard", h.Leaderboard)
		r.Get("/games/history", h.History)
		r.Get("/games/stats", h.Stats)
		r.Get("/games/questions", h.ListQuestions)
		r.Post("/games/questions", h.CreateQuestion)
		r.Patch("/games/questions/{questionID}", h.UpdateQuestion)
		r.Delete("/games/questions/{questionID}", h.DeleteQuestion)
	})
}

func (h *Handler) RegisterWebSocketRoute(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		r.Get("/ws/games/{roomID}", h.HandleWebSocket)
	})
}

func (h *Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	var req createRoomRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	room, err := h.game.CreateRoom(r.Context(), userID, usecase.CreateRoomInput{
		Title:            req.Title,
		GameType:         req.GameType,
		MaxPlayers:       req.MaxPlayers,
		Password:         req.Password,
		IsRanked:         req.IsRanked,
		QuestionCount:    req.QuestionCount,
		AnswerTimeoutSec: req.AnswerTimeoutSec,
	})
	if err != nil {
		if errors.Is(err, usecase.ErrActiveCreatedRoom) {
			utils.WriteJSON(w, http.StatusConflict, map[string]any{
				"error": "У вас уже есть своя созданная комната.",
				"room":  mapRoom(room, requestLanguage(r)),
			})
			return
		}
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, mapRoom(room, requestLanguage(r)))
}

func (h *Handler) JoinRoom(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	var req joinRoomRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	room, err := h.game.JoinRoom(r.Context(), userID, req.InviteCode, req.RoomID, req.Password)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapRoom(room, requestLanguage(r)))
}

func (h *Handler) ListRooms(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	rooms, err := h.game.ListRooms(r.Context(), userID, queryInt(r, "limit", 50), queryInt(r, "offset", 0))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	resp := make([]roomResponse, 0, len(rooms))
	for _, room := range rooms {
		resp = append(resp, mapRoom(room, requestLanguage(r)))
	}
	utils.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetRoom(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	room, err := h.game.GetRoom(r.Context(), userID, roomID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapRoom(room, requestLanguage(r)))
}

func (h *Handler) DisbandRoom(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	if err := h.game.DisbandRoom(r.Context(), userID, roomID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) LeaveRoom(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	if err := h.game.LeaveRoom(r.Context(), userID, roomID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) KickPlayer(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	profileID, err := strconv.ParseInt(chi.URLParam(r, "profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		writeError(w, "invalid input", http.StatusBadRequest)
		return
	}
	if err := h.game.KickPlayer(r.Context(), userID, roomID, profileID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetReady(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	var req readyRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	if err := h.game.SetReady(r.Context(), userID, roomID, req.IsReady); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetReplayReady(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	var req readyRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	room, err := h.game.SetReplayReady(r.Context(), userID, roomID, req.IsReady)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapRoom(room, requestLanguage(r)))
}

func (h *Handler) UpdateRoomPassword(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	var req passwordRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	if err := h.game.UpdateRoomPassword(r.Context(), userID, roomID, req.Password); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpdateRoomTitle(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	var req titleRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	if err := h.game.UpdateRoomTitle(r.Context(), userID, roomID, req.Title); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpdateRoomRanked(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	var req rankedRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	if err := h.game.UpdateRoomRanked(r.Context(), userID, roomID, req.IsRanked); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AssignAdmin(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	var req adminRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	profileID, err := strconv.ParseInt(req.ProfileID, 10, 64)
	if err != nil || profileID <= 0 {
		writeError(w, "invalid input", http.StatusBadRequest)
		return
	}
	if err := h.game.AssignAdmin(r.Context(), userID, roomID, profileID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) StartRoom(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	room, err := h.game.StartRoom(r.Context(), userID, roomID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapRoom(room, requestLanguage(r)))
}

func (h *Handler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	var req submitAnswerRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	room, err := h.game.SubmitAnswer(r.Context(), userID, roomID, req.Answer)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapRoom(room, requestLanguage(r)))
}

func (h *Handler) PauseRoom(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	room, err := h.game.PauseRoom(r.Context(), userID, roomID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapRoom(room, requestLanguage(r)))
}

func (h *Handler) ForceResumeRoom(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	room, err := h.game.ForceResumeRoom(r.Context(), userID, roomID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapRoom(room, requestLanguage(r)))
}

func (h *Handler) ListRoomMessages(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	messages, err := h.game.ListRoomMessages(r.Context(), userID, roomID, queryInt(r, "limit", 100), queryInt(r, "offset", 0))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	resp := make([]roomMessageResponse, 0, len(messages))
	for _, message := range messages {
		resp = append(resp, mapRoomMessage(message))
	}
	utils.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) SendRoomMessage(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	var req roomMessageRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	message, err := h.game.SendRoomMessage(r.Context(), userID, roomID, req.Text)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	resp := mapRoomMessage(message)
	h.broadcastRoomMessage(roomID, resp)
	utils.WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	items, err := h.game.History(r.Context(), userID, queryInt(r, "limit", 50), queryInt(r, "offset", 0))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	resp := make([]historyResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, historyResponse{Room: mapRoom(item.Room, requestLanguage(r)), MyScore: item.MyScore, OpponentScore: item.OpponentScore})
	}
	utils.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	stats, err := h.game.Stats(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, statsResponse{Played: stats.Played, Won: stats.Won, Lost: stats.Lost, Drawn: stats.Drawn})
}

func (h *Handler) Leaderboard(w http.ResponseWriter, r *http.Request) {
	board, err := h.game.Leaderboard(r.Context(), chi.URLParam(r, "gameType"), queryInt(r, "limit", 50), queryInt(r, "offset", 0))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapLeaderboard(board))
}

func (h *Handler) ListQuestions(w http.ResponseWriter, r *http.Request) {
	items, err := h.game.ListQuestions(
		r.Context(),
		r.URL.Query().Get("gameType"),
		r.URL.Query().Get("includeInactive") == "true",
		queryInt(r, "limit", 100),
		queryInt(r, "offset", 0),
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	resp := make([]questionResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, mapQuestion(item))
	}
	utils.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	var req questionRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	question, err := h.game.CreateQuestion(r.Context(), userID, mapQuestionInput(req))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, mapQuestion(question))
}

func (h *Handler) UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	questionID, ok := parsePathID(w, chi.URLParam(r, "questionID"), "неверный ID вопроса")
	if !ok {
		return
	}
	var req questionRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	question, err := h.game.UpdateQuestion(r.Context(), userID, questionID, mapQuestionInput(req))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapQuestion(question))
}

func (h *Handler) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	questionID, ok := parsePathID(w, chi.URLParam(r, "questionID"), "неверный ID вопроса")
	if !ok {
		return
	}
	if err := h.game.DeleteQuestion(r.Context(), userID, questionID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID, roomID, ok := h.userAndRoomID(w, r)
	if !ok {
		return
	}
	room, err := h.game.GetRoom(r.Context(), userID, roomID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	_ = h.game.TouchWaitingRoomMember(r.Context(), userID, roomID)
	language := requestLanguage(r)
	initial, _ := utils.MarshalJSON(socketEvent{Type: "room_state", Room: ptr(mapRoom(room, language))})
	if err := websocket.Serve(
		h.hub,
		w,
		r,
		strconv.FormatInt(roomID, 10),
		userID,
		language,
		initial,
		h.handleSocketMessage,
		h.handleSocketDisconnect,
		h.handleSocketHeartbeat,
	); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
	}
}

func (h *Handler) handleSocketMessage(ctx context.Context, roomIDRaw string, userID int64, payload []byte) ([]byte, error) {
	roomID, err := strconv.ParseInt(roomIDRaw, 10, 64)
	if err != nil || roomID <= 0 {
		return nil, usecase.ErrInvalidInput
	}
	var msg socketMessage
	if err := utils.UnmarshalJSON(payload, &msg); err != nil {
		return marshalEvent(socketEvent{Type: "error", Error: "неверный формат сообщения"}), nil
	}
	if msg.Type == "room_message" || msg.Type == "room_chat_message" {
		message, err := h.game.SendRoomMessage(ctx, userID, roomID, msg.Text)
		if err != nil {
			return marshalEvent(socketEvent{Type: "error", Error: serviceErrorMessage(err)}), nil
		}
		h.broadcastRoomMessage(roomID, mapRoomMessage(message))
		return nil, nil
	}
	if msg.Type == "pause_game" {
		if _, err := h.game.PauseRoom(ctx, userID, roomID); err != nil {
			return marshalEvent(socketEvent{Type: "error", Error: serviceErrorMessage(err)}), nil
		}
		return nil, nil
	}
	if msg.Type == "force_resume" {
		if _, err := h.game.ForceResumeRoom(ctx, userID, roomID); err != nil {
			return marshalEvent(socketEvent{Type: "error", Error: serviceErrorMessage(err)}), nil
		}
		return nil, nil
	}
	if msg.Type == "replay_ready" {
		if _, err := h.game.SetReplayReady(ctx, userID, roomID, msg.IsReady); err != nil {
			return marshalEvent(socketEvent{Type: "error", Error: serviceErrorMessage(err)}), nil
		}
		return nil, nil
	}
	if msg.Type != "" && msg.Type != "submit_answer" {
		return marshalEvent(socketEvent{Type: "error", Error: "неподдерживаемый тип сообщения"}), nil
	}
	if _, err := h.game.SubmitAnswer(ctx, userID, roomID, msg.Answer); err != nil {
		return marshalEvent(socketEvent{Type: "error", Error: serviceErrorMessage(err)}), nil
	}
	return nil, nil
}

func (h *Handler) handleSocketDisconnect(ctx context.Context, roomIDRaw string, userID int64) {
	roomID, err := strconv.ParseInt(roomIDRaw, 10, 64)
	if err != nil || roomID <= 0 {
		return
	}
	go func() {
		time.Sleep(waitingRoomDisconnectGrace)
		if h.hub != nil && h.hub.HasUser(roomIDRaw, userID) {
			return
		}
		disconnectCtx, cancel := context.WithTimeout(context.Background(), waitingRoomDisconnectGrace)
		defer cancel()
		if room, err := h.game.GetRoom(disconnectCtx, userID, roomID); err == nil {
			h.broadcastDisconnectRemovalMessage(room.ID, disconnectingPlayer(room))
		}
		_ = h.game.LeaveWaitingRoomOnDisconnect(disconnectCtx, userID, roomID)
	}()
}

func (h *Handler) handleSocketHeartbeat(ctx context.Context, roomIDRaw string, userID int64) {
	roomID, err := strconv.ParseInt(roomIDRaw, 10, 64)
	if err != nil || roomID <= 0 {
		return
	}
	_ = h.game.TouchWaitingRoomMember(ctx, userID, roomID)
}

func (h *Handler) broadcastRoom(ctx context.Context, roomID int64) {
	if h.hub == nil {
		return
	}
	h.hub.BroadcastRoomFunc(strconv.FormatInt(roomID, 10), func(userID int64, language string) []byte {
		room, err := h.game.GetRoom(ctx, userID, roomID)
		if err != nil {
			return marshalEvent(socketEvent{Type: "room_updated"})
		}
		return marshalEvent(socketEvent{Type: "room_state", Room: ptr(mapRoom(room, language))})
	})
}

func (h *Handler) broadcastRoomMessage(roomID int64, message roomMessageResponse) {
	if h.hub == nil {
		return
	}
	h.hub.BroadcastToRoom(strconv.FormatInt(roomID, 10), marshalEvent(socketEvent{Type: "room_message", Message: ptr(message)}))
}

func (h *Handler) broadcastDisconnectRemovalMessage(roomID int64, player usecase.Player) {
	if h.hub == nil || player.ProfileID <= 0 {
		return
	}
	now := time.Now()
	playerName := playerDisplayName(player)
	h.broadcastRoomMessage(roomID, roomMessageResponse{
		ID:              fmt.Sprintf("system:disconnect:%d:%d:%d", roomID, player.ProfileID, now.UnixNano()),
		RoomID:          int64String(roomID),
		AuthorName:      "Сервер",
		AuthorFirstName: "Сервер",
		AuthorUsername:  "server",
		Text:            fmt.Sprintf("Соединение с пользователем %s потеряно. Пользователь удален из комнаты.", playerName),
		CreatedAt:       now.Format(time.RFC3339Nano),
	})
}

func disconnectingPlayer(room usecase.Room) usecase.Player {
	for _, player := range room.Players {
		if player.IsMe {
			return player
		}
	}
	return usecase.Player{}
}

func playerDisplayName(player usecase.Player) string {
	if player.Name != "" {
		return player.Name
	}
	name := strings.TrimSpace(player.FirstName + " " + player.LastName)
	if name != "" {
		return name
	}
	if player.Username != "" {
		return player.Username
	}
	return "игроком"
}

func (h *Handler) userAndRoomID(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	userID, ok := userIDFromContext(w, r)
	if !ok {
		return 0, 0, false
	}
	roomID, ok := parsePathID(w, chi.URLParam(r, "roomID"), "неверный ID комнаты")
	return userID, roomID, ok
}

func userIDFromContext(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := r.Context().Value("user_id").(int64)
	if !ok || userID <= 0 {
		writeError(w, "не авторизован", http.StatusUnauthorized)
		return 0, false
	}
	return userID, true
}

func parsePathID(w http.ResponseWriter, raw string, message string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, message, http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func queryInt(r *http.Request, name string, fallback int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidInput), errors.Is(err, usecase.ErrAlreadyAnswered), errors.Is(err, usecase.ErrAlreadyStarted), errors.Is(err, usecase.ErrRoomFull), errors.Is(err, usecase.ErrRoomTitleTaken), errors.Is(err, usecase.ErrGamePaused), errors.Is(err, usecase.ErrPauseAlreadyUsed):
		writeError(w, serviceErrorMessage(err), http.StatusBadRequest)
	case errors.Is(err, usecase.ErrActiveCreatedRoom):
		writeError(w, "У вас уже есть своя созданная комната.", http.StatusConflict)
	case errors.Is(err, usecase.ErrForbidden):
		writeError(w, "доступ запрещён", http.StatusForbidden)
	case errors.Is(err, usecase.ErrNotFound):
		writeError(w, "не найдено", http.StatusNotFound)
	default:
		writeError(w, "internal server error", http.StatusInternalServerError)
	}
}

func serviceErrorMessage(err error) string {
	switch {
	case errors.Is(err, usecase.ErrAlreadyAnswered):
		return "ответ уже отправлен"
	case errors.Is(err, usecase.ErrAlreadyStarted):
		return "игра уже началась"
	case errors.Is(err, usecase.ErrRoomFull):
		return "комната заполнена"
	case errors.Is(err, usecase.ErrRoomTitleTaken):
		return "Комната с таким названием уже существует"
	case errors.Is(err, usecase.ErrGamePaused):
		return "игра на паузе"
	case errors.Is(err, usecase.ErrPauseAlreadyUsed):
		return "пауза уже использована"
	case errors.Is(err, usecase.ErrActiveCreatedRoom):
		return "У вас уже есть своя созданная комната."
	case errors.Is(err, usecase.ErrInvalidInput):
		return "неверные данные"
	case errors.Is(err, usecase.ErrForbidden):
		return "доступ запрещён"
	case errors.Is(err, usecase.ErrNotFound):
		return "не найдено"
	default:
		return "internal server error"
	}
}

func writeError(w http.ResponseWriter, message string, status int) {
	utils.WriteJSON(w, status, errorResponse{Error: message})
}

func marshalEvent(event socketEvent) []byte {
	data, _ := utils.MarshalJSON(event)
	return data
}

func requestLanguage(r *http.Request) string {
	for _, value := range []string{
		r.URL.Query().Get("lang"),
		r.URL.Query().Get("language"),
		r.Header.Get("X-Interface-Language"),
		r.Header.Get("Accept-Language"),
	} {
		if language := normalizeLanguage(value); language != "" {
			return language
		}
	}
	return "ru"
}

func normalizeLanguage(value string) string {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(strings.ToLower(part))
		switch {
		case strings.HasPrefix(part, "en"):
			return "en"
		case strings.HasPrefix(part, "ru"):
			return "ru"
		}
	}
	return ""
}

func localizedValue(text usecase.LocalizedText, language string) string {
	if normalizeLanguage(language) == "en" && strings.TrimSpace(text.EN) != "" {
		return text.EN
	}
	if strings.TrimSpace(text.RU) != "" {
		return text.RU
	}
	return text.EN
}

func mapQuestionInput(req questionRequest) usecase.QuestionInput {
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	return usecase.QuestionInput{
		GameType:      req.GameType,
		Text:          usecase.LocalizedText{RU: req.Text.RU, EN: req.Text.EN},
		CorrectAnswer: req.CorrectAnswer,
		IsActive:      active,
	}
}

func mapRoom(room usecase.Room, language string) roomResponse {
	resp := roomResponse{
		ID:                      int64String(room.ID),
		Title:                   room.Title,
		InviteCode:              room.InviteCode,
		GameType:                room.GameType,
		Status:                  room.Status,
		CreatedByProfileID:      int64String(room.CreatedByProfileID),
		WinnerProfileID:         int64PtrString(room.WinnerProfileID),
		MaxPlayers:              room.MaxPlayers,
		HasPassword:             room.HasPassword,
		Password:                room.Password,
		IsRanked:                room.IsRanked,
		QuestionCount:           room.QuestionCount,
		AnswerTimeoutSec:        room.AnswerTimeoutSec,
		Creator:                 mapPlayer(room.Creator),
		CurrentQuestionIndex:    room.CurrentQuestionIndex,
		NextQuestionAt:          room.NextQuestionAt,
		PausedByProfileID:       int64PtrString(room.PausedByProfileID),
		PauseStartedAt:          room.PauseStartedAt,
		PauseUntilAt:            room.PauseUntilAt,
		PauseForceVotes:         room.PauseForceVotes,
		PauseForceVotesRequired: room.PauseForceVotesRequired,
		Players:                 mapPlayers(room.Players),
		Questions:               mapRoomQuestions(room.Questions, language),
		RatingChanges:           mapRatingChanges(room.RatingChanges),
		ProfileStats:            statsResponse{Played: room.ProfileStats.Played, Won: room.ProfileStats.Won, Lost: room.ProfileStats.Lost, Drawn: room.ProfileStats.Drawn},
		CreatedAt:               room.CreatedAt,
		UpdatedAt:               room.UpdatedAt,
		FinishedAt:              room.FinishedAt,
	}
	if room.CurrentQuestion != nil {
		resp.CurrentQuestion = &currentQuestionResponse{
			Position:    room.CurrentQuestion.Position,
			ID:          int64String(room.CurrentQuestion.ID),
			Text:        localizedValue(room.CurrentQuestion.Text, language),
			StartedAt:   room.CurrentQuestion.StartedAt,
			DeadlineAt:  room.CurrentQuestion.DeadlineAt,
			HasAnswered: room.CurrentQuestion.HasAnswered,
		}
	}
	return resp
}

func mapPlayers(items []usecase.Player) []playerResponse {
	result := make([]playerResponse, 0, len(items))
	for _, item := range items {
		result = append(result, mapPlayer(item))
	}
	return result
}

func mapPlayer(item usecase.Player) playerResponse {
	return playerResponse{
		ProfileID:            int64String(item.ProfileID),
		UserAccountID:        int64String(item.UserAccountID),
		Name:                 item.Name,
		Username:             item.Username,
		FirstName:            item.FirstName,
		LastName:             item.LastName,
		AvatarID:             item.AvatarID,
		Score:                item.Score,
		IsReady:              item.IsReady,
		HasAnswered:          item.HasAnswered,
		PauseUsed:            item.PauseUsed,
		ForceResumeRequested: item.ForceResumeRequested,
		IsMe:                 item.IsMe,
	}
}

func mapRoomQuestions(items []usecase.RoomQuestion, language string) []roomQuestionResponse {
	result := make([]roomQuestionResponse, 0, len(items))
	for _, item := range items {
		result = append(result, roomQuestionResponse{
			Position:        item.Position,
			Status:          item.Status,
			Question:        mapRoomQuestionPayload(item.Question, language),
			WinnerProfileID: int64PtrString(item.WinnerProfileID),
			StartedAt:       item.StartedAt,
			DeadlineAt:      item.DeadlineAt,
			CompletedAt:     item.CompletedAt,
			Answers:         mapAnswers(item.Answers),
		})
	}
	return result
}

func mapQuestion(item usecase.Question) questionResponse {
	return questionResponse{
		ID:            int64String(item.ID),
		Text:          localizedTextResponse{RU: item.Text.RU, EN: item.Text.EN},
		CorrectAnswer: item.CorrectAnswer,
		IsActive:      item.IsActive,
	}
}

func mapRoomQuestionPayload(item usecase.Question, language string) roomQuestionPayloadResponse {
	return roomQuestionPayloadResponse{
		ID:            int64String(item.ID),
		Text:          localizedValue(item.Text, language),
		CorrectAnswer: item.CorrectAnswer,
		IsActive:      item.IsActive,
	}
}

func mapAnswers(items []usecase.Answer) []answerResponse {
	result := make([]answerResponse, 0, len(items))
	for _, item := range items {
		result = append(result, answerResponse{
			ProfileID:      int64String(item.ProfileID),
			Answer:         item.Answer,
			Distance:       item.Distance,
			AnsweredAt:     item.AnsweredAt,
			ResponseTimeMs: item.ResponseTimeMs,
		})
	}
	return result
}

func mapRatingChanges(items []usecase.RatingChange) []ratingChangeResponse {
	result := make([]ratingChangeResponse, 0, len(items))
	for _, item := range items {
		result = append(result, ratingChangeResponse{
			ProfileID:    int64String(item.ProfileID),
			Score:        item.Score,
			Place:        item.Place,
			BeforeRating: item.BeforeRating,
			AfterRating:  item.AfterRating,
			RatingDelta:  item.RatingDelta,
			RatingWeight: item.RatingWeight,
			SeasonNumber: item.SeasonNumber,
			SeasonTitle:  item.SeasonTitle,
		})
	}
	return result
}

func mapLeaderboard(board usecase.Leaderboard) leaderboardResponse {
	entries := make([]leaderboardEntryResponse, 0, len(board.Entries))
	for _, item := range board.Entries {
		entries = append(entries, leaderboardEntryResponse{
			Rank:        item.Rank,
			ProfileID:   int64String(item.ProfileID),
			Player:      mapPlayer(item.Player),
			Rating:      item.Rating,
			GamesPlayed: item.GamesPlayed,
			Wins:        item.Wins,
			Draws:       item.Draws,
		})
	}
	return leaderboardResponse{
		GameType: board.GameType,
		Season: ratingSeasonResponse{
			SeasonNumber: board.Season.SeasonNumber,
			Title:        board.Season.Title,
			StartsAt:     board.Season.StartsAt,
			EndsAt:       board.Season.EndsAt,
		},
		Entries: entries,
	}
}

func mapRoomMessage(item usecase.RoomMessage) roomMessageResponse {
	return roomMessageResponse{
		ID:                  int64String(item.ID),
		RoomID:              int64String(item.RoomID),
		AuthorProfileID:     int64String(item.ProfileID),
		AuthorUserAccountID: int64String(item.Author.UserAccountID),
		AuthorName:          item.Author.Name,
		AuthorFirstName:     item.Author.FirstName,
		AuthorLastName:      item.Author.LastName,
		AuthorUsername:      item.Author.Username,
		AuthorAvatarID:      item.Author.AvatarID,
		Text:                item.Text,
		CreatedAt:           item.CreatedAt,
	}
}

func int64String(value int64) string {
	if value <= 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}

func int64PtrString(value *int64) *string {
	if value == nil {
		return nil
	}
	out := strconv.FormatInt(*value, 10)
	return &out
}

func ptr[T any](value T) *T {
	return &value
}
