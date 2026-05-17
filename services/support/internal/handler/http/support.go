package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	models "github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/model"
	tickets "github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/websocket"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"go.uber.org/zap"
)

type SupportHandler struct {
	ticketService tickets.TicketService
	hub           *websocket.Hub
}

func NewSupportHandler(ticketService tickets.TicketService, hubs ...*websocket.Hub) *SupportHandler {
	var hub *websocket.Hub
	if len(hubs) > 0 {
		hub = hubs[0]
	}
	return &SupportHandler{
		ticketService: ticketService,
		hub:           hub,
	}
}

func (h *SupportHandler) RegisterRoutes(r chi.Router) {
	r.Post("/tickets", h.SendTicket)
	r.Get("/tickets", h.GetAllTickets)
	r.Get("/tickets/my", h.GetMyTickets)
	r.Get("/stats", h.GetStats)
	r.Get("/tickets/{ticketID}", h.GetTicket)
	r.Patch("/tickets/{ticketID}", h.UpdateTicket)
	r.Patch("/tickets/{ticketID}/status", h.UpdateTicketStatus)
	r.Get("/tickets/{ticketID}/messages", h.GetTicketMessages)
	r.Post("/tickets/{ticketID}/messages", h.SendTicketMessage)
	r.Patch("/tickets/{ticketID}/assign", h.AssignTicket)
	r.Patch("/tickets/{ticketID}/escalate", h.EscalateTicket)
	r.Post("/tickets/{ticketID}/rating", h.RateTicket)
}

func (h *SupportHandler) buildSupportTicketResponse(r *http.Request, ticket *models.SupportTicket) SupportTicketResponse {
	medias := h.ticketService.GetMediasByTicketID(r.Context(), ticket.ID)
	mediaResponse := make([]MediaRequestData, 0, len(medias))
	for _, media := range medias {
		mediaResponse = append(mediaResponse, MediaRequestData{
			MediaID:  media.ID,
			MediaURL: media.Link,
		})
	}

	return SupportTicketResponse{
		ID:              strconv.FormatInt(ticket.ID, 10),
		UID:             fmt.Sprintf("SUP-%d", ticket.ID),
		ProfileID:       strconv.FormatInt(ticket.ProfileID, 10),
		Login:           ticket.Login,
		Email:           ticket.Email,
		Category:        ticketCategoryToString(ticket.Category),
		Title:           ticket.Title,
		Description:     ticket.Description,
		Status:          ticketStatusToString(ticket.Status),
		Priority:        ticket.Priority,
		Line:            ticket.Line,
		AssignedAgentID: formatOptionalInt64(ticket.AssignedAgentID),
		Rating:          ticket.Rating,
		Media:           mediaResponse,
		CreatedAt:       ticket.CreatedAt,
		UpdatedAt:       ticket.UpdatedAt,
		ClosedAt:        ticket.ClosedAt,
	}
}

func (h *SupportHandler) buildSupportTicketResponses(r *http.Request, tickets []models.SupportTicket) []SupportTicketResponse {
	responses := make([]SupportTicketResponse, 0, len(tickets))
	for i := range tickets {
		responses = append(responses, h.buildSupportTicketResponse(r, &tickets[i]))
	}
	return responses
}

func (h *SupportHandler) buildMessageResponse(ctxRequest *http.Request, message models.SupportTicketMessage) SupportMessageResponse {
	authorName := ""
	profile, err := h.ticketService.GetUserProfileByProfileID(ctxRequest.Context(), message.AuthorID)
	if err == nil && profile != nil {
		authorName = strings.TrimSpace(profile.FirstName + " " + profile.LastName)
	}
	if authorName == "" {
		authorName = strconv.FormatInt(message.AuthorID, 10)
	}

	return SupportMessageResponse{
		ID:         strconv.FormatInt(message.ID, 10),
		TicketID:   strconv.FormatInt(message.TicketID, 10),
		Text:       message.Text,
		AuthorID:   strconv.FormatInt(message.AuthorID, 10),
		AuthorName: authorName,
		AuthorRole: message.AuthorRole,
		CreatedAt:  message.CreatedAt,
	}
}

func (h *SupportHandler) buildMessageResponses(r *http.Request, messages []models.SupportTicketMessage) []SupportMessageResponse {
	responses := make([]SupportMessageResponse, 0, len(messages))
	for _, message := range messages {
		responses = append(responses, h.buildMessageResponse(r, message))
	}
	return responses
}

func (h *SupportHandler) getCurrentProfile(w http.ResponseWriter, r *http.Request) (*models.Profile, bool) {
	log := logger.FromContext(r.Context())

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			utils.WriteError(w, "Unauthorised", http.StatusUnauthorized)
			return nil, false
		}
		profile, err := h.ticketService.GetProfileBySessionID(r.Context(), models.SessionID(cookie.Value))
		if err != nil {
			utils.WriteError(w, "Unauthorised", http.StatusUnauthorized)
			return nil, false
		}
		return profile, true
	}

	profile, err := h.ticketService.GetProfileByUserAccountID(r.Context(), userAccountID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return nil, false
		}
		log.Error("failed_to_get_profile", zap.Int64("userAccount_id", userAccountID), zap.Error(err))
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return nil, false
	}

	return profile, true
}

func (h *SupportHandler) getCurrentRole(w http.ResponseWriter, r *http.Request, profileID int64) (models.SupportRole, bool) {
	log := logger.FromContext(r.Context())
	role, err := h.ticketService.GetProfileRole(r.Context(), profileID)
	if err != nil {
		log.Error("failed_to_get_support_role", zap.Int64("profile_id", profileID), zap.Error(err))
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return models.SupportRoleUser, false
	}
	return role, true
}

func (h *SupportHandler) requireSupportAgent(w http.ResponseWriter, r *http.Request) (*models.Profile, models.SupportRole, bool) {
	profile, ok := h.getCurrentProfile(w, r)
	if !ok {
		return nil, models.SupportRoleUser, false
	}
	role, ok := h.getCurrentRole(w, r, profile.ID)
	if !ok {
		return nil, models.SupportRoleUser, false
	}
	if role != models.SupportRoleAdmin && role != models.SupportRoleSupportL1 && role != models.SupportRoleSupportL2 {
		utils.WriteError(w, xerrors.SupportForbidden.Error(), http.StatusForbidden)
		return nil, models.SupportRoleUser, false
	}
	return profile, role, true
}

func (h *SupportHandler) SendTicket(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	profile, ok := h.getCurrentProfile(w, r)
	if !ok {
		return
	}

	userAccount, err := h.ticketService.GetUserAccountByProfileID(r.Context(), profile.ID)
	if err != nil {
		utils.WriteError(w, err.Error(), http.StatusNotFound)
		return
	}

	var request SupportRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	request.Title = strings.TrimSpace(request.Title)
	request.Login = strings.TrimSpace(request.Login)
	request.Description = strings.TrimSpace(request.Description)
	if request.Title == "" || request.Description == "" || request.Login == "" {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}
	if request.Category.Value < models.CategoryBug || request.Category.Value > models.CategoryOther {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(request.Email)
	if email == "" && userAccount.Email != nil {
		email = *userAccount.Email
	}
	if email == "" {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	ticket := models.NewSupportTicket(profile.ID, request.Login, email, request.Category.Value, request.Title, request.Description)
	ticketID, err := h.ticketService.Save(r.Context(), ticket)
	if err != nil {
		log.Error("failed_to_save_support_ticket", zap.Int64("profile_id", profile.ID), zap.Error(err))
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	mediaResponse := make([]MediaRequestData, 0)
	if request.Medias != nil {
		mediaRefs := make([]tickets.MediaRef, 0, len(*request.Medias))
		mediaResponse = make([]MediaRequestData, 0, len(*request.Medias))
		for _, media := range *request.Medias {
			mediaRefs = append(mediaRefs, tickets.MediaRef{MediaID: media.MediaID, MediaURL: media.MediaURL})
			mediaResponse = append(mediaResponse, media)
		}
		mediaErrors := h.ticketService.AttachMedia(r.Context(), ticketID, mediaRefs)
		if len(mediaErrors.Errs) != 0 {
			status := http.StatusInternalServerError
			for _, attachmentErr := range mediaErrors.Errs {
				if errors.Is(attachmentErr.Err, xerrors.UnsupportedContentType) {
					status = http.StatusUnsupportedMediaType
					break
				}
			}
			utils.WriteError(w, "Can't attach media to support ticket", status)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(h.buildSupportTicketResponse(r, ticket))
}

func (h *SupportHandler) GetMyTickets(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile, ok := h.getCurrentProfile(w, r)
	if !ok {
		return
	}

	ticketsList, err := h.ticketService.GetByProfileID(r.Context(), profile.ID)
	if err != nil {
		log.Error("failed_to_get_support_tickets", zap.Int64("profile_id", profile.ID), zap.Error(err))
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.buildSupportTicketResponses(r, ticketsList))
}

func (h *SupportHandler) GetAllTickets(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	_, role, ok := h.requireSupportAgent(w, r)
	if !ok {
		return
	}

	filter, ok := parseTicketFilter(w, r)
	if !ok {
		return
	}

	ticketsList, err := h.ticketService.GetAll(r.Context(), role, filter)
	if err != nil {
		log.Error("failed_to_get_all_support_tickets", zap.Error(err))
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(SupportTicketListResponse{Tickets: h.buildSupportTicketResponses(r, ticketsList)})
}

func (h *SupportHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile, ok := h.getCurrentProfile(w, r)
	if !ok {
		return
	}
	role, ok := h.getCurrentRole(w, r, profile.ID)
	if !ok {
		return
	}

	ticketID, ok := parseTicketID(w, r)
	if !ok {
		return
	}

	ticket, err := h.ticketService.CanAccessTicket(r.Context(), ticketID, profile.ID, role)
	if err != nil {
		handleTicketAccessError(w, log, err, ticketID, profile.ID)
		return
	}
	if !roleCanWorkWithTicket(role, ticket) && ticket.ProfileID != profile.ID {
		utils.WriteError(w, xerrors.SupportForbidden.Error(), http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.buildSupportTicketResponse(r, ticket))
}

func (h *SupportHandler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile, ok := h.getCurrentProfile(w, r)
	if !ok {
		return
	}
	ticketID, ok := parseTicketID(w, r)
	if !ok {
		return
	}

	var request SupportUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	upd := tickets.TicketUpdate{Title: normalizeOptionalString(request.Title), Description: normalizeOptionalString(request.Description)}
	if request.Category != nil {
		category := request.Category.Value
		upd.Category = &category
	}
	if upd.IsEmpty() {
		utils.WriteError(w, xerrors.ErrNothingToUpdate.Error(), http.StatusBadRequest)
		return
	}
	if upd.Title != nil && *upd.Title == "" || upd.Description != nil && *upd.Description == "" || upd.Category != nil && (*upd.Category < models.CategoryBug || *upd.Category > models.CategoryOther) {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	ticket, err := h.ticketService.Update(r.Context(), ticketID, profile.ID, upd)
	if err != nil {
		handleTicketAccessError(w, log, err, ticketID, profile.ID)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.buildSupportTicketResponse(r, ticket))
}

func (h *SupportHandler) UpdateTicketStatus(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	_, role, ok := h.requireSupportAgent(w, r)
	if !ok {
		return
	}
	ticketID, ok := parseTicketID(w, r)
	if !ok {
		return
	}

	var request SupportStatusUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	if request.Status == nil {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	currentTicket, err := h.ticketService.GetByIDForAgent(r.Context(), ticketID)
	if err != nil {
		handleTicketAccessError(w, log, err, ticketID, 0)
		return
	}
	if !roleCanWorkWithTicket(role, currentTicket) {
		utils.WriteError(w, xerrors.SupportForbidden.Error(), http.StatusForbidden)
		return
	}

	ticket, err := h.ticketService.UpdateStatusByAgent(r.Context(), ticketID, request.Status.Value)
	if err != nil {
		if errors.Is(err, tickets.ErrInvalidTicketStatus) {
			utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
			return
		}
		handleTicketAccessError(w, log, err, ticketID, 0)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.buildSupportTicketResponse(r, ticket))
}

func (h *SupportHandler) AssignTicket(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile, role, ok := h.requireSupportAgent(w, r)
	if !ok {
		return
	}
	ticketID, ok := parseTicketID(w, r)
	if !ok {
		return
	}

	var request SupportAssignRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	agentID, err := strconv.ParseInt(strings.TrimSpace(request.AgentID), 10, 64)
	if err != nil || agentID <= 0 {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}
	if role != models.SupportRoleAdmin && agentID != profile.ID {
		utils.WriteError(w, xerrors.SupportForbidden.Error(), http.StatusForbidden)
		return
	}
	agentRole, ok := h.getCurrentRole(w, r, agentID)
	if !ok {
		return
	}
	if agentRole != models.SupportRoleAdmin && agentRole != models.SupportRoleSupportL1 && agentRole != models.SupportRoleSupportL2 {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	currentTicket, err := h.ticketService.GetByIDForAgent(r.Context(), ticketID)
	if err != nil {
		handleTicketAccessError(w, log, err, ticketID, profile.ID)
		return
	}
	if !roleCanWorkWithTicket(role, currentTicket) {
		utils.WriteError(w, xerrors.SupportForbidden.Error(), http.StatusForbidden)
		return
	}

	ticket, err := h.ticketService.Assign(r.Context(), ticketID, agentID)
	if err != nil {
		handleTicketAccessError(w, log, err, ticketID, profile.ID)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.buildSupportTicketResponse(r, ticket))
}

func (h *SupportHandler) EscalateTicket(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	_, role, ok := h.requireSupportAgent(w, r)
	if !ok {
		return
	}
	if role != models.SupportRoleAdmin && role != models.SupportRoleSupportL1 {
		utils.WriteError(w, xerrors.SupportForbidden.Error(), http.StatusForbidden)
		return
	}
	ticketID, ok := parseTicketID(w, r)
	if !ok {
		return
	}
	var request SupportEscalateRequest
	_ = json.NewDecoder(r.Body).Decode(&request)
	defer r.Body.Close()

	currentTicket, err := h.ticketService.GetByIDForAgent(r.Context(), ticketID)
	if err != nil {
		handleTicketAccessError(w, log, err, ticketID, 0)
		return
	}
	if currentTicket.Line != 1 {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	ticket, err := h.ticketService.Escalate(r.Context(), ticketID)
	if err != nil {
		handleTicketAccessError(w, log, err, ticketID, 0)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.buildSupportTicketResponse(r, ticket))
}

func (h *SupportHandler) RateTicket(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile, ok := h.getCurrentProfile(w, r)
	if !ok {
		return
	}
	ticketID, ok := parseTicketID(w, r)
	if !ok {
		return
	}

	var request SupportRatingRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	ticket, err := h.ticketService.Rate(r.Context(), ticketID, profile.ID, request.Rating)
	if err != nil {
		if errors.Is(err, tickets.ErrInvalidRating) {
			utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
			return
		}
		handleTicketAccessError(w, log, err, ticketID, profile.ID)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{"rating": *ticket.Rating})
}

func (h *SupportHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	if _, _, ok := h.requireSupportAgent(w, r); !ok {
		return
	}

	stats, err := h.ticketService.GetStats(r.Context())
	if err != nil {
		log.Error("failed_to_get_support_stats", zap.Error(err))
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func (h *SupportHandler) GetTicketMessages(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile, role, ticket, ok := h.requireTicketAccess(w, r)
	if !ok {
		return
	}
	if !roleCanWorkWithTicket(role, ticket) && ticket.ProfileID != profile.ID {
		utils.WriteError(w, xerrors.SupportForbidden.Error(), http.StatusForbidden)
		return
	}

	messages, err := h.ticketService.GetMessages(r.Context(), ticket.ID)
	if err != nil {
		log.Error("failed_to_get_support_ticket_messages", zap.Int64("ticket_id", ticket.ID), zap.Error(err))
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.buildMessageResponses(r, messages))
}

func (h *SupportHandler) SendTicketMessage(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	profile, role, ticket, ok := h.requireTicketAccess(w, r)
	if !ok {
		return
	}
	if !roleCanWorkWithTicket(role, ticket) && ticket.ProfileID != profile.ID {
		utils.WriteError(w, xerrors.SupportForbidden.Error(), http.StatusForbidden)
		return
	}

	var request SupportMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	if strings.TrimSpace(request.Text) == "" {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	message, err := h.ticketService.SaveMessage(r.Context(), ticket.ID, profile.ID, role, request.Text)
	if err != nil {
		log.Error("failed_to_save_support_ticket_message", zap.Int64("ticket_id", ticket.ID), zap.Error(err))
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}
	response := h.buildMessageResponse(r, *message)

	if h.hub != nil {
		payload, err := json.Marshal(response)
		if err == nil {
			h.hub.BroadcastToChat(supportRoomID(ticket.ID), payload)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *SupportHandler) HandleTicketWebSocket(w http.ResponseWriter, r *http.Request) {
	profile, role, ticket, ok := h.requireTicketAccess(w, r)
	if !ok {
		return
	}
	if !roleCanWorkWithTicket(role, ticket) && ticket.ProfileID != profile.ID {
		utils.WriteError(w, xerrors.SupportForbidden.Error(), http.StatusForbidden)
		return
	}
	if h.hub == nil {
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}
	if err := websocket.ServeWebSocket(h.hub, w, r, supportRoomID(ticket.ID), profile.ID); err != nil {
		utils.WriteError(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *SupportHandler) requireTicketAccess(w http.ResponseWriter, r *http.Request) (*models.Profile, models.SupportRole, *models.SupportTicket, bool) {
	log := logger.FromContext(r.Context())
	profile, ok := h.getCurrentProfile(w, r)
	if !ok {
		return nil, models.SupportRoleUser, nil, false
	}
	role, ok := h.getCurrentRole(w, r, profile.ID)
	if !ok {
		return nil, models.SupportRoleUser, nil, false
	}
	ticketID, ok := parseTicketID(w, r)
	if !ok {
		return nil, models.SupportRoleUser, nil, false
	}
	ticket, err := h.ticketService.CanAccessTicket(r.Context(), ticketID, profile.ID, role)
	if err != nil {
		handleTicketAccessError(w, log, err, ticketID, profile.ID)
		return nil, models.SupportRoleUser, nil, false
	}
	return profile, role, ticket, true
}

func parseTicketFilter(w http.ResponseWriter, r *http.Request) (tickets.TicketFilter, bool) {
	query := r.URL.Query()
	var filter tickets.TicketFilter
	if raw := query.Get("status"); raw != "" {
		status, err := parseTicketStatusString(raw)
		if err != nil {
			utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
			return filter, false
		}
		filter.Status = &status
	}
	if raw := query.Get("category"); raw != "" {
		category, err := parseTicketCategoryString(raw)
		if err != nil {
			utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
			return filter, false
		}
		filter.Category = &category
	}
	if raw := query.Get("line"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || (value != 1 && value != 2) {
			utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
			return filter, false
		}
		filter.Line = &value
	}
	if raw := query.Get("assignedAgentId"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
			return filter, false
		}
		filter.AssignedAgentID = &value
	}
	if raw := query.Get("assigned_agent_id"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 {
			utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
			return filter, false
		}
		filter.AssignedAgentID = &value
	}
	return filter, true
}

func parseTicketID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	ticketID, err := strconv.ParseInt(chi.URLParam(r, "ticketID"), 10, 64)
	if err != nil || ticketID <= 0 {
		utils.WriteError(w, xerrors.InvalidID, http.StatusBadRequest)
		return 0, false
	}
	return ticketID, true
}

func roleCanWorkWithTicket(role models.SupportRole, ticket *models.SupportTicket) bool {
	return role == models.SupportRoleAdmin ||
		(role == models.SupportRoleSupportL1 && ticket.Line == 1) ||
		(role == models.SupportRoleSupportL2 && ticket.Line == 2)
}

func supportRoomID(ticketID int64) string {
	return "support:" + strconv.FormatInt(ticketID, 10)
}

func handleTicketAccessError(w http.ResponseWriter, log *zap.Logger, err error, ticketID, profileID int64) {
	if errors.Is(err, xerrors.SupportTicketNotFound) {
		utils.WriteError(w, xerrors.SupportTicketNotFound.Error(), http.StatusNotFound)
		return
	}
	if errors.Is(err, tickets.ErrForbidden) {
		utils.WriteError(w, xerrors.SupportForbidden.Error(), http.StatusForbidden)
		return
	}
	if log != nil {
		log.Error("failed_to_handle_support_ticket", zap.Int64("ticket_id", ticketID), zap.Int64("profile_id", profileID), zap.Error(err))
	}
	utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	return &normalized
}

func parseTicketStatusJSON(data []byte) (models.TicketStatus, error) {
	var rawString string
	if err := json.Unmarshal(data, &rawString); err == nil {
		return parseTicketStatusString(rawString)
	}

	var rawInt int
	if err := json.Unmarshal(data, &rawInt); err == nil {
		status := models.TicketStatus(rawInt)
		if isValidTicketStatus(status) {
			return status, nil
		}
	}

	return models.TicketStatusOpen, errors.New("invalid ticket status")
}

func parseTicketStatusString(raw string) (models.TicketStatus, error) {
	raw = strings.TrimSpace(raw)
	switch strings.ToLower(raw) {
	case "open":
		return models.TicketStatusOpen, nil
	case "in_progress":
		return models.TicketStatusInProgress, nil
	case "waiting_user":
		return models.TicketStatusWaitingUser, nil
	case "closed":
		return models.TicketStatusClosed, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return models.TicketStatusOpen, err
	}
	status := models.TicketStatus(value)
	if !isValidTicketStatus(status) {
		return models.TicketStatusOpen, errors.New("invalid ticket status")
	}
	return status, nil
}

func parseTicketCategoryJSON(data []byte) (models.TicketCategory, error) {
	var rawString string
	if err := json.Unmarshal(data, &rawString); err == nil {
		return parseTicketCategoryString(rawString)
	}

	var rawInt int
	if err := json.Unmarshal(data, &rawInt); err == nil {
		category := models.TicketCategory(rawInt)
		if isValidTicketCategory(category) {
			return category, nil
		}
	}

	return models.CategoryBug, errors.New("invalid ticket category")
}

func parseTicketCategoryString(raw string) (models.TicketCategory, error) {
	raw = strings.TrimSpace(raw)
	switch strings.ToLower(raw) {
	case "bug":
		return models.CategoryBug, nil
	case "feature_request":
		return models.CategoryFeatureRequest, nil
	case "complaint":
		return models.CotegoryComplaint, nil
	case "question":
		return models.CategoryQuestion, nil
	case "other":
		return models.CategoryOther, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return models.CategoryBug, err
	}
	category := models.TicketCategory(value)
	if !isValidTicketCategory(category) {
		return models.CategoryBug, errors.New("invalid ticket category")
	}
	return category, nil
}

func ticketStatusToString(status models.TicketStatus) string {
	switch status {
	case models.TicketStatusOpen:
		return "open"
	case models.TicketStatusInProgress:
		return "in_progress"
	case models.TicketStatusWaitingUser:
		return "waiting_user"
	case models.TicketStatusClosed:
		return "closed"
	default:
		return "open"
	}
}

func ticketCategoryToString(category models.TicketCategory) string {
	switch category {
	case models.CategoryBug:
		return "bug"
	case models.CategoryFeatureRequest:
		return "feature_request"
	case models.CotegoryComplaint:
		return "complaint"
	case models.CategoryQuestion:
		return "question"
	case models.CategoryOther:
		return "other"
	default:
		return "other"
	}
}

func isValidTicketStatus(status models.TicketStatus) bool {
	return status >= models.TicketStatusOpen && status <= models.TicketStatusClosed
}

func isValidTicketCategory(category models.TicketCategory) bool {
	return category >= models.CategoryBug && category <= models.CategoryOther
}

func formatOptionalInt64(value *int64) *string {
	if value == nil {
		return nil
	}
	formatted := strconv.FormatInt(*value, 10)
	return &formatted
}
