package profile

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-playground/validator/v10"
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
	BirthdayDate string        `json:"dirthday,omitempty" validate:"datetime=02/01/2006,omitempty"` // ? type
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
	cookie, err := r.Cookie("session_id")
	if err != nil {
		utils.WriteError(w, "Unauthorised", http.StatusUnauthorized)
		return
	}

	session, err := h.sessionService.Get(r.Context(), models.SessionID(cookie.Value))
	if err != nil {
		utils.WriteError(w, "Unauthorised", http.StatusUnauthorized)
		return
	}

	userProfile, err := h.userService.GetUserProfileByUserAccountID(r.Context(), session.UserID)
	if err != nil {
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	userAccount, err := h.userService.GetUserAccountByUserProfileID(r.Context(), userProfile.ID)
	if err != nil {
		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// profile, err := h.userService.GetProfileByUserProfileID(r.Context(), userProfile.ID)
	// if err != nil {
	// 	fmt.Println("ERROR 4:", err)
	// 	return
	// }

	// заглушка
	avatarLink := "http://arisnet.ru/assets/img/logo.svg"

	eduREsponse := EducationResponse{
		Institution: userProfile.Institution,
		Group:       userProfile.Group,
	}

	workResponse := WorkResponse{
		Company:  userProfile.Company,
		JobTitle: userProfile.JobTitle,
	}

	gender := "male"
	if userProfile.Gender == models.Female {
		gender = "female"
	}

	response := GetProfileMeResponse{
		FirstName:    userProfile.FirstName,
		LastName:     userProfile.LastName,
		Bio:          userProfile.Bio,
		ImageLink:    &avatarLink,
		Gender:       models.Gender(gender),
		BirthdayDate: userProfile.BirthdayDate.Format(time.DateOnly),
		NativeTown:   userProfile.NativeTown,
		Phone:        userAccount.Phone,
		Email:        userAccount.Email,
		Town:         userProfile.Town,
		Interests:    userProfile.Interests,
		FavMusic:     userProfile.FavMusic,
		Education:    []EducationResponse{eduREsponse},
		Work:         []WorkResponse{workResponse},
	}

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
	cookie, err := r.Cookie("session_id")
	if err != nil {
		utils.WriteError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	session, err := h.sessionService.Get(r.Context(), models.SessionID(cookie.Value))
	if err != nil {
		utils.WriteError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userProfile, err := h.userService.GetUserProfileByUserAccountID(r.Context(), session.UserID)
	if err != nil {
		utils.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	userAccount, err := h.userService.GetUserAccountByUserProfileID(r.Context(), userProfile.ID)
	if err != nil {
		utils.WriteError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var req dto.UpdateFullProfileRequestDTO

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Println(err)
		utils.WriteError(w, "Invalid body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := validate.Struct(req); err != nil {
		utils.WriteError(w, "validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	req.UserAccountID = userAccount.ID
	req.UserProfileID = userProfile.ID

	fmt.Println("Пошло на обновление !")
	fmt.Println(req)

	err = h.userService.UpdateMe(r.Context(), req)
	if err != nil {
		utils.WriteError(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
