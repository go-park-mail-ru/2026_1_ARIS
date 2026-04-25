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
}

func NewSupportHandler(sessionService session.SessionService, userService user.UserService, ticketService tickets.TicketService) *SupportHandler {
	return &SupportHandler{
		sessionService: sessionService,
		userService:    userService,
		ticketService:  ticketService,
	}
}

type SupportResponse struct {
	ID     int64               `json:"id"`
	Login  string              `json:"login"`
	Status models.TicketStatus `json:"status"`
}

type SupportTicketResponse struct {
	ID          int64                 `json:"id"`
	ProfileID   int64                 `json:"profileID"`
	Login       string                `json:"login"`
	Email       string                `json:"email"`
	Category    models.TicketCategory `json:"category"`
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Status      models.TicketStatus   `json:"status"`
	Priority    models.TicketPriority `json:"priority"`
	CreatedAt   time.Time             `json:"createdAt"`
	UpdatedAt   time.Time             `json:"updatedAt"`
	ClosedAt    *time.Time            `json:"closedAt,omitempty"`
}

type SupportUpdateRequest struct {
	Category    *models.TicketCategory `json:"category"`
	Title       *string                `json:"title"`
	Description *string                `json:"description"`
}

func buildSupportTicketResponse(ticket *models.SupportTicket) SupportTicketResponse {
	return SupportTicketResponse{
		ID:          ticket.ID,
		ProfileID:   ticket.ProfileID,
		Login:       ticket.Login,
		Email:       ticket.Email,
		Category:    ticket.Category,
		Title:       ticket.Title,
		Description: ticket.Description,
		Status:      ticket.Status,
		Priority:    ticket.Priority,
		CreatedAt:   ticket.CreatedAt,
		UpdatedAt:   ticket.UpdatedAt,
		ClosedAt:    ticket.ClosedAt,
	}
}

func buildSupportTicketResponses(tickets []models.SupportTicket) []SupportTicketResponse {
	responses := make([]SupportTicketResponse, 0, len(tickets))
	for i := range tickets {
		responses = append(responses, buildSupportTicketResponse(&tickets[i]))
	}
	return responses
}

func (h *SupportHandler) getCurrentProfile(w http.ResponseWriter, r *http.Request) (*models.Profile, bool) {
	log := logger.FromContext(r.Context())

	cookie, err := r.Cookie("session_id")
	if err != nil {
		log.Warn("cannot_get_support_profile_missing_session",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, "Unauthorised", http.StatusUnauthorized)
		return nil, false
	}

	session, err := h.sessionService.Get(r.Context(), models.SessionID(cookie.Value))
	if err != nil {
		log.Warn("cannot_get_support_profile_invalid_session",
			zap.String("session_id", cookie.Value),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, "Unauthorised", http.StatusUnauthorized)
		return nil, false
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), session.UserID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_get_support_profile_profile_not_found",
				zap.Int64("userAccount_id", session.UserID),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return nil, false
		}
		log.Error("failed_to_get_profile",
			zap.Int64("userAccount_id", session.UserID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return nil, false
	}

	return profile, true
}

func (h *SupportHandler) SendTicket(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	cookie, err := r.Cookie("session_id")
	if err != nil {
		log.Warn("cannot_get_profile_me_missing_session",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, "Unauthorised", http.StatusUnauthorized)
		return
	}

	session, err := h.sessionService.Get(r.Context(), models.SessionID(cookie.Value))
	if err != nil {
		log.Warn("cannot_get_profile_me_invalid_session",
			zap.String("session_id", cookie.Value),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, "Unauthorised", http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), session.UserID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_create_post_profile_not_found",
				zap.Int64("userAccount_id", session.UserID),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_profile",
			zap.Int64("userAccount_id", session.UserID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	userAccount, err := h.userService.GetUserAccountByProfileID(r.Context(), profile.ID)
	if err != nil {
		utils.WriteError(w, err.Error(), http.StatusNotFound)
		return
	}

	var request SupportRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Warn("cannot_create_post_invalid_body",
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	request.Title = strings.TrimSpace(request.Title)
	request.Login = strings.TrimSpace(request.Login)
	request.Description = strings.TrimSpace(request.Description)
	if request.Title == "" || request.Description == "" {
		log.Warn("cannot_create_support_ticket_empty_content",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	if request.Category < models.CategoryBug || request.Category > models.CategoryOther {
		log.Warn("cannot_create_support_ticket_invalid_category",
			zap.Int("category", int(request.Category)),
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	if request.Login == "" {
		log.Warn("cannot_create_support_ticket_empty_login",
			zap.Int64("profile_id", profile.ID),
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(request.Email)
	if email == "" && userAccount.Email != nil {
		email = *userAccount.Email
	}
	if email == "" {
		log.Warn("cannot_create_support_ticket_empty_email",
			zap.Int64("profile_id", profile.ID),
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	ticket := models.NewSupportTicket(
		profile.ID,
		request.Login,
		email,
		request.Category,
		request.Title,
		request.Description,
	)

	ticketID, err := h.ticketService.Save(r.Context(), ticket)
	if err != nil {
		log.Error("failed_to_save_support_ticket",
			zap.Int64("profile_id", profile.ID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	log.Info("support_ticket_created",
		zap.Int64("ticket_id", ticketID),
		zap.Int64("profile_id", profile.ID),
		zap.Int("category", int(ticket.Category)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(SupportResponse{
		ID:     ticketID,
		Login:  ticket.Login,
		Status: ticket.Status,
	})
}

func (h *SupportHandler) GetMyTickets(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	profile, ok := h.getCurrentProfile(w, r)
	if !ok {
		return
	}

	tickets, err := h.ticketService.GetByProfileID(r.Context(), profile.ID)
	if err != nil {
		log.Error("failed_to_get_support_tickets",
			zap.Int64("profile_id", profile.ID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(buildSupportTicketResponses(tickets))
}

func (h *SupportHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	profile, ok := h.getCurrentProfile(w, r)
	if !ok {
		return
	}

	ticketID, err := strconv.ParseInt(chi.URLParam(r, "ticketID"), 10, 64)
	if err != nil || ticketID <= 0 {
		utils.WriteError(w, xerrors.InvalidID, http.StatusBadRequest)
		return
	}

	ticket, err := h.ticketService.GetByID(r.Context(), ticketID, profile.ID)
	if err != nil {
		if errors.Is(err, xerrors.SupportTicketNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_support_ticket",
			zap.Int64("ticket_id", ticketID),
			zap.Int64("profile_id", profile.ID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(buildSupportTicketResponse(ticket))
}

func (h *SupportHandler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	profile, ok := h.getCurrentProfile(w, r)
	if !ok {
		return
	}

	ticketID, err := strconv.ParseInt(chi.URLParam(r, "ticketID"), 10, 64)
	if err != nil || ticketID <= 0 {
		utils.WriteError(w, xerrors.InvalidID, http.StatusBadRequest)
		return
	}

	var request SupportUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Warn("cannot_update_support_ticket_invalid_body",
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	upd := tickets.TicketUpdate{
		Title:       normalizeOptionalString(request.Title),
		Description: normalizeOptionalString(request.Description),
		Category:    request.Category,
	}
	if upd.IsEmpty() {
		utils.WriteError(w, xerrors.ErrNothingToUpdate.Error(), http.StatusBadRequest)
		return
	}

	if upd.Title != nil && *upd.Title == "" {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}
	if upd.Description != nil && *upd.Description == "" {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}
	if upd.Category != nil && (*upd.Category < models.CategoryBug || *upd.Category > models.CategoryOther) {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return
	}

	ticket, err := h.ticketService.Update(r.Context(), ticketID, profile.ID, upd)
	if err != nil {
		if errors.Is(err, xerrors.SupportTicketNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_update_support_ticket",
			zap.Int64("ticket_id", ticketID),
			zap.Int64("profile_id", profile.ID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(buildSupportTicketResponse(ticket))
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	normalized := strings.TrimSpace(*value)
	return &normalized
}
