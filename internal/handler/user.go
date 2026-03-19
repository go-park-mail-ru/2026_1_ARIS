package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
	"github.com/google/uuid"
)

type UserHandler struct {
	UserService  service.UserService
	MediaService service.MediaService
}

type latestEventDTO struct {
	Id         string `json:"id"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	Username   string `json:"username"`
	AvatarLink string `json:"avatarLink"`
	Type       int    `json:"type"`
}

type latestEventsResponse struct {
	Items []latestEventDTO `json:"items"`
}

type suggestedUserDTO struct {
	Id         string `json:"id"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	Username   string `json:"username"`
	AvatarLink string `json:"avatarLink"`
}

type suggestedUsersResponse struct {
	Items []suggestedUserDTO `json:"items"`
}

func (h *UserHandler) GetSuggestedUsers(w http.ResponseWriter, r *http.Request) {

	userIDFromCtx := r.Context().Value("user_id")
	if userIDFromCtx == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID, ok := userIDFromCtx.(uuid.UUID)
	if !ok {
		http.Error(w, "invalid user id in context", http.StatusUnauthorized)
		return
	}

	users, err := h.UserService.GetSuggestedUsers(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var items []suggestedUserDTO

	for _, user := range users {

		avatar := ""

		if user.AvatarID != nil && h.MediaService != nil {
			media, err := h.MediaService.GetAvatarByID(*user.AvatarID)
			if err == nil && media != nil {
				avatar = media.Link
			}
		}
		profile, err := h.UserService.GetUserProfileByProfile(r.Context(), user.ID)
		if err != nil {
			continue
		}
		items = append(items, suggestedUserDTO{
			Id:         user.ID.String(),
			FirstName:  profile.FirstName,
			LastName:   profile.LastName,
			Username:   user.Username,
			AvatarLink: avatar,
		})
	}

	json.NewEncoder(w).Encode(suggestedUsersResponse{
		Items: items,
	})
}

func (h *UserHandler) GetPublicPopularUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.UserService.GetPublicPopularUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var items []suggestedUserDTO

	for _, user := range users {
		avatar := ""

		if user.AvatarID != nil && h.MediaService != nil {
			media, err := h.MediaService.GetAvatarByID(*user.AvatarID)
			if err == nil && media != nil {
				avatar = media.Link
			}
		}

		profile, err := h.UserService.GetUserProfileByProfile(r.Context(), user.ID)
		if err != nil {
			continue
		}

		items = append(items, suggestedUserDTO{
			Id:         user.ID.String(),
			FirstName:  profile.FirstName,
			LastName:   profile.LastName,
			Username:   user.Username,
			AvatarLink: avatar,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestedUsersResponse{
		Items: items,
	})
}

func (h *UserHandler) GetLatestEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.UserService.GetLatestEvents(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var items []latestEventDTO

	for _, event := range events {
		user := event.Profile
		avatar := ""

		if user.AvatarID != nil && h.MediaService != nil {
			media, err := h.MediaService.GetAvatarByID(*user.AvatarID)
			if err == nil && media != nil {
				avatar = media.Link
			}
		}

		profile, err := h.UserService.GetUserProfileByProfile(r.Context(), user.ID)
		if err != nil {
			continue
		}

		items = append(items, latestEventDTO{
			Id:         user.ID.String(),
			FirstName:  profile.FirstName,
			LastName:   profile.LastName,
			Username:   user.Username,
			AvatarLink: avatar,
			Type:       event.Type,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(latestEventsResponse{
		Items: items,
	})
}
