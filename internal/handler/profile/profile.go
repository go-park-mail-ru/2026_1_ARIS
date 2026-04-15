package profile

import (
	"encoding/json"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type ProfileHandler struct {
	sessionService session.SessionService
	userService    user.UserService
	mediaService   media.MediaService
}

func NewProfileHandler(userService user.UserService, mediaService media.MediaService, sessionService session.SessionService) *ProfileHandler {
	return &ProfileHandler{
		userService:    userService,
		sessionService: sessionService,
		mediaService:   mediaService,
	}
}

type EducationResponse struct {
	Institution *string `json:"institution,omitempty"`
	Group       *string `json:"grade,omitempty"` // ? naming
}

type WorkResponse struct {
	Company  *string `json:"company,omitempty"`
	JobTitle *string `json:"jobTitle,omitempty"`
}

type GetProfileMeResponse struct {
	FirstName string  `json:"firstName,omitempty" validate:"required,alphaunicode,omitempty"`
	LastName  string  `json:"lastName,omitempty" validate:"required,alphaunicode,omitempty"`
	Bio       *string `json:"bio,omitempty" validate:"omitempty,omitnil"`
	ImageLink *string `json:"imageLink,omitempty" validate:"url,omitempty,omitnil"`

	Gender       models.Gender `json:"gender,omitempty" validate:"oneof=1 2,omitempty"`
	BirthdayDate string        `json:"birthday,omitempty" validate:"datetime=02/01/2006,omitempty"` // ? type
	NativeTown   *string       `json:"nativeTown,omitempty" validate:"omitempty,omitnil"`
	Phone        *string       `json:"phone,omitempty" validate:"omitempty,omitnil"`
	Email        *string       `json:"email,omitempty" validate:"omitempty,omitnil"`
	Town         *string       `json:"town,omitempty" validate:"omitempty,omitnil"`

	Education []EducationResponse `json:"education,omitempty" validate:"omitempty,omitzero"`

	Work []WorkResponse `json:"work,omitempty" validate:"omitempty,omitzero"`

	Interests *string `json:"interests,omitempty" validate:"omitempty,omitnil"`
	FavMusic  *string `json:"favMusic,omitempty" validate:"omitempty,omitnil"`
}

var validate = validator.New()

func (h *ProfileHandler) buildProfileResponse(r *http.Request, profileID int64) (*GetProfileMeResponse, int, string) {
	userProfile, err := h.userService.GetUserProfileByProfileID(r.Context(), profileID)
	if err != nil {
		return nil, http.StatusNotFound, "profile not found"
	}

	userAccount, err := h.userService.GetUserAccountByProfileID(r.Context(), profileID)
	if err != nil {
		return nil, http.StatusInternalServerError, "internal server error"
	}

	var imageLink *string
	if h.mediaService != nil {
		profile, err := h.userService.GetProfileByUserProfileID(r.Context(), userProfile.ID)
		if err == nil && profile != nil && profile.AvatarID != nil {
			avatar, avatarErr := h.mediaService.GetAvatarByID(r.Context(), profile.AvatarID)
			if avatarErr == nil && avatar != nil {
				imageLink = &avatar.Link
			}
		}
	}

	bio := userProfile.Bio
	if bio != nil {
		b := html.EscapeString(*bio)
		bio = &b
	}
	town := userProfile.Town
	if town != nil {
		t := html.EscapeString(*town)
		town = &t
	}
	interests := userProfile.Interests
	if interests != nil {
		i := html.EscapeString(*interests)
		interests = &i
	}
	fav := userProfile.FavMusic
	if fav != nil {
		t := html.EscapeString(*fav)
		fav = &t
	}
	institution := userProfile.Institution
	if institution != nil {
		t := html.EscapeString(*institution)
		institution = &t
	}
	group := userProfile.Group
	if group != nil {
		t := html.EscapeString(*group)
		group = &t
	}
	company := userProfile.Company
	if company != nil {
		t := html.EscapeString(*company)
		company = &t
	}
	jon := userProfile.JobTitle
	if jon != nil {
		t := html.EscapeString(*jon)
		jon = &t
	}

	response := &GetProfileMeResponse{
		FirstName:  html.EscapeString(userProfile.FirstName),
		LastName:   html.EscapeString(userProfile.LastName),
		Bio:        userProfile.Bio,
		ImageLink:  imageLink,
		Gender:     userProfile.Gender,
		NativeTown: userProfile.NativeTown,
		Phone:      userAccount.Phone,
		Email:      userAccount.Email,
		Town:       userProfile.Town,
		Interests:  userProfile.Interests,
		FavMusic:   userProfile.FavMusic,
		Education: []EducationResponse{{
			Institution: userProfile.Institution,
			Group:       userProfile.Group,
		}},
		Work: []WorkResponse{{
			Company:  userProfile.Company,
			JobTitle: userProfile.JobTitle,
		}},
	}

	if !userProfile.BirthdayDate.IsZero() {
		response.BirthdayDate = userProfile.BirthdayDate.Format(time.DateOnly)
	}

	return response, http.StatusOK, ""
}

// @Description		Get current user profile data
// @ID				get-profile
// @Summary			Get current profile
// @Tags			profile
// @Accept			json
// @Produce			json
// @Security		SessionAuth
// @Success			200	{object}	GetProfileMeResponse
// @Failure			401	{object}	dto.CommonErrorResponse
// @Failure			500	{object}	dto.CommonErrorResponse
// @Router			/profile/me [get]
func (h *ProfileHandler) GetProfileMe(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	cookie, err := r.Cookie("session_id")
	if err != nil {
		log.Warn("cannot_get_profile_me_missing_session",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, "Unauthorised", http.StatusUnauthorized)
		return
	}

	session, err := h.sessionService.Get(r.Context(), models.SessionID(cookie.Value))
	if err != nil {
		log.Warn("cannot_get_profile_me_invalid_session",
			zap.String("session_id", cookie.Value),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, "Unauthorised", http.StatusUnauthorized)
		return
	}

	userProfile, err := h.userService.GetUserProfileByUserAccountID(r.Context(), session.UserID)
	if err != nil {
		log.Error("failed_to_get_user_profile_by_account",
			zap.Int64("userAccount_id", session.UserID),
			zap.Error(err),
		)
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	response, status, message := h.buildProfileResponse(r, userProfile.ProfileID)
	if status != http.StatusOK {
		log.Warn("cannot_get_profile_me_build_response",
			zap.Int("status", status),
			zap.String("message", message),
			zap.Int64("profile_id", userProfile.ProfileID),
		)
		utils.WriteError(w, message, status)
		return
	}

	log.Info("profile_me_returned",
		zap.Int64("userAccount_id", session.UserID),
		zap.Int64("profile_id", userProfile.ProfileID),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// @Description		Get public profile data by profile id
// @ID				get-profile-by-id
// @Summary			Get profile by id
// @Tags			profile
// @Accept			json
// @Produce			json
// @Security		SessionAuth
// @Param			id	path		int	true	"Profile ID"
// @Success			200	{object}	GetProfileMeResponse
// @Failure			400	{object}	dto.CommonErrorResponse
// @Failure			401	{object}	dto.CommonErrorResponse
// @Failure			404	{object}	dto.CommonErrorResponse
// @Failure			500	{object}	dto.CommonErrorResponse
// @Router			/profile/{id} [get]
func (h *ProfileHandler) GetProfileByID(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	profileID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || profileID <= 0 {
		log.Warn("cannot_get_profile_by_id_invalid_id",
			zap.String("profile_id", chi.URLParam(r, "id")),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, "invalid profile id", http.StatusBadRequest)
		return
	}

	response, status, message := h.buildProfileResponse(r, profileID)
	if status != http.StatusOK {
		log.Warn("cannot_get_profile_by_id_build_response",
			zap.Int("status", status),
			zap.String("message", message),
			zap.Int64("profile_id", profileID),
		)
		utils.WriteError(w, message, status)
		return
	}

	log.Info("profile_by_id_returned",
		zap.Int64("profile_id", profileID),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// @Description		Edit current user profile
// @ID				edit-profile
// @Summary			Edit current profile
// @Tags			profile
// @Accept			json
// @Param			updated_fields	body	dto.UpdateFullProfileRequestDTO	true	"patch data"
// @Security		SessionAuth
// @Success			204
// @Failure			400	{object}	dto.CommonErrorResponse
// @Failure			401	{object}	dto.CommonErrorResponse
// @Failure			500	{object}	dto.CommonErrorResponse
// @Router			/profile/me/edit [patch]
func (h *ProfileHandler) EditProfileMe(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	cookie, err := r.Cookie("session_id")
	if err != nil {
		log.Warn("cannot_edit_profile_me_missing_session",
			zap.String("path", r.URL.Path),
		)
		utils.WriteError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	session, err := h.sessionService.Get(r.Context(), models.SessionID(cookie.Value))
	if err != nil {
		log.Warn("cannot_edit_profile_me_invalid_session",
			zap.String("session_id", cookie.Value),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userProfile, err := h.userService.GetUserProfileByUserAccountID(r.Context(), session.UserID)
	if err != nil {
		log.Error("failed_to_get_user_profile_for_edit",
			zap.Int64("userAccount_id", session.UserID),
			zap.Error(err),
		)
		utils.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	userAccount, err := h.userService.GetUserAccountByUserProfileID(r.Context(), userProfile.ID)
	if err != nil {
		log.Error("failed_to_get_user_account_for_edit",
			zap.Int64("profile_id", userProfile.ID),
			zap.Error(err),
		)
		utils.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var req dto.UpdateFullProfileRequestDTO

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {

		utils.WriteError(w, "Invalid body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	reqForValidation := req

	if reqForValidation.Username != nil && strings.TrimSpace(*reqForValidation.Username) == "" {
		reqForValidation.Username = nil
	}
	if reqForValidation.Email != nil && strings.TrimSpace(*reqForValidation.Email) == "" {
		reqForValidation.Email = nil
	}
	if reqForValidation.Phone != nil && strings.TrimSpace(*reqForValidation.Phone) == "" {
		reqForValidation.Phone = nil
	}

	if err := validate.Struct(reqForValidation); err != nil {
		log.Warn("cannot_edit_profile_me_validation_failed",
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		utils.WriteError(w, "validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	req.UserAccountID = userAccount.ID
	req.UserProfileID = userProfile.ID
	req.ProfileID = userProfile.ProfileID

	err = h.userService.UpdateMe(r.Context(), req)
	if err != nil {

		utils.WriteError(w, "Update failed", http.StatusInternalServerError)
		return
	}

	log.Info("profile_me_updated",
		zap.Int64("userAccount_id", userAccount.ID),
		zap.Int64("profile_id", userProfile.ProfileID),
	)

	w.WriteHeader(http.StatusNoContent)
}
