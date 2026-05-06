package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	servicedto "github.com/go-park-mail-ru/2026_1_ARIS/internal/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/user/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
)

type friendsResponse struct {
	Friends []servicedto.FriendDTO `json:"friends"`
}

type friendRequest struct {
	FriendID int64 `json:"friendID"`
}

func (h *Handler) GetFriends(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	status := chi.URLParam(r, "status")
	if status == "" {
		status = string(models.FriendshipAccepted)
	}
	friends, err := h.user.GetFriends(r.Context(), userAccountID, models.FriendshipStatus(status))
	if err != nil {
		writeFriendshipError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, friendsResponse{Friends: friends})
}

func (h *Handler) GetUsersFriends(w http.ResponseWriter, r *http.Request) {
	profileID, ok := parseID(w, chi.URLParam(r, "id"), xerrors.InvalidID)
	if !ok {
		return
	}
	friends, err := h.user.GetUsersFriends(r.Context(), profileID)
	if err != nil {
		writeFriendshipError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, friendsResponse{Friends: friends})
}

func (h *Handler) DeleteFriend(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	friendID, ok := parseID(w, chi.URLParam(r, "userID"), xerrors.InvalidID)
	if !ok {
		return
	}
	if err := h.user.DeleteFriend(r.Context(), userAccountID, friendID); err != nil {
		writeFriendshipError(w, err)
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
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}
	if err := h.user.RequestFriendship(r.Context(), userAccountID, req.FriendID); err != nil {
		writeFriendshipError(w, err)
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
		status = string(models.FriendshipPending)
	}
	friends, err := h.user.GetIncomingFriendRequests(r.Context(), userAccountID, status)
	if err != nil {
		writeFriendshipError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, friendsResponse{Friends: friends})
}

func (h *Handler) GetOutgoingFriendRequests(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	status := chi.URLParam(r, "status")
	if status == "" {
		status = string(models.FriendshipPending)
	}
	friends, err := h.user.GetOutgoingFriendRequests(r.Context(), userAccountID, status)
	if err != nil {
		writeFriendshipError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, friendsResponse{Friends: friends})
}

func (h *Handler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	requesterID, ok := parseID(w, chi.URLParam(r, "requesterID"), xerrors.InvalidID)
	if !ok {
		return
	}
	if err := h.user.AcceptFriendRequest(r.Context(), userAccountID, requesterID); err != nil {
		writeFriendshipError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) DeclineFriendRequest(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	requesterID, ok := parseID(w, chi.URLParam(r, "requesterID"), xerrors.InvalidID)
	if !ok {
		return
	}
	if err := h.user.DeclineFriendRequest(r.Context(), userAccountID, requesterID); err != nil {
		writeFriendshipError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) RevokeFriendRequest(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	addresseeID, ok := parseID(w, chi.URLParam(r, "addresseeID"), xerrors.InvalidID)
	if !ok {
		return
	}
	if err := h.user.RevokeFriendRequest(r.Context(), userAccountID, addresseeID); err != nil {
		writeFriendshipError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func writeFriendshipError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput), errors.Is(err, service.ErrInvalidStatus):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrProfileNotFound), errors.Is(err, service.ErrFriendshipNotFound), errors.Is(err, service.ErrFriendshipNotExists):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrAlreadyFriends):
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: xerrors.InternalServerErrorStr})
	}
}
