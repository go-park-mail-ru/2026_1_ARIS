package friend

import (
	"encoding/json"
	"errors"
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
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"go.uber.org/zap"
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

// @Description	Getting current users' friends with optionaly status
// @ID			get-friends
// @Summary		Get friends
// @Tags		friend
// @Security	SessionAuth
// @Produce		json
// @Param status path string false "friendship status, default accepted"
// @Success		200		{object}	friendsResponse
// @Failure		400		{object}	dto.CommonErrorResponse
// @Failure		401		{object}	dto.CommonErrorResponse
// @Failure		404		{object}	dto.CommonErrorResponse
// @Failure		500		{object}	dto.CommonErrorResponse
// @Router		/friends/{status} [get]
func (h *FriendHandler) GetFriends(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	friendshipStatus := chi.URLParam(r, "status")
	if friendshipStatus == "" {
		friendshipStatus = string(models.FriendshipAccepted)
	}

	if friendshipStatus != string(models.FriendshipPending) &&
		friendshipStatus != string(models.FriendshipAccepted) {

		log.Warn("cannot_get_friends_invalid_status",
			zap.String("status", friendshipStatus),
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, "Unknown status value", http.StatusBadRequest)
		return
	}

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		log.Warn("cannot_get_friends_missing_user",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), userAccountID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_get_friends_profile_not_found",
				zap.Int64("userAccount_id", userAccountID),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_profile",
			zap.Int64("userAccount_id", userAccountID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	friends, err := h.friendshipService.GetFriends(r.Context(), profile.ID, models.FriendshipStatus(friendshipStatus))
	if err != nil {
		log.Error("failed_to_get_friends",
			zap.Int64("profile_id", profile.ID),
			zap.String("status", friendshipStatus),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	response := friendsResponse{
		Friends: friends,
	}

	log.Info("friends_returned",
		zap.Int64("profile_id", profile.ID),
		zap.String("status", friendshipStatus),
		zap.Int("count", len(friends)),
	)

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
	log := logger.FromContext(r.Context())

	profileIDstr := chi.URLParam(r, "id")
	profileID, err := strconv.Atoi(profileIDstr)
	if err != nil {
		log.Warn("cannot_get_users_friends_invalid_id",
			zap.String("profile_id", profileIDstr),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InvalidID, http.StatusBadRequest)
		return
	}

	if profileID <= 0 {
		log.Warn("cannot_get_users_friends_invalid_id",
			zap.Int("profile_id", profileID),
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidID, http.StatusBadRequest)
		return
	}

	profile, err := h.userService.GetProfileByProfileID(r.Context(), int64(profileID))
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_get_users_friends_profile_not_found",
				zap.Int64("profile_id", int64(profileID)),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_profile",
			zap.Int64("profile_id", int64(profileID)),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	friends, err := h.friendshipService.GetFriends(r.Context(), profile.ID, models.FriendshipAccepted)
	if err != nil {
		log.Error("failed_to_get_users_friends",
			zap.Int64("profile_id", profile.ID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	response := friendsResponse{
		Friends: friends,
	}

	log.Info("users_friends_returned",
		zap.Int64("profile_id", profile.ID),
		zap.Int("count", len(friends)),
	)

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
	log := logger.FromContext(r.Context())

	friendIDstr := chi.URLParam(r, "userID")
	friendID, err := strconv.Atoi(friendIDstr)
	if err != nil {
		log.Warn("cannot_delete_friend_invalid_id",
			zap.String("userID", friendIDstr),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	if friendID <= 0 {
		log.Warn("cannot_delete_friend_invalid_id",
			zap.Int("userID", friendID),
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		log.Warn("cannot_delete_friend_missing_user",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), userAccountID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_delete_friend_profile_not_found",
				zap.Int64("userAccount_id", userAccountID),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_profile",
			zap.Int64("userAccount_id", userAccountID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	if err := h.friendshipService.DeleteFriend(r.Context(), profile.ID, int64(friendID)); err != nil {
		switch {
		case errors.Is(err, xerrors.NoRowsAffected):
			log.Warn("cannot_delete_friend_not_found",
				zap.Int64("friend_id", int64(friendID)),
				zap.Int64("profile_id", profile.ID),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		case errors.Is(err, xerrors.MultipleRowsAffect):
			log.Error("failed_to_delete_friend_multiple_rows",
				zap.Int64("friend_id", int64(friendID)),
				zap.Int64("profile_id", profile.ID),
				zap.Error(err),
			)
			utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
			return
		default:
			log.Error("failed_to_delete_friend",
				zap.Int64("friend_id", int64(friendID)),
				zap.Int64("profile_id", profile.ID),
				zap.Error(err),
			)
			utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
			return
		}
	}

	log.Info("friend_deleted",
		zap.Int64("friend_id", int64(friendID)),
		zap.Int64("profile_id", profile.ID),
	)

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
	log := logger.FromContext(r.Context())

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		log.Warn("cannot_request_friendship_missing_user",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), userAccountID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_request_friendship_profile_not_found",
				zap.Int64("userAccount_id", userAccountID),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_profile",
			zap.Int64("userAccount_id", userAccountID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	var request friendRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Warn("cannot_request_friendship_invalid_body",
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}

	err = h.friendshipService.MakeFriends(r.Context(), profile.ID, request.FriendID)
	if err != nil {
		switch {
		case errors.Is(err, xerrors.NoRowsAffected):
			log.Warn("cannot_request_friendship_not_found",
				zap.Int64("friend_id", request.FriendID),
				zap.Int64("profile_id", profile.ID),
				zap.Error(err),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		case errors.Is(err, xerrors.MultipleRowsAffect):
			log.Error("failed_to_request_friendship_multiple_rows",
				zap.Int64("friend_id", request.FriendID),
				zap.Int64("profile_id", profile.ID),
				zap.Error(err),
			)
			utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
			return
		case errors.Is(err, xerrors.AllreadyExists):
			log.Warn("cannot_request_friendship_already_exists",
				zap.Int64("friend_id", request.FriendID),
				zap.Int64("profile_id", profile.ID),
			)
			utils.WriteError(w, err.Error(), http.StatusConflict)
			return
		default:
			log.Error("failed_to_request_friendship",
				zap.Int64("friend_id", request.FriendID),
				zap.Int64("profile_id", profile.ID),
				zap.Error(err),
			)
			utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
			return
		}
	}

	log.Info("friend_request_sent",
		zap.Int64("profile_id", profile.ID),
		zap.Int64("friend_id", request.FriendID),
	)

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
	log := logger.FromContext(r.Context())

	status := chi.URLParam(r, "status")
	if status == "" {
		status = string(models.FriendshipPending)
	}

	if status != string(models.FriendshipPending) &&
		status != string(models.FriendshipAccepted) {

		log.Warn("cannot_get_incoming_friend_requests_invalid_status",
			zap.String("status", status),
			zap.String("path", r.URL.Path),
		)

		utils.WriteError(w, "Unknown status value", http.StatusBadRequest)
		return
	}

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		log.Warn("cannot_get_incoming_friend_requests_missing_user",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), userAccountID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_get_incoming_friend_requests_profile_not_found",
				zap.Int64("userAccount_id", userAccountID),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_profile",
			zap.Int64("userAccount_id", userAccountID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	friends, err := h.friendshipService.GetIncomingFriends(r.Context(), profile.ID, status)
	if err != nil {
		log.Error("failed_to_get_incoming_friend_requests",
			zap.Int64("profile_id", profile.ID),
			zap.String("status", status),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	response := friendsResponse{
		Friends: friends,
	}

	log.Info("incoming_friend_requests_returned",
		zap.Int64("profile_id", profile.ID),
		zap.String("status", status),
		zap.Int("count", len(friends)),
	)

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
	log := logger.FromContext(r.Context())

	status := chi.URLParam(r, "status")
	if status == "" {
		status = string(models.FriendshipPending)
	}

	if status != string(models.FriendshipPending) &&
		status != string(models.FriendshipAccepted) {

		log.Warn("cannot_get_outgoing_friend_requests_invalid_status",
			zap.String("status", status),
			zap.String("path", r.URL.Path),
		)

		utils.WriteError(w, "Unknown status value", http.StatusBadRequest)
		return
	}

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		log.Warn("cannot_get_outgoing_friend_requests_missing_user",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), userAccountID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_get_outgoing_friend_requests_profile_not_found",
				zap.Int64("userAccount_id", userAccountID),
			)
			utils.WriteError(w, "Profile not found", http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_profile",
			zap.Int64("userAccount_id", userAccountID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	friends, err := h.friendshipService.GetOutgoingFriends(r.Context(), profile.ID, status)
	if err != nil {
		log.Error("failed_to_get_outgoing_friend_requests",
			zap.Int64("profile_id", profile.ID),
			zap.String("status", status),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	response := friendsResponse{
		Friends: friends,
	}

	log.Info("outgoing_friend_requests_returned",
		zap.Int64("profile_id", profile.ID),
		zap.String("status", status),
		zap.Int("count", len(friends)),
	)

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
	log := logger.FromContext(r.Context())

	requesterIDstr := chi.URLParam(r, "requesterID")
	requesterID, err := strconv.Atoi(requesterIDstr)
	if err != nil {
		log.Warn("cannot_decline_friend_request_invalid_id",
			zap.String("requesterID", requesterIDstr),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	if requesterID <= 0 {
		log.Warn("cannot_decline_friend_request_invalid_id",
			zap.Int("requesterID", requesterID),
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		log.Warn("cannot_decline_friend_request_missing_user",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), userAccountID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_decline_friend_request_profile_not_found",
				zap.Int64("userAccount_id", userAccountID),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_profile",
			zap.Int64("userAccount_id", userAccountID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	err = h.friendshipService.DeclineFriendship(r.Context(), int64(requesterID), profile.ID)
	if err != nil {
		switch {
		case errors.Is(err, xerrors.NoRowsAffected):
			log.Warn("cannot_decline_friend_request_not_found",
				zap.Int64("requester_id", int64(requesterID)),
				zap.Int64("profile_id", profile.ID),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		case errors.Is(err, xerrors.MultipleRowsAffect):
			log.Error("failed_to_decline_friend_request_multiple_rows",
				zap.Int64("requester_id", int64(requesterID)),
				zap.Int64("profile_id", profile.ID),
				zap.Error(err),
			)
			utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
			return
		default:
			log.Error("failed_to_decline_friend_request",
				zap.Int64("requester_id", int64(requesterID)),
				zap.Int64("profile_id", profile.ID),
				zap.Error(err),
			)
			utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
			return
		}
	}

	log.Info("friend_request_declined",
		zap.Int64("requester_id", int64(requesterID)),
		zap.Int64("profile_id", profile.ID),
	)

	w.WriteHeader(http.StatusOK)
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
	log := logger.FromContext(r.Context())

	requesterIDstr := chi.URLParam(r, "requesterID")
	requesterID, err := strconv.Atoi(requesterIDstr)
	if err != nil {
		log.Warn("cannot_accept_friend_request_invalid_id",
			zap.String("requesterID", requesterIDstr),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	if requesterID <= 0 {
		log.Warn("cannot_accept_friend_request_invalid_id",
			zap.Int("requesterID", requesterID),
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		log.Warn("cannot_accept_friend_request_missing_user",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), userAccountID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_accept_friend_request_profile_not_found",
				zap.Int64("userAccount_id", userAccountID),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_profile",
			zap.Int64("userAccount_id", userAccountID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	err = h.friendshipService.AcceptFriendship(r.Context(), int64(requesterID), profile.ID)
	if err != nil {
		switch {
		case errors.Is(err, xerrors.NoRowsAffected):
			log.Warn("cannot_accept_friend_request_not_found",
				zap.Int64("requester_id", int64(requesterID)),
				zap.Int64("profile_id", profile.ID),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		case errors.Is(err, xerrors.MultipleRowsAffect):
			log.Error("failed_to_accept_friend_request_multiple_rows",
				zap.Int64("requester_id", int64(requesterID)),
				zap.Int64("profile_id", profile.ID),
				zap.Error(err),
			)
			utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
			return
		default:
			log.Error("failed_to_accept_friend_request",
				zap.Int64("requester_id", int64(requesterID)),
				zap.Int64("profile_id", profile.ID),
				zap.Error(err),
			)
			utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
			return
		}
	}

	log.Info("friend_request_accepted",
		zap.Int64("requester_id", int64(requesterID)),
		zap.Int64("profile_id", profile.ID),
	)

	w.WriteHeader(http.StatusOK)
}

// @Description	Revoke a friend request to user by ID
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
	log := logger.FromContext(r.Context())

	addresseeIDstr := chi.URLParam(r, "addresseeID")
	addresseeID, err := strconv.Atoi(addresseeIDstr)
	if err != nil {
		log.Warn("cannot_revoke_friend_request_invalid_id",
			zap.String("addresseeID", addresseeIDstr),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	if addresseeID <= 0 {
		log.Warn("cannot_revoke_friend_request_invalid_id",
			zap.Int("addresseeID", addresseeID),
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, "Invalid ID parameter", http.StatusBadRequest)
		return
	}

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		log.Warn("cannot_revoke_friend_request_missing_user",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return
	}

	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), userAccountID)
	if err != nil {
		if errors.Is(err, xerrors.ProfileNotFound) {
			log.Warn("cannot_revoke_friend_request_profile_not_found",
				zap.Int64("userAccount_id", userAccountID),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		}
		log.Error("failed_to_get_profile",
			zap.Int64("userAccount_id", userAccountID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	err = h.friendshipService.RevokeFriendRequest(r.Context(), profile.ID, int64(addresseeID))
	if err != nil {
		switch {
		case errors.Is(err, xerrors.NoRowsAffected):
			log.Warn("cannot_revoke_friend_request_not_found",
				zap.Int64("addressee_id", int64(addresseeID)),
				zap.Int64("profile_id", profile.ID),
			)
			utils.WriteError(w, err.Error(), http.StatusNotFound)
			return
		case errors.Is(err, xerrors.MultipleRowsAffect):
			log.Error("failed_to_revoke_friend_request_multiple_rows",
				zap.Int64("addressee_id", int64(addresseeID)),
				zap.Int64("profile_id", profile.ID),
				zap.Error(err),
			)
			utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
			return
		default:
			log.Error("failed_to_revoke_friend_request",
				zap.Int64("addressee_id", int64(addresseeID)),
				zap.Int64("profile_id", profile.ID),
				zap.Error(err),
			)
			utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
			return
		}
	}

	log.Info("friend_request_revoked",
		zap.Int64("addressee_id", int64(addresseeID)),
		zap.Int64("profile_id", profile.ID),
	)

	w.WriteHeader(http.StatusOK)
}
