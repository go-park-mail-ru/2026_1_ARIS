package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-playground/validator/v10"
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
}

type UserDTO struct {
	user        models.User
	userProfile models.UserProfile
	profile     models.Profile
}

func NewAuthHandler(authService service.AuthService, sessSvc service.SessionService) *AuthHandler {
	return &AuthHandler{
		authService:    authService,
		sessionService: sessSvc,
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

	profile, err := h.authService.Register(r.Context(), req.FirstName, req.LastName, req.Login, req.Password1, req.Birthday)
	if err != nil {
		if err.Error() == "пользователь с таким email уже существует" {
			http.Error(w, `{"error":"email already registered"}`, http.StatusConflict)
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

	w.Header().Set("Content-Type", "application/json")
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

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    string(session.SessionID),
		Expires:  session.ExpiredAt,
		HttpOnly: true,
		Path:     "/",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
