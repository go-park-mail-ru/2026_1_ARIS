package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Username string `json:"username" validate:"required,min=3"`
	Phone    string `json:"phone" validate:"omitempty,e164"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthHandler struct {
	authService    *service.AuthService
	sessionService service.SessionService
}

func NewAuthHandler(authService *service.AuthService, sessSvc service.SessionService) *AuthHandler {
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "validation failed: " + err.Error(),
		})
		return
	}

	user, err := h.authService.Register(r.Context(), req.Email, req.Password, req.Username, req.Phone)
	if err != nil {
		if err.Error() == "пользователь с таким email уже существует" {
			http.Error(w, `{"error":"email already registered"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"ошибка на стороне сервера"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"неверный запрос"}`, http.StatusBadRequest)
		return
	}

	user, err := h.authService.Login(r.Context(), req.Email, req.Password)
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
