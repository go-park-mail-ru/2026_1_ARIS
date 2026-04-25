package support

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session"
	tickets "github.com/go-park-mail-ru/2026_1_ARIS/internal/service/support"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/websocket"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"go.uber.org/zap"
)

type MediaRequestData struct {
	MediaID  int64  `json:"mediaID"`
	MediaURL string `json:"mediaURL"`
}

type SupportRequest struct {
	Category    models.TicketCategory `json:"category"`
	Title       string                `json:"title"`
	Login       string                `json:"login"`
	Email       string                `json:"email"`
	Description string                `json:"description"`
	Medias      *[]MediaRequestData   `json:"media"`
}

type SupportHandler struct {
	sessionService session.SessionService
	userService    user.UserService
	ticketService  tickets.TicketService
	hub            *websocket.Hub
}

func NewSupportHandler(sessionService session.SessionService, userService user.UserService, ticketService tickets.TicketService, hubs ...*websocket.Hub) *SupportHandler {
	var hub *websocket.Hub
	if len(hubs) > 0 {
		hub = hubs[0]
	}
	return &SupportHandler{
		sessionService: sessionService,
		userService:    userService,
		ticketService:  ticketService,
		hub:            hub,
	}
}

type SupportResponse struct {
	ID     int64               `json:"id"`
	Login  string              `json:"login"`
	Status models.TicketStatus `json:"status"`
	Media  []MediaRequestData  `json:"media,omitempty"`
}

type SupportTicketResponse struct {
	ID              int64                 `json:"id"`
	UID             string                `json:"uid"`
	ProfileID       int64                 `json:"profileID"`
	Login           string                `json:"login"`
	Email           string                `json:"email"`
	Category        models.TicketCategory `json:"category"`
	Title           string                `json:"title"`
	Description     string                `json:"description"`
	Status          models.TicketStatus   `json:"status"`
	Priority        models.TicketPriority `json:"priority"`
	Line            int                   `json:"line"`
	AssignedAgentID *int64                `json:"assignedAgentId"`
	Rating          *int                  `json:"rating"`
	Media           []MediaRequestData    `json:"media"`
	CreatedAt       time.Time             `json:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt"`
	ClosedAt        *time.Time            `json:"closedAt,omitempty"`
}

type SupportTicketListResponse struct {
	Tickets []SupportTicketResponse `json:"tickets"`
}

type SupportUpdateRequest struct {
	Category    *models.TicketCategory `json:"category"`
	Title       *string                `json:"title"`
	Description *string                `json:"description"`
}

type SupportStatusUpdateRequest struct {
	Status *models.TicketStatus `json:"status"`
}

type SupportAssignRequest struct {
	AgentID string `json:"agentId"`
}

type SupportEscalateRequest struct {
	Reason string `json:"reason,omitempty"`
}

type SupportRatingRequest struct {
	Rating int `json:"rating"`
}

type SupportMessageRequest struct {
	Text string `json:"text"`
}

type SupportMessageResponse struct {
	ID         int64              `json:"id"`
	TicketID   int64              `json:"ticketId"`
	Text       string             `json:"text"`
	AuthorID   int64              `json:"authorId"`
	AuthorName string             `json:"authorName"`
	AuthorRole models.SupportRole `json:"authorRole"`
	CreatedAt  time.Time          `json:"createdAt"`
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
		ID:              ticket.ID,
		UID:             ticket.Uid.String(),
		ProfileID:       ticket.ProfileID,
		Login:           ticket.Login,
		Email:           ticket.Email,
		Category:        ticket.Category,
		Title:           ticket.Title,
		Description:     ticket.Description,
		Status:          ticket.Status,
		Priority:        ticket.Priority,
		Line:            ticket.Line,
		AssignedAgentID: ticket.AssignedAgentID,
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
	profile, err := h.userService.GetUserProfileByProfileID(ctxRequest.Context(), message.AuthorID)
	if err == nil && profile != nil {
		authorName = strings.TrimSpace(profile.FirstName + " " + profile.LastName)
	}
	if authorName == "" {
		authorName = strconv.FormatInt(message.AuthorID, 10)
	}

	return SupportMessageResponse{
		ID:         message.ID,
		TicketID:   message.TicketID,
		Text:       message.Text,
		AuthorID:   message.AuthorID,
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
		session, err := h.sessionService.Get(r.Context(), models.SessionID(cookie.Value))
		if err != nil {
			utils.WriteError(w, "Unauthorised", http.StatusUnauthorized)
			return nil, false
		}
		userAccountID = session.UserID
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), userAccountID)
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

	userAccount, err := h.userService.GetUserAccountByProfileID(r.Context(), profile.ID)
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
	if request.Category < models.CategoryBug || request.Category > models.CategoryOther {
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

	ticket := models.NewSupportTicket(profile.ID, request.Login, email, request.Category, request.Title, request.Description)
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
	_ = json.NewEncoder(w).Encode(SupportResponse{ID: ticketID, Login: ticket.Login, Status: ticket.Status, Media: mediaResponse})
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

	upd := tickets.TicketUpdate{Title: normalizeOptionalString(request.Title), Description: normalizeOptionalString(request.Description), Category: request.Category}
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

	ticket, err := h.ticketService.UpdateStatusByAgent(r.Context(), ticketID, *request.Status)
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
		value, err := strconv.Atoi(raw)
		if err != nil || value < int(models.TicketStatusOpen) || value > int(models.TicketStatusClosed) {
			utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
			return filter, false
		}
		status := models.TicketStatus(value)
		filter.Status = &status
	}
	if raw := query.Get("category"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < int(models.CategoryBug) || value > int(models.CategoryOther) {
			utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
			return filter, false
		}
		category := models.TicketCategory(value)
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
