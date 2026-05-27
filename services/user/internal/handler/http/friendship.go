package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/user/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
)

func (h *Handler) GetFriends(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	status := chi.URLParam(r, "status")
	if status == "" {
		status = string(model.FriendshipAccepted)
	}
	friends, err := h.user.GetFriends(r.Context(), userAccountID, model.FriendshipStatus(status))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, friendsResponse{Friends: mapFriends(friends)})
}

func (h *Handler) GetUsersFriends(w http.ResponseWriter, r *http.Request) {
	profileID, ok := parseID(w, chi.URLParam(r, "id"), "invalid id")
	if !ok {
		return
	}
	friends, err := h.user.GetUsersFriends(r.Context(), profileID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, friendsResponse{Friends: mapFriends(friends)})
}

func (h *Handler) DeleteFriend(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	friendID, ok := parseID(w, chi.URLParam(r, "userID"), "invalid id")
	if !ok {
		return
	}
	if err := h.user.DeleteFriend(r.Context(), userAccountID, friendID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) RequestFriendship(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	var req friendRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	if err := h.user.RequestFriendship(r.Context(), userAccountID, req.FriendID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetIncomingFriendRequests(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	status := chi.URLParam(r, "status")
	if status == "" {
		status = string(model.FriendshipPending)
	}
	friends, err := h.user.GetIncomingFriendRequests(r.Context(), userAccountID, status)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, friendsResponse{Friends: mapFriends(friends)})
}

func (h *Handler) GetOutgoingFriendRequests(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	status := chi.URLParam(r, "status")
	if status == "" {
		status = string(model.FriendshipPending)
	}
	friends, err := h.user.GetOutgoingFriendRequests(r.Context(), userAccountID, status)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, friendsResponse{Friends: mapFriends(friends)})
}

func (h *Handler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	requesterID, ok := parseID(w, chi.URLParam(r, "requesterID"), "invalid id")
	if !ok {
		return
	}
	if err := h.user.AcceptFriendRequest(r.Context(), userAccountID, requesterID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) DeclineFriendRequest(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	requesterID, ok := parseID(w, chi.URLParam(r, "requesterID"), "invalid id")
	if !ok {
		return
	}
	if err := h.user.DeclineFriendRequest(r.Context(), userAccountID, requesterID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) RevokeFriendRequest(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	addresseeID, ok := parseID(w, chi.URLParam(r, "addresseeID"), "invalid id")
	if !ok {
		return
	}
	if err := h.user.RevokeFriendRequest(r.Context(), userAccountID, addresseeID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func mapFriends(friends []model.Friend) []friendDTO {
	items := make([]friendDTO, 0, len(friends))
	for _, friend := range friends {
		items = append(items, friendDTO{
			AvatarID:   friend.AvatarID,
			ProfileID:  friend.ProfileID,
			FirstName:  friend.FirstName,
			LastName:   friend.LastName,
			Username:   friend.Username,
			Status:     friend.Status,
			AvatarLink: friend.Link,
			CreatedAt:  friend.CreatedAt,
			UpdatedAt:  friend.UpdatedAt,
		})
	}
	return items
}
