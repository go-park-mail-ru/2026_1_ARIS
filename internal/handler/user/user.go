package user

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/settings"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type UserHandler struct {
	UserService     user.UserService
	MediaService    media.MediaService
	SettingsService settings.UserSettingsService
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

func NewUserHandler(userService user.UserService, mediaService media.MediaService, settingsService settings.UserSettingsService) *UserHandler {
	return &UserHandler{
		UserService:     userService,
		MediaService:    mediaService,
		SettingsService: settingsService,
	}
}

var validate = validator.New()

// @Description		Get suggested users
// @ID				get-sug-users
// @Summary			Get suggested users
// @Tags			feed
// @Security		SessionAuth
// @Success			200 {object} 	suggestedUsersResponse
// @Failure			401	{object}	dto.CommonErrorResponse
// @Failure			500	{object}	dto.CommonErrorResponse
// @Router			/users/suggested [get]
func (h *UserHandler) GetSuggestedUsers(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())
	log.Info("called GetSuggestedUsers", zap.Time("call time", time.Now()))

	userIDFromCtx := r.Context().Value("user_id")
	if userIDFromCtx == nil {
		log.Warn("cannot_get_suggested_users_missing_user",
			zap.String("path", r.URL.Path),
		)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID, ok := userIDFromCtx.(int64)
	if !ok {
		log.Warn("cannot_get_suggested_users_invalid_user_id_type",
			zap.String("path", r.URL.Path),
			zap.Any("user_id", userIDFromCtx),
		)
		http.Error(w, "invalid user id in context", http.StatusUnauthorized)
		return
	}

	users, err := h.UserService.GetSuggestedUsers(r.Context(), userID)
	if err != nil {
		log.Error("failed_to_get_suggested_users",
			zap.Int64("userAccount_id", userID),
			zap.Error(err),
		)
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
			log.Warn("cannot_build_suggested_user_missing_profile",
				zap.Int64("profile_id", user.ID),
				zap.Error(err),
			)
			continue
		}
		userAccount, err := h.UserService.GetUserAccountByUserProfileID(r.Context(), userProfile.ID)
		if err != nil {
			log.Warn("cannot_build_suggested_user_missing_account",
				zap.Int64("profile_id", userProfile.ID),
				zap.Error(err),
			)
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

	log.Info("suggested_users_returned",
		zap.Int64("userAccount_id", userID),
		zap.Int("count", len(items)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(suggestedUsersResponse{
		Items: items,
	})
}

// @Description		Get public suggested users
// @ID				get-pub-pop-users
// @Summary			Get public suggested users
// @Tags			feed
// @Success			200 {object} 	suggestedUsersResponse
// @Failure			500	{object}	dto.CommonErrorResponse
// @Router			/public/popular-users [get]
func (h *UserHandler) GetPublicPopularUsers(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	users, err := h.UserService.GetPublicPopularUsers(r.Context())
	if err != nil {
		log.Error("failed_to_get_public_popular_users",
			zap.Error(err),
		)
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

	log.Info("public_popular_users_returned",
		zap.Int("count", len(items)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(suggestedUsersResponse{
		Items: items,
	})
}

// @Description		Get latest events
// @ID				get-latest-events
// @Summary			Get latest events
// @Tags			feed
// @Security		SessionAuth
// @Success			200 {object} 	suggestedUsersResponse
// @Failure			500	{object}	dto.CommonErrorResponse
// @Router			/users/latest-events [get]
func (h *UserHandler) GetLatestEvents(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	events, err := h.UserService.GetLatestEvents(r.Context())
	if err != nil {
		log.Error("failed_to_get_latest_events",
			zap.Error(err),
		)
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

	log.Info("latest_events_returned",
		zap.Int("count", len(items)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(latestEventsResponse{
		Items: items,
	})
}

func (h *UserHandler) SetSettings(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		log.Warn("cannot_set_settings_missing_user",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return
	}

	var request dto.UserSettingsUpdate

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Warn("cannot_set_settings_invalid_body",
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}

	if err := validate.Struct(request); err != nil {
		log.Warn("cannot_set_settings_validation_error",
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.ValidationError, http.StatusBadRequest)
		return
	}

	settings, err := h.SettingsService.Update(r.Context(), userAccountID, request)
	if errors.Is(err, xerrors.ErrNothingToUpdate) {
		settings, err = h.SettingsService.GetByUserID(r.Context(), userAccountID)
	}
	if err != nil {
		log.Error("failed_to_set_settings",
			zap.Int64("userAccount_id", userAccountID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	log.Info("settings_updated",
		zap.Int64("userAccount_id", userAccountID),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(settings)
}

func (h *UserHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok {
		log.Warn("cannot_get_settings_missing_user",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return
	}

	settings, err := h.SettingsService.GetByUserID(r.Context(), userAccountID)
	if err != nil {
		log.Error("failed_to_get_settings",
			zap.Int64("userAccount_id", userAccountID),
			zap.Error(err),
		)
		utils.WriteError(w, xerrors.InternalServerErrorStr, http.StatusInternalServerError)
		return
	}

	log.Info("settings_returned",
		zap.Int64("userAccount_id", userAccountID),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(settings)
}
