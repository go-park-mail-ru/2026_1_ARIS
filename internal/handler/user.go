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

	userIDfromCtx := r.Context().Value("user_id")
	if userIDfromCtx == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID := userIDfromCtx.(uuid.UUID)

	users, err := h.UserService.GetSuggestedUsers(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var items []suggestedUserDTO

	for _, u := range users {

		avatar := ""

		if u.AvatarID != nil && h.MediaService != nil {
			media, err := h.MediaService.GetAvatarByID(*u.AvatarID)
			if err == nil && media != nil {
				avatar = media.Link
			}
		}
		profile, err := h.UserService.GetUserProfileByProfile(r.Context(), u.ID)
		if err != nil {
			continue
		}
		items = append(items, suggestedUserDTO{
			Id:         u.ID.String(),
			FirstName:  profile.FirstName,
			LastName:   profile.LastName,
			Username:   u.Username,
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

	for _, u := range users {
		avatar := ""

		if u.AvatarID != nil && h.MediaService != nil {
			media, err := h.MediaService.GetAvatarByID(*u.AvatarID)
			if err == nil && media != nil {
				avatar = media.Link
			}
		}

		profile, err := h.UserService.GetUserProfileByProfile(r.Context(), u.ID)
		if err != nil {
			continue
		}

		items = append(items, suggestedUserDTO{
			Id:         u.ID.String(),
			FirstName:  profile.FirstName,
			LastName:   profile.LastName,
			Username:   u.Username,
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
		u := event.Profile
		avatar := ""

		if u.AvatarID != nil && h.MediaService != nil {
			media, err := h.MediaService.GetAvatarByID(*u.AvatarID)
			if err == nil && media != nil {
				avatar = media.Link
			}
		}

		profile, err := h.UserService.GetUserProfileByProfile(r.Context(), u.ID)
		if err != nil {
			continue
		}

		items = append(items, latestEventDTO{
			Id:         u.ID.String(),
			FirstName:  profile.FirstName,
			LastName:   profile.LastName,
			Username:   u.Username,
			AvatarLink: avatar,
			Type:       event.Type,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(latestEventsResponse{
		Items: items,
	})
}
