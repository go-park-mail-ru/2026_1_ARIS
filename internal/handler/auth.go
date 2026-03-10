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
	Birthday  string `json:"birthday" validate:"required,min=8,max=10"`
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

type UserDTO struct {
	user        models.User
	userProfile models.UserProfile
	profile     models.Profile
}

func NewAuthHandler(authService service.AuthService, sessSvc service.SessionService, usService service.UserService) *AuthHandler {
	return &AuthHandler{
		authService:    authService,
		sessionService: sessSvc,
		userService:    usService,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"неправильное тело запроса"}`, http.StatusBadRequest)
		return
	}

	if err := validate.Struct(req); err != nil {
		utils.WriteError(w, "validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Password1 != req.Password2 {
		http.Error(w, "passwords dont match", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	profile, err := h.authService.Register(r.Context(), req.FirstName, req.LastName, req.Login, req.Password1, req.Birthday)
	if err != nil {
		if err.Error() == "пользователь с таким login уже существует" {
			http.Error(w, `{"error":"login already registered"}`, http.StatusConflict)
			return
		} else if err.Error() == "invalid birthday date" {
			http.Error(w, `{"error":"invalid birthday date"}`, http.StatusConflict)
			return
		} else if err.Error() == "you are too young, buddy" {
			http.Error(w, `{"error":"you are too young, buddy"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"ошибка на стороне сервера"}`, http.StatusInternalServerError)
		return
	}

	user, err := h.userService.GerUserProfileByProfile(r.Context(), profile.ID)
	if err != nil {
		fmt.Println("Errrrrrrrrrrrrrrrrrrrrrr")
		return
	}

	session, err := h.sessionService.Create(r.Context(), user.ID)
	if err != nil {
		http.Error(w, `{"error":"не удалось создать сессию"}`, http.StatusInternalServerError)
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

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"неверный запрос"}`, http.StatusBadRequest)
		return
	}

	user, err := h.authService.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		http.Error(w, `{"error":"неверные учетные данные"}`, http.StatusUnauthorized)
		return
	}

	session, err := h.sessionService.Create(r.Context(), user.ID)
	if err != nil {
		http.Error(w, `{"error":"не удалось создать сессию"}`, http.StatusInternalServerError)
		return
	}

	fmt.Println("кука поставлена")
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    string(session.SessionID),
		Expires:  session.ExpiredAt,
		HttpOnly: true,
		Path:     "/",
	})
	fmt.Println(user)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {

	cookie, err := r.Cookie("session_id")
	if err != nil {

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "already logged out"})
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
	json.NewEncoder(w).Encode(map[string]string{"message": "successfully logged out"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Ответ me")
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		fmt.Println("error: не авторизован")
		http.Error(w, `{"error":"не авторизован"}`, http.StatusUnauthorized)
		return
	}

	fmt.Println(userID)
	//user, err := h.userService.GerUserProfileByProfile(r.Context(), userID)
	user, err := h.userService.GetUserProfileByUserProfileID(userID)
	fmt.Println(user)
	if err != nil {
		fmt.Println("error: пользователь не найден")
		http.Error(w, `{"error":"пользователь не найден"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}
