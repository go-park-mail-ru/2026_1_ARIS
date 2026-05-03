package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/user/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	user     *service.Service
	validate *validator.Validate
}

func New(user *service.Service) *Handler {
	return &Handler{user: user, validate: validator.New()}
}

func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Get("/public/popular-users", h.GetPublicPopularUsers)
	r.Get("/users/{id}/friends", h.GetUsersFriends)

	r.Group(func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		r.Get("/users/suggested", h.GetSuggestedUsers)
		r.Get("/users/latest-events", h.GetLatestEvents)
		r.Get("/profile/me", h.GetProfileMe)
		r.Get("/profile/{id}", h.GetProfileByID)
		r.Patch("/profile/me/edit", h.EditProfileMe)
		r.Get("/settings/", h.GetSettings)
		r.Post("/settings/", h.SetSettings)
	})

	r.Route("/friends", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		r.Post("/request", h.RequestFriendship)
		r.Post("/accept/{requesterID}", h.AcceptFriendRequest)
		r.Post("/decline/{requesterID}", h.DeclineFriendRequest)
		r.Delete("/request/{addresseeID}", h.RevokeFriendRequest)
		r.Get("/requests/incoming/{status}", h.GetIncomingFriendRequests)
		r.Get("/requests/incoming", h.GetIncomingFriendRequests)
		r.Get("/requests/outgoing/{status}", h.GetOutgoingFriendRequests)
		r.Get("/requests/outgoing", h.GetOutgoingFriendRequests)
		r.Delete("/{userID}", h.DeleteFriend)
		r.Get("/{status}", h.GetFriends)
		r.Get("/", h.GetFriends)
	})
}

func (h *Handler) GetProfileMe(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}

	profile, err := h.user.GetProfileMe(r.Context(), userAccountID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapProfile(profile))
}

func (h *Handler) GetProfileByID(w http.ResponseWriter, r *http.Request) {
	profileID, ok := parseID(w, chi.URLParam(r, "id"), "invalid profile id")
	if !ok {
		return
	}

	profile, err := h.user.GetProfileByID(r.Context(), profileID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapProfile(profile))
}

func (h *Handler) EditProfileMe(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}

	var req dto.UpdateFullProfileRequestDTO
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}

	reqForValidation := normalizeOptionalEmptyFields(req)
	if err := h.validate.Struct(reqForValidation); err != nil {
		utils.WriteError(
			w,
			fmt.Sprintf("%s: %s", xerrors.ValidationError, err.Error()),
			http.StatusBadRequest)
		return
	}

	if err := h.user.UpdateMe(r.Context(), userAccountID, req); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetSuggestedUsers(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}

	users, err := h.user.GetSuggestedUsers(r.Context(), userAccountID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, suggestedUsersResponse{Items: mapUserCards(users)})
}

func (h *Handler) GetPublicPopularUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.user.GetPublicPopularUsers(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, suggestedUsersResponse{Items: mapUserCards(users)})
}

func (h *Handler) GetLatestEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.user.GetLatestEvents(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	items := make([]latestEventDTO, 0, len(events))
	for _, event := range events {
		items = append(items, latestEventDTO{
			Id:         strconv.FormatInt(event.ID, 10),
			FirstName:  event.FirstName,
			LastName:   event.LastName,
			Username:   event.Username,
			AvatarLink: event.AvatarLink,
			Type:       event.Type,
		})
	}

	writeJSON(w, http.StatusOK, latestEventsResponse{Items: items})
}

func (h *Handler) SetSettings(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}

	var req dto.UserSettingsUpdate
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		utils.WriteError(w, xerrors.ValidationError, http.StatusBadRequest)
		return
	}

	settings, err := h.user.UpdateSettings(r.Context(), userAccountID, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, settings)
}

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}

	settings, err := h.user.GetSettings(r.Context(), userAccountID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, settings)
}

func userIDFromContext(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok || userAccountID <= 0 {
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return 0, false
	}
	return userAccountID, true
}

func parseID(w http.ResponseWriter, value string, errorMessage string) (int64, bool) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, errorMessage, http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput), errors.Is(err, service.ErrNothingToUpdate):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrProfileNotFound), errors.Is(err, service.ErrUserProfileNotFound), errors.Is(err, service.ErrUserAccountNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: xerrors.InternalServerErrorStr})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func mapProfile(profile *service.ProfileDetails) GetProfileMeResponse {
	resp := GetProfileMeResponse{
		ProfileID:     profile.ProfileID,
		UserAccountID: profile.UserAccountID,
		Username:      html.EscapeString(profile.Username),
		FirstName:     html.EscapeString(profile.FirstName),
		LastName:      html.EscapeString(profile.LastName),
		Bio:           escapePtr(profile.Bio),
		ImageLink:     profile.ImageLink,
		Gender:        profile.Gender,
		NativeTown:    escapePtr(profile.NativeTown),
		Phone:         profile.Phone,
		Email:         profile.Email,
		Town:          escapePtr(profile.Town),
		Interests:     escapePtr(profile.Interests),
		FavMusic:      escapePtr(profile.FavMusic),
	}

	if !profile.BirthdayDate.IsZero() {
		resp.BirthdayDate = profile.BirthdayDate.Format(time.DateOnly)
	}

	resp.Education = make([]EducationResponse, 0, len(profile.Education))
	for _, education := range profile.Education {
		if education.Institution == nil && education.Group == nil {
			continue
		}
		resp.Education = append(resp.Education, EducationResponse{
			Institution: escapePtr(education.Institution),
			Group:       escapePtr(education.Group),
		})
	}

	resp.Work = make([]WorkResponse, 0, len(profile.Work))
	for _, work := range profile.Work {
		if work.Company == nil && work.JobTitle == nil {
			continue
		}
		resp.Work = append(resp.Work, WorkResponse{
			Company:  escapePtr(work.Company),
			JobTitle: escapePtr(work.JobTitle),
		})
	}

	return resp
}

func mapUserCards(users []service.UserCard) []suggestedUserDTO {
	items := make([]suggestedUserDTO, 0, len(users))
	for _, user := range users {
		items = append(items, suggestedUserDTO{
			Id:         strconv.FormatInt(user.ID, 10),
			FirstName:  user.FirstName,
			LastName:   user.LastName,
			Username:   user.Username,
			AvatarLink: user.AvatarLink,
		})
	}
	return items
}

func escapePtr(value *string) *string {
	if value == nil {
		return nil
	}
	escaped := html.EscapeString(*value)
	return &escaped
}

func normalizeOptionalEmptyFields(req dto.UpdateFullProfileRequestDTO) dto.UpdateFullProfileRequestDTO {
	if req.Username != nil && strings.TrimSpace(*req.Username) == "" {
		req.Username = nil
	}
	if req.Email != nil && strings.TrimSpace(*req.Email) == "" {
		req.Email = nil
	}
	if req.Phone != nil && strings.TrimSpace(*req.Phone) == "" {
		req.Phone = nil
	}
	return req
}
