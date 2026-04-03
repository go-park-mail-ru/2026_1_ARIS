package friend

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/friend"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
)

type FriendHandler struct {
	sessionService    session.SessionService
	userService       user.UserService
	friendshipService friend.FriendshipService
}

func NewFriendHandler(sessionService session.SessionService, userService user.UserService, friendshipService friend.FriendshipService) *FriendHandler {
	return &FriendHandler{
		sessionService:    sessionService,
		userService:       userService,
		friendshipService: friendshipService,
	}
}

type friendsResponse struct {
	Friends []dto.FriendDTO `json:"friends"`
}

type friendRequest struct {
	FriendID int64 `json:"friendID"`
}

// @Description	Getting current users' friends
// @ID			get-friends
// @Summary		Get friends
// @Tags		friend
// @Security	SessionAuth
// @Produce		json
// @Param status path string false "friendship status"
// @Success		200		{object}	friendsResponse
// @Failure		400		{object}	dto.CommonErrorResponse
// @Failure		401		{object}	dto.CommonErrorResponse
// @Failure		404		{object}	dto.CommonErrorResponse
// @Failure		500		{object}	dto.CommonErrorResponse
// @Router		/friends/{status} [get]
func (h *FriendHandler) GetFriends(w http.ResponseWriter, r *http.Request) {
	friendshipStatus := chi.URLParam(r, "status")
	if friendshipStatus == "" {
		friendshipStatus = string(models.FriendshipPending)
	}

	if friendshipStatus != string(models.FriendshipPending) &&
		friendshipStatus != string(models.FriendshipAccepted) {

		utils.WriteError(w, "Unknown status value", http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("session_id")
	if err != nil {
		utils.WriteError(w, xerrors.UnauthorizedStr, http.StatusUnauthorized)
		return
	}

	session, err := h.sessionService.Get(r.Context(), models.SessionID(cookie.Value))
	if err != nil {
		utils.WriteError(w, xerrors.UnauthorizedStr, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), session.UserID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	friends, err := h.friendshipService.GetFriends(r.Context(), profile.ID, models.FriendshipStatus(friendshipStatus))
	if err != nil {
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	response := friendsResponse{
		Friends: friends,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// @Description	Getting any users' friends by profile ID
// @ID			get-any-friends
// @Summary		Get any users' friends
// @Tags		friend
// @Produce		json
// @Param 		user_id path	integer	true "user id to get its friends"
// @Success		200		{object}	friendsResponse
// @Failure		400		{object}	dto.CommonErrorResponse
// @Failure		404		{object}	dto.CommonErrorResponse
// @Failure		500		{object}	dto.CommonErrorResponse
// @Router		/users/{id}/friends [get]
func (h *FriendHandler) GetUsersFriends(w http.ResponseWriter, r *http.Request) {
	profileIDstr := chi.URLParam(r, "id")
	profileID, err := strconv.Atoi(profileIDstr)
	if err != nil {
		fmt.Println(profileIDstr, err)
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	if profileID <= 0 {
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	profile, err := h.userService.GetProfileByProfileID(r.Context(), int64(profileID))
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	friends, err := h.friendshipService.GetFriends(r.Context(), profile.ID, models.FriendshipAccepted)
	if err != nil {
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	response := friendsResponse{
		Friends: friends,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// @Description	Delete current users' friend by friend ID
// @ID			delete-friend
// @Summary		Delete current users' friend
// @Tags		friend
// @Security	SessionAuth
// @Produce		json
// @Success		200
// @Failure		400		{object}	dto.CommonErrorResponse
// @Failure		401		{object}	dto.CommonErrorResponse
// @Failure		404		{object}	dto.CommonErrorResponse
// @Failure		500		{object}	dto.CommonErrorResponse
// @Router		/friends/{userID} [delete]
func (h *FriendHandler) DeleteFriend(w http.ResponseWriter, r *http.Request) {
	friendIDstr := chi.URLParam(r, "userID")
	friendID, err := strconv.Atoi(friendIDstr)
	if err != nil {
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	if friendID <= 0 {
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("session_id")
	if err != nil {
		utils.WriteError(w, xerrors.UnauthorizedStr, http.StatusUnauthorized)
		return
	}

	session, err := h.sessionService.Get(r.Context(), models.SessionID(cookie.Value))
	if err != nil {
		utils.WriteError(w, xerrors.UnauthorizedStr, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), session.UserID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	if err := h.friendshipService.DeleteFriend(r.Context(), profile.ID, int64(friendID)); err != nil {
		if errors.Is(err, xerrors.NoRowsAffected) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		utils.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// @Description	Send friendship request from current user to any user by ID
// @ID			request-friendship
// @Summary		Send friendship request
// @Tags		friend
// @Security	SessionAuth
// @Produce		json
// @Param 		profile_id	body friendRequest true "user ID to send friendship request to"
// @Success		200
// @Success		201
// @Failure		400		{object}	dto.CommonErrorResponse
// @Failure		401		{object}	dto.CommonErrorResponse
// @Failure		404		{object}	dto.CommonErrorResponse
// @Failure		500		{object}	dto.CommonErrorResponse
// @Router		/friends/request [post]
func (h *FriendHandler) RequestFriendship(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		utils.WriteError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	session, err := h.sessionService.Get(r.Context(), models.SessionID(cookie.Value))
	if err != nil {
		if errors.Is(err, xerrors.SessionNotFound) {
			utils.WriteError(w, xerrors.UnauthorizedStr, http.StatusUnauthorized)
			return
		}
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), session.UserID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	var request friendRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}

	// Ранее никто из них не был друзьями
	if areFriends, _ := h.friendshipService.CheckFriendship(r.Context(), profile.ID, request.FriendID); !areFriends {
		err := h.friendshipService.MakeFriends(r.Context(), profile.ID, request.FriendID, models.FriendshipPending)
		if err != nil {
			if errors.Is(err, xerrors.NoRowsAffected) {
				utils.WriteError(w, err.Error(), http.StatusNotFound)
				return
			}
			utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		return
	}

	// Если пользователь уже отправлял заявку
	if areFriends, _ := h.friendshipService.CheckFriendshipBy(r.Context(), profile.ID, request.FriendID); areFriends {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Если есть встречный запрос в друзья - автоматически принимается
	if areFriends, status := h.friendshipService.CheckFriendshipBy(r.Context(), request.FriendID, profile.ID); areFriends && status == models.FriendshipPending {
		err := h.friendshipService.AcceptFriendship(r.Context(), profile.ID, request.FriendID)
		if err != nil {
			if errors.Is(err, xerrors.NoRowsAffected) {
				utils.WriteError(w, err.Error(), http.StatusNotFound)
				return
			}
			utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// @Description	Get incoming friend requests with status
// @ID			get-incoming
// @Summary		Get incoming friend requests
// @Tags		friend
// @Security	SessionAuth
// @Produce		json
// @Param 		status 	path string false "friendship status"
// @Success		200		{object}	friendsResponse
// @Failure		400		{object}	dto.CommonErrorResponse
// @Failure		401		{object}	dto.CommonErrorResponse
// @Failure		404		{object}	dto.CommonErrorResponse
// @Failure		500		{object}	dto.CommonErrorResponse
// @Router		/friends/requests/incoming/{status} [get]
func (h *FriendHandler) GetIncomingFriendRequests(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if status == "" {
		status = string(models.FriendshipPending)
	}

	if status != string(models.FriendshipPending) &&
		status != string(models.FriendshipAccepted) {

		utils.WriteError(w, "Unknown status value", http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("session_id")
	if err != nil {
		utils.WriteError(w, xerrors.UnauthorizedStr, http.StatusUnauthorized)
		return
	}

	session, err := h.sessionService.Get(r.Context(), models.SessionID(cookie.Value))
	if err != nil {
		utils.WriteError(w, xerrors.UnauthorizedStr, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), session.UserID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	friends, err := h.friendshipService.GetIncomingFriends(r.Context(), profile.ID, status)
	if err != nil {
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	response := friendsResponse{
		Friends: friends,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// @Description	Get outgoing friend requests with status
// @ID			get-outgoing
// @Summary		Get outgoing friend requests
// @Tags		friend
// @Security	SessionAuth
// @Produce		json
// @Param 		status 	path string false "friendship status"
// @Success		200		{object}	friendsResponse
// @Failure		400		{object}	dto.CommonErrorResponse
// @Failure		401		{object}	dto.CommonErrorResponse
// @Failure		404		{object}	dto.CommonErrorResponse
// @Failure		500		{object}	dto.CommonErrorResponse
// @Router		/friends/requests/outgoing/{status} [get]
func (h *FriendHandler) GetOutgoingFriendRequests(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if status == "" {
		status = string(models.FriendshipPending)
	}

	if status != string(models.FriendshipPending) &&
		status != string(models.FriendshipAccepted) {

		utils.WriteError(w, "Unknown status value", http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("session_id")
	if err != nil {
		utils.WriteError(w, xerrors.UnauthorizedStr, http.StatusUnauthorized)
		return
	}

	session, err := h.sessionService.Get(r.Context(), models.SessionID(cookie.Value))
	if err != nil {
		utils.WriteError(w, xerrors.UnauthorizedStr, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), session.UserID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			utils.WriteError(w, "Profile not found", http.StatusNotFound)
			return
		}
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	friends, err := h.friendshipService.GetOutgoingFriends(r.Context(), profile.ID, status)
	if err != nil {
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	response := friendsResponse{
		Friends: friends,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// @Description	Decline a friend request from a user by its ID
// @ID			decline-friend
// @Summary		Decline a friend request
// @Tags		friend
// @Security	SessionAuth
// @Produce		json
// @Param 		requesterID 	path string true "users' id to de declined as friend"
// @Success		200
// @Failure		400		{object}	dto.CommonErrorResponse
// @Failure		401		{object}	dto.CommonErrorResponse
// @Failure		404		{object}	dto.CommonErrorResponse
// @Failure		500		{object}	dto.CommonErrorResponse
// @Router		/friends/decline/{requesterID} [post]
func (h *FriendHandler) DeclineFriendRequest(w http.ResponseWriter, r *http.Request) {
	requesterIDstr := chi.URLParam(r, "requesterID")
	requesterID, err := strconv.Atoi(requesterIDstr)
	if err != nil {
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	if requesterID <= 0 {
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("session_id")
	if err != nil {
		utils.WriteError(w, xerrors.UnauthorizedStr, http.StatusUnauthorized)
		return
	}

	session, err := h.sessionService.Get(r.Context(), models.SessionID(cookie.Value))
	if err != nil {
		utils.WriteError(w, xerrors.UnauthorizedStr, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), session.UserID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	areFriends, status := h.friendshipService.CheckFriendshipBy(r.Context(), int64(requesterID), profile.ID)

	if areFriends && status == models.FriendshipPending {
		err := h.friendshipService.DeclineFriendship(r.Context(), int64(requesterID), profile.ID)
		if err != nil {
			if errors.Is(err, xerrors.NoRowsAffected) {
				utils.WriteError(w, err.Error(), http.StatusNotFound)
				return
			}
			utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

// @Description	Accept a friend request from a user by its ID
// @ID			accept-friend
// @Summary		Accept a friend request
// @Tags		friend
// @Security	SessionAuth
// @Produce		json
// @Param 		requesterID 	path string true "users' id to de accepted as friend"
// @Success		200
// @Failure		400		{object}	dto.CommonErrorResponse
// @Failure		401		{object}	dto.CommonErrorResponse
// @Failure		404		{object}	dto.CommonErrorResponse
// @Failure		500		{object}	dto.CommonErrorResponse
// @Router		/friends/accept/{requesterID} [post]
func (h *FriendHandler) AcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	requesterIDstr := chi.URLParam(r, "requesterID")
	requesterID, err := strconv.Atoi(requesterIDstr)
	if err != nil {
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	if requesterID <= 0 {
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("session_id")
	if err != nil {
		utils.WriteError(w, xerrors.UnauthorizedStr, http.StatusUnauthorized)
		return
	}

	session, err := h.sessionService.Get(r.Context(), models.SessionID(cookie.Value))
	if err != nil {
		utils.WriteError(w, xerrors.UnauthorizedStr, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), session.UserID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	areFriends, status := h.friendshipService.CheckFriendshipBy(r.Context(), int64(requesterID), profile.ID)
	if areFriends && status == models.FriendshipPending {
		err := h.friendshipService.AcceptFriendship(r.Context(), int64(requesterID), profile.ID)
		if err != nil {
			if errors.Is(err, xerrors.NoRowsAffected) {
				utils.WriteError(w, err.Error(), http.StatusNotFound)
				return
			}
			utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

// @Description	Revoke a friend request to a user by its ID
// @ID			revoke-friend
// @Summary		Revoke a friend request
// @Tags		friend
// @Security	SessionAuth
// @Produce		json
// @Param 		addresseeID 	path string true "users' id to de revoked as friend"
// @Success		200
// @Failure		400		{object}	dto.CommonErrorResponse
// @Failure		401		{object}	dto.CommonErrorResponse
// @Failure		404		{object}	dto.CommonErrorResponse
// @Failure		500		{object}	dto.CommonErrorResponse
// @Router /friends/request/{addresseeID} [delete]
func (h *FriendHandler) RevokeFriendRequest(w http.ResponseWriter, r *http.Request) {
	addresseeIDstr := chi.URLParam(r, "addresseeID")
	addresseeID, err := strconv.Atoi(addresseeIDstr)
	if err != nil {
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	if addresseeID <= 0 {
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	cookie, err := r.Cookie("session_id")
	if err != nil {
		utils.WriteError(w, xerrors.UnauthorizedStr, http.StatusUnauthorized)
		return
	}

	session, err := h.sessionService.Get(r.Context(), models.SessionID(cookie.Value))
	if err != nil {
		utils.WriteError(w, xerrors.UnauthorizedStr, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), session.UserID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	err = h.friendshipService.RevokeFriendRequest(r.Context(), profile.ID, int64(addresseeID))
	if err != nil {
		if errors.Is(err, xerrors.NoRowsAffected) {
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
