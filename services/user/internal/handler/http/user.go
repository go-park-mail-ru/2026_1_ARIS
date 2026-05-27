package http

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/common"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	user     UserService
	validate *validator.Validate
}

func New(user UserService) *Handler {
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
	utils.WriteJSON(w, http.StatusOK, mapProfile(profile))
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
	utils.WriteJSON(w, http.StatusOK, mapProfile(profile))
}

func (h *Handler) EditProfileMe(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	var req updateProfileRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	req = normalizeOptionalEmptyFields(req)
	if err := h.validate.Struct(req); err != nil {
		writeError(w, fmt.Sprintf("validation error: %s", err.Error()), http.StatusBadRequest)
		return
	}
	if err := h.user.UpdateMe(r.Context(), userAccountID, usecase.UpdateFullProfileInput{
		Username:     req.Username,
		Email:        req.Email,
		Phone:        req.Phone,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Bio:          req.Bio,
		BirthdayDate: req.BirthdayDate,
		Gender:       req.Gender,
		NativeTown:   req.NativeTown,
		Town:         req.Town,
		Institution:  req.Institution,
		Group:        req.Group,
		Company:      req.Company,
		JobTitle:     req.JobTitle,
		Interests:    req.Interests,
		FavMusic:     req.FavMusic,
		AvatarID:     req.AvatarID,
		RemoveAvatar: req.RemoveAvatar,
	}); err != nil {
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
	utils.WriteJSON(w, http.StatusOK, userCardsResponse{Items: mapUserCards(users)})
}

func (h *Handler) GetPublicPopularUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.user.GetPublicPopularUsers(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, userCardsResponse{Items: mapUserCards(users)})
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
			ID:         strconv.FormatInt(event.ID, 10),
			FirstName:  event.FirstName,
			LastName:   event.LastName,
			Username:   event.Username,
			AvatarLink: event.AvatarLink,
			Type:       event.Type,
		})
	}
	utils.WriteJSON(w, http.StatusOK, latestEventsResponse{Items: items})
}

func (h *Handler) SetSettings(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	var req settingsUpdateRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	if err := h.validate.Struct(req); err != nil {
		writeError(w, "validation error", http.StatusBadRequest)
		return
	}
	settings, err := h.user.UpdateSettings(r.Context(), userAccountID, repository.SettingsUpdate{
		Language: req.Language,
		Theme:    req.Theme,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, settings)
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
	utils.WriteJSON(w, http.StatusOK, settings)
}

func userIDFromContext(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok || userAccountID <= 0 {
		writeError(w, "invalid user context", http.StatusUnauthorized)
		return 0, false
	}
	return userAccountID, true
}

func parseID(w http.ResponseWriter, value string, errorMessage string) (int64, bool) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, errorMessage, http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidInput), errors.Is(err, usecase.ErrNothingToUpdate), errors.Is(err, usecase.ErrInvalidStatus):
		writeError(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, usecase.ErrProfileNotFound), errors.Is(err, usecase.ErrUserProfileNotFound),
		errors.Is(err, usecase.ErrUserAccountNotFound), errors.Is(err, usecase.ErrFriendshipNotFound),
		errors.Is(err, usecase.ErrFriendshipNotExists):
		writeError(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, usecase.ErrAlreadyFriends):
		writeError(w, err.Error(), http.StatusConflict)
	default:
		writeError(w, "internal server error", http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, message string, status int) {
	utils.WriteJSON(w, status, common.ErrorResponse{Error: message})
}

func mapProfile(profile *usecase.ProfileDetails) profileResponse {
	resp := profileResponse{
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
		IsOnline:      profile.IsOnline,
	}
	if !profile.BirthdayDate.IsZero() {
		resp.BirthdayDate = profile.BirthdayDate.Format(time.DateOnly)
	}
	if profile.LastSeenAt != nil && !profile.LastSeenAt.IsZero() {
		lastSeenAt := profile.LastSeenAt.UTC().Format(time.RFC3339Nano)
		resp.LastSeenAt = &lastSeenAt
	}
	resp.Education = make([]educationResponse, 0, len(profile.Education))
	for _, education := range profile.Education {
		if education.Institution == nil && education.Group == nil {
			continue
		}
		resp.Education = append(resp.Education, educationResponse{
			Institution: escapePtr(education.Institution),
			Group:       escapePtr(education.Group),
		})
	}
	resp.Work = make([]workResponse, 0, len(profile.Work))
	for _, work := range profile.Work {
		if work.Company == nil && work.JobTitle == nil {
			continue
		}
		resp.Work = append(resp.Work, workResponse{
			Company:  escapePtr(work.Company),
			JobTitle: escapePtr(work.JobTitle),
		})
	}
	return resp
}

func mapUserCards(users []usecase.UserCard) []userCardDTO {
	items := make([]userCardDTO, 0, len(users))
	for _, user := range users {
		items = append(items, userCardDTO{
			ID:         strconv.FormatInt(user.ID, 10),
			FirstName:  user.FirstName,
			LastName:   user.LastName,
			Username:   user.Username,
			AvatarLink: user.AvatarLink,
			IsOnline:   user.IsOnline,
			LastSeenAt: formatOptionalTime(user.LastSeenAt),
		})
	}
	return items
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil || value.IsZero() {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func escapePtr(value *string) *string {
	if value == nil {
		return nil
	}
	escaped := html.EscapeString(*value)
	return &escaped
}

func normalizeOptionalEmptyFields(req updateProfileRequest) updateProfileRequest {
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
