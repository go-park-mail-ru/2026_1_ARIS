package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var validate = validator.New()

type RegisterRequest struct {
	FirstName string `json:"firstName" validate:"required,alphaunicode"`
	LastName  string `json:"lastName" validate:"required,alphaunicode"`
	Birthday  string `json:"birthday" validate:"required,min=8,max=10" example:"24/02/2005"`
	Login     string `json:"login" validate:"required,alphanumunicode"`
	Password1 string `json:"password1" validate:"required,min=6,max=72,printascii"`
	Password2 string `json:"password2" validate:"required,min=6,max=72,printascii"`
}

type LoginRequest struct {
	Login    string `json:"login" validate:"required,alphanumunicode"`
	Password string `json:"password" validate:"required"`
}

type AuthHandler struct {
	authService    service.AuthService
	sessionService service.SessionService
	userService    service.UserService
}

type LoginResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"createdAt"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type UserDTO struct {
	user        models.User
	userProfile models.UserProfile
	profile     models.Profile
}

type CommonResponse struct {
	Message string `json:"massage"`
}

type CommonErrorResponse struct {
	Message string `json:"error"`
}

func NewAuthHandler(authService service.AuthService, sessSvc service.SessionService, usService service.UserService) *AuthHandler {
	return &AuthHandler{
		authService:    authService,
		sessionService: sessSvc,
		userService:    usService,
	}
}

// @Description User registration
// @ID registration
// @Summary Register user
// @Tags auth
// @Accept json
// @Produce json
// @Param input body RegisterRequest true "post data"
// @Success 201 {object} models.Profile
// @Failure 400 {object} CommonResponse
// @Failure 409 {object} CommonResponse
// @Failure 500 {object} CommonResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, "неправильное тело запроса", http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		utils.WriteError(w, "validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Password1 != req.Password2 {
		utils.WriteError(w, "passwords dont match", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	profile, err := h.authService.Register(r.Context(), req.FirstName, req.LastName, req.Login, req.Password1, req.Birthday)
	if err != nil {
		if err.Error() == "пользователь с таким login уже существует" {
			utils.WriteError(w, "login already registered", http.StatusConflict)
			return
		} else if err.Error() == "invalid birthday date" {
			utils.WriteError(w, "invalid birthday date", http.StatusConflict)
			return
		} else if err.Error() == "you are too young, buddy" {
			utils.WriteError(w, "you are too young, buddy", http.StatusConflict)
			return
		}
		utils.WriteError(w, "ошибка на стороне сервера", http.StatusInternalServerError)
		return
	}

	user, err := h.userService.GetUserProfileByProfile(r.Context(), profile.ID)
	if err != nil {
		utils.WriteError(w, "Use not found", http.StatusInternalServerError)
		return
	}

	session, err := h.sessionService.Create(r.Context(), user.ID)
	if err != nil {
		utils.WriteError(w, "не удалось создать сессию", http.StatusInternalServerError)
		return
	}

	fmt.Println("кука поставлена для пользователя")
	fmt.Println(user.ID)
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    string(session.SessionID),
		Expires:  session.ExpiredAt,
		HttpOnly: true,
		Path:     "/",
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(profile)
}

// @Description User login
// @ID login
// @Summary Login user
// @Tags auth
// @Accept json
// @Produce json
// @Param input body LoginRequest true "post data"
// @Success 201 {object} LoginResponse
// @Failure 400 {object} CommonResponse
// @Failure 401 {object} CommonResponse
// @Failure 500 {object} CommonResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, "неверный запрос", http.StatusBadRequest)
		return
	}

	user, err := h.authService.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		utils.WriteError(w, "неверные учетные данные", http.StatusUnauthorized)
		return
	}

	session, err := h.sessionService.Create(r.Context(), user.ID)
	if err != nil {
		utils.WriteError(w, "не удалось создать сессию", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    string(session.SessionID),
		Expires:  session.ExpiredAt,
		HttpOnly: true,
		// Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	userProfile, err := h.userService.GetUserProfileByUser(r.Context(), user.ID)
	if err != nil {
		utils.WriteError(w, "user not found", http.StatusInternalServerError)
		return
	}

	loginResponse := LoginResponse{
		ID:        userProfile.ID.String(),
		CreatedAt: userProfile.CreatedAt.UTC().Format(time.RFC3339Nano),
		FirstName: userProfile.FirstName,
		LastName:  userProfile.LastName,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loginResponse)
}

// @Description User logout
// @ID lohout
// @Summary Logout user
// @Tags auth
// @Produce json
// @Security SessionAuth
// @Success 200 {object} CommonResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {

	cookie, err := r.Cookie("session_id")
	if err != nil {

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(CommonResponse{Message: "already logged out"})
		return
	}

	sessionID := models.SessionID(cookie.Value)

	if err := h.sessionService.Delete(r.Context(), sessionID); err != nil {

	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Path:     "/",
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CommonResponse{Message: "successfully logged out"})
}

// @Description Get current user from context
// @Summary Get current user
// @Tags auth
// @Produce json
// @Success 200 {object} models.User
// @Failure 401 {object} CommonResponse
// @Failure 404 {object} CommonResponse
// @Security SessionAuth
// @Router /auth/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Ответ me")
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		utils.WriteError(w, "не авторизован", http.StatusUnauthorized)
		return
	}

	fmt.Println(userID)
	user, err := h.userService.GetUserProfileByUserProfileID(userID)
	fmt.Println(user)
	if err != nil {
		utils.WriteError(w, "пользователь не найден", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}
