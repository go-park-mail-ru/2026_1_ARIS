package user

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
)

type UserHandler struct {
	UserService  user.UserService
	MediaService media.MediaService
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

func NewUserHandler(userProfileService user.UserService, mediaService media.MediaService) *UserHandler {
	return &UserHandler{
		UserService:  userProfileService,
		MediaService: mediaService,
	}
}

func (h *UserHandler) GetSuggestedUsers(w http.ResponseWriter, r *http.Request) {

	userIDFromCtx := r.Context().Value("user_id")
	if userIDFromCtx == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID, ok := userIDFromCtx.(int64)
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
			media, err := h.MediaService.GetAvatarByID(r.Context(), user.AvatarID)
			if err == nil && media != nil {
				avatar = media.Link
			}
		}
		userProfile, err := h.UserService.GetUserProfileByProfileID(r.Context(), user.ID)
		if err != nil {
			continue
		}
		userAccount, err := h.UserService.GetUserAccountByUserProfileID(r.Context(), userProfile.ID)
		if err != nil {
			continue
		}

		items = append(items, suggestedUserDTO{
			Id:         strconv.FormatInt(user.ID, 10),
			FirstName:  userProfile.FirstName,
			LastName:   userProfile.LastName,
			Username:   userAccount.Username,
			AvatarLink: avatar,
		})
	}

	json.NewEncoder(w).Encode(suggestedUsersResponse{
		Items: items,
	})
}

func (h *UserHandler) GetPublicPopularUsers(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GetPublicPopularUsers")
	users, err := h.UserService.GetPublicPopularUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println("not error h.UserService.GetPublicPopularUsers(r.Context())")
	fmt.Println(users)

	var items []suggestedUserDTO

	for _, user := range users {
		fmt.Println("User:", user)
		avatar := ""

		if user.AvatarID != nil && h.MediaService != nil {
			media, err := h.MediaService.GetAvatarByID(r.Context(), user.AvatarID)
			if err == nil && media != nil {
				avatar = media.Link
			}
		}

		userProfile, err := h.UserService.GetUserProfileByProfileID(r.Context(), user.ID)
		if err != nil {
			fmt.Println("Err != nil: h.UserService.GetUserProfileByProfileID", err.Error())
			continue
		}
		userAccount, err := h.UserService.GetUserAccountByUserProfileID(r.Context(), userProfile.ID)
		if err != nil {
			fmt.Println(userProfile.ID, err.Error())
			continue
		}

		items = append(items, suggestedUserDTO{
			Id:         strconv.FormatInt(user.ID, 10),
			FirstName:  userProfile.FirstName,
			LastName:   userProfile.LastName,
			Username:   userAccount.Username,
			AvatarLink: avatar,
		})
	}

	fmt.Println("Items:")
	fmt.Println(items)

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
			media, err := h.MediaService.GetAvatarByID(r.Context(), user.AvatarID)
			if err == nil && media != nil {
				avatar = media.Link
			}
		}

		userProfile, err := h.UserService.GetUserProfileByProfileID(r.Context(), user.ID)
		if err != nil {
			continue
		}
		userAccount, err := h.UserService.GetUserAccountByUserProfileID(r.Context(), userProfile.ID)
		if err != nil {
			continue
		}

		items = append(items, latestEventDTO{
			Id:         strconv.FormatInt(user.ID, 10),
			FirstName:  userProfile.FirstName,
			LastName:   userProfile.LastName,
			Username:   userAccount.Username,
			AvatarLink: avatar,
			Type:       event.Type,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(latestEventsResponse{
		Items: items,
	})
}
