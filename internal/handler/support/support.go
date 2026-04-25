package support

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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
	Status models.TicketStatus `json:"status"`
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

	login := strings.TrimSpace(request.Login)
	if login == "" {
		login = userAccount.Username
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
		login,
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
		Status: ticket.Status,
	})
}
