package auth

import (
	"encoding/json"
	"html"
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/auth"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

const (
	ErrUserExists      = "пользователь с таким login уже существует"
	ErrInvalidBirthday = "invalid birthday date"
	ErrTooYoung        = "you are too young, buddy"
	ErrUserNotFound    = "user not found"
)

var validate = validator.New()

type RegisterRequest struct {
	FirstName string `json:"firstName" validate:"required,alphaunicode"`
	LastName  string `json:"lastName" validate:"required,alphaunicode"`
	Birthday  string `json:"birthday" validate:"required,min=8,max=10,datetime=02/01/2006" example:"24/02/2005"`
	Gender    int    `json:"gender" validate:"required,oneof=1 2"`
	Login     string `json:"login" validate:"required,alphanumunicode"`
	Password1 string `json:"password1" validate:"required,min=6,max=72,printascii,eqfield=Password2"`
	Password2 string `json:"password2" validate:"required,min=6,max=72,printascii"`
}

type RegisterStepOneRequest struct {
	Login     string `json:"login" validate:"required,alphanumunicode"`
	Password1 string `json:"password1" validate:"required,min=6,max=72,printascii"`
	Password2 string `json:"password2" validate:"required,min=6,max=72,printascii"`
}

type ValidationErrorsResponse struct {
	Ok     bool              `json:"ok"`
	Errors map[string]string `json:"errors"`
}

type LoginRequest struct {
	Login    string `json:"login" validate:"required,alphanumunicode"`
	Password string `json:"password" validate:"required"`
}

type AuthHandler struct {
	authService    auth.AuthService
	sessionService session.SessionService
	userService    user.UserService
	mediaService   media.MediaService
}

type LoginResponse struct {
	ID         string `json:"id"`
	CreatedAt  string `json:"createdAt"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	AvatarLink string `json:"avatarLink,omitempty"`
}

type CommonResponse struct {
	Message string `json:"message"`
}

func NewAuthHandler(
	authService auth.AuthService,
	sessSvc session.SessionService,
	usService user.UserService,
	mediaServices ...media.MediaService,
) *AuthHandler {
	var mediaService media.MediaService
	if len(mediaServices) > 0 {
		mediaService = mediaServices[0]
	}

	return &AuthHandler{
		authService:    authService,
		sessionService: sessSvc,
		userService:    usService,
		mediaService:   mediaService,
	}
}

// @Description	User registration
// @ID			registration
// @Summary		Register user
// @Tags		auth
// @Accept		json
// @Produce		json
// @Param		input	body		RegisterRequest	true	"post data"
// @Success		201		{object}	models.Profile
// @Failure		400		{object}	dto.CommonErrorResponse
// @Failure		404		{object}	dto.CommonErrorResponse
// @Failure		409		{object}	dto.CommonErrorResponse
// @Failure		500		{object}	dto.CommonErrorResponse
// @Router		/auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("http_request", zap.String("auth", "register"))
		utils.WriteError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		log.Warn("http_request", zap.String("auth", "register"))
		utils.WriteError(w, "validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Password1 != req.Password2 {
		log.Warn("http_request", zap.String("auth", "register"))
		utils.WriteError(w, "passwords do not match", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var gender models.Gender = models.Female

	if req.Gender == 1 {
		gender = models.Male
	} else {
		gender = models.Female
	}

	profile, err := h.authService.Register(
		r.Context(),
		req.FirstName,
		req.LastName,
		req.Login,
		req.Password1,
		req.Birthday,
		//models.Gender(req.Gender-1),
		gender,
	)
	if err != nil {
		errMsg := err.Error()

		switch errMsg {
		case ErrUserExists:
			utils.WriteError(w, "login already exists", http.StatusConflict)
		case ErrInvalidBirthday:
			utils.WriteError(w, "invalid birthday date", http.StatusBadRequest)
		case ErrTooYoung:
			utils.WriteError(w, "user is too young", http.StatusBadRequest)
		default:
			utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	userProfile, err := h.userService.GetUserProfileByProfileID(r.Context(), profile.ID)
	if err != nil {
		utils.WriteError(w, "user profile not found", http.StatusInternalServerError)
		return
	}
	userAccount, err := h.userService.GetUserAccountByUserProfileID(r.Context(), userProfile.ID)
	if err != nil {
		utils.WriteError(w, "user account not found", http.StatusInternalServerError)
		return
	}

	session, err := h.sessionService.Create(r.Context(), userAccount.ID)
	if err != nil {
		log.Warn("failed to create session", zap.Int64("userAccount_id", userAccount.ID))
		utils.WriteError(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    string(session.SessionID),
		Expires:  session.ExpiredAt,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
		Path:     "/",
	})

	log.Info("user registered", zap.Int64("profile_id", profile.ID), zap.Int64("userAccount_id", userAccount.ID))
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(profile)
}

// @Description	User login
// @ID			login
// @Summary		Login user
// @Tags		auth
// @Accept		json
// @Produce		json
// @Param		input	body		LoginRequest	true	"post data"
// @Success		200		{object}	LoginResponse
// @Failure		400		{object}	CommonResponse
// @Failure		401		{object}	CommonResponse
// @Failure		404		{object}	CommonResponse
// @Failure		500		{object}	CommonResponse
// @Router		/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, "invalid request", http.StatusBadRequest)
		return
	}

	userAccount, err := h.authService.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		log.Warn("auth_login_failed", zap.String("auth", "login"))
		utils.WriteError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	userProfile, err := h.userService.GetUserProfileByUserID(r.Context(), userAccount.ID)
	if err != nil {
		errMsg := err.Error()
		if errMsg == ErrUserNotFound {
			utils.WriteError(w, ErrUserNotFound, http.StatusNotFound)
			return
		}

		utils.WriteError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	session, err := h.sessionService.Create(r.Context(), userAccount.ID)
	if err != nil {
		log.Error("failed_create_session",
			zap.Int64("userAccount_id", userAccount.ID),
			zap.Error(err),
		)
		utils.WriteError(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    string(session.SessionID),
		Expires:  session.ExpiredAt,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
		Path:     "/",
	})

	loginResponse := LoginResponse{
		ID:        strconv.FormatInt(userProfile.ID, 10),
		CreatedAt: userProfile.CreatedAt.UTC().Format(time.RFC3339Nano),
		FirstName: html.EscapeString(userProfile.FirstName),
		LastName:  html.EscapeString(userProfile.LastName),
	}

	log.Info("user_logged_in",
		zap.Int64("userAccount_id", userAccount.ID),
		zap.String("login", req.Login),
		zap.Int64("profile_id", userProfile.ID),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(loginResponse)
}

// @Description	User logout
// @ID			logout
// @Summary		Logout user
// @Tags		auth
// @Produce		json
// @Security	SessionAuth
// @Success		200	{object}	CommonResponse
// @Router		/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	cookie, err := r.Cookie("session_id")
	if err != nil {
		log.Info("logout_already_logged_out", zap.String("message", "no session cookie"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(CommonResponse{Message: "already logged out"})
		return
	}

	sessionID := models.SessionID(cookie.Value)

	if err := h.sessionService.Delete(r.Context(), sessionID); err != nil {
		log.Warn("failed_delete_session",
			zap.String("session_id", string(sessionID)),
			zap.Error(err),
		)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
		Path:     "/",
	})

	log.Info("user_logged_out",
		zap.String("session_id", string(sessionID)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CommonResponse{Message: "successfully logged out"})
}

// @Description	Get current user from context
// @Summary		Get current user
// @Tags		auth
// @Produce		json
// @Success		200	{object}	models.UserAccount
// @Failure		401	{object}	CommonResponse
// @Failure		404	{object}	CommonResponse
// @Security	SessionAuth
// @Router		/auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	userIDFromCtx := r.Context().Value("user_id")
	if userIDFromCtx == nil {
		log.Warn("unauthorized", zap.String("message", "user id not found in context"), zap.String("method", r.Method), zap.String("path", r.URL.Path))
		utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID, ok := userIDFromCtx.(int64)
	if !ok {
		utils.WriteError(w, "invalid user id in context", http.StatusUnauthorized)
		return
	}

	//userProfile, err := h.userService.GetUserProfileByUser(r.Context(), userID)

	userProfile, err := h.userService.GetUserProfileByUserAccountID(r.Context(), userID)

	if err != nil {
		log.Warn("user_profile_not_found",
			zap.Int64("userAccount_id", userID),
			zap.Error(err),
		)
		utils.WriteError(w, ErrUserNotFound, http.StatusNotFound)
		return
	}

	avatar := ""
	profile, err := h.userService.GetProfileByUserAccountID(r.Context(), userID)
	if err == nil && profile != nil && profile.AvatarID != nil && h.mediaService != nil {
		media, mediaErr := h.mediaService.GetAvatarByID(r.Context(), profile.AvatarID)
		if mediaErr == nil && media != nil {
			avatar = media.Link
		}
	}

	log.Info("me_request_success",
		zap.Int64("userAccount_id", userID),
		zap.Int64("profile_id", userProfile.ProfileID),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(LoginResponse{
		ID:         strconv.FormatInt(userID, 10),
		FirstName:  html.EscapeString(userProfile.FirstName),
		LastName:   html.EscapeString(userProfile.LastName),
		AvatarLink: avatar,
	})
}

// @Description	Validate register first step
// @ID			validate-register-step-one
// @Summary		Validate register step one
// @Tags		auth
// @Accept		json
// @Produce		json
// @Param		input	body		RegisterStepOneRequest	true	"step one data"
// @Success		200		{object}	ValidationErrorsResponse
// @Failure		400		{object}	CommonResponse
// @Failure		500		{object}	CommonResponse
// @Router		/auth/register/step-one [post]
func (h *AuthHandler) ValidateRegisterStepOne(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	var req RegisterStepOneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("http_request", zap.String("auth", "validate_register_step_one"))
		utils.WriteError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	errorsMap := make(map[string]string)

	if err := validate.Struct(req); err != nil {
		if req.Login == "" {
			errorsMap["login"] = "Обязательное поле"
		}
		if req.Password1 == "" {
			errorsMap["password1"] = "Обязательное поле"
		}
		if req.Password2 == "" {
			errorsMap["password2"] = "Обязательное поле"
		}
	}

	if req.Password1 != "" && len(req.Password1) < 7 {
		errorsMap["password1"] = "Пароль слишком короткий (мин. 7 символов)"
	}

	if req.Password1 != "" && len(req.Password1) > 20 {
		errorsMap["password1"] = "Пароль может содержать максимум 20 символов"
	}

	if req.Password2 != "" && req.Password1 != req.Password2 {
		errorsMap["password2"] = "Пароли не совпадают"
	}

	if req.Login != "" {
		if len(req.Login) < 6 {
			errorsMap["login"] = "Логин слишком короткий (мин. 6 символов)"
		} else if len(req.Login) > 12 {
			errorsMap["login"] = "Логин может содержать максимум 12 символов"
		}
	}

	if len(errorsMap) == 0 {
		serviceErrors, err := h.authService.ValidateRegisterStepOne(
			r.Context(),
			req.Login,
			req.Password1,
			req.Password2,
		)
		if err != nil {
			utils.WriteError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		for key, value := range serviceErrors {
			errorsMap[key] = value
		}
	}

	log.Info("validate_register_step_one",
		zap.String("login", req.Login),
		zap.Bool("ok", len(errorsMap) == 0),
		zap.Int("errors", len(errorsMap)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ValidationErrorsResponse{
		Ok:     len(errorsMap) == 0,
		Errors: errorsMap,
	})
}
