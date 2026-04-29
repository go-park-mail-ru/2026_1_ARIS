package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/auth/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

const sessionCookieName = "session_id"

type Handler struct {
	auth         *service.Service
	cookieSecure bool
}

func New(auth *service.Service, cookieSecure bool) *Handler {
	return &Handler{auth: auth, cookieSecure: cookieSecure}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/register/step-one", h.RegisterStepOne)
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/logout", h.Logout)
	r.Get("/me", h.Me)
}

func (h *Handler) RegisterStepOne(w http.ResponseWriter, r *http.Request) {
	var req registerStepOneRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	err := h.auth.RegisterStepOne(r.Context(), service.RegisterStepOneInput{
		Login:     req.Login,
		Password1: req.Password1,
		Password2: req.Password2,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	result, err := h.auth.Register(r.Context(), service.RegisterInput{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Login:     req.Login,
		Password1: req.Password1,
		Password2: req.Password2,
		Birthday:  req.Birthday,
		Gender:    parseGender(req.Gender),
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	h.setSessionCookie(w, result.Session.SessionID, result.Session.ExpiredAt)
	writeJSON(w, http.StatusCreated, mapUser(result.User))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	result, err := h.auth.Login(r.Context(), service.LoginInput{Login: req.Login, Password: req.Password})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	h.setSessionCookie(w, result.Session.SessionID, result.Session.ExpiredAt)
	writeJSON(w, http.StatusOK, mapUser(result.User))
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		_ = h.auth.Logout(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   h.cookieSecure,
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	user, err := h.auth.GetMe(r.Context(), cookie.Value)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, meResponse{
		ID:         strconv.FormatInt(user.UserAccountID, 10),
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		AvatarLink: user.AvatarURL,
	})
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, id models.SessionID, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    string(id),
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   h.cookieSecure,
		Path:     "/",
	})
}

func parseGender(value int) models.Gender {
	if value == 1 {
		return models.Male
	}
	return models.Female
}

func mapUser(user service.User) userResponse {
	return userResponse{
		ID:            user.ProfileID,
		UserAccountID: user.UserAccountID,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		AvatarURL:     user.AvatarURL,
		CreatedAt:     user.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return false
	}
	return true
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrLoginAlreadyExists):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "login already exists"})
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, service.ErrSessionNotFound):
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
	case errors.Is(err, service.ErrInvalidInput), errors.Is(err, service.ErrInvalidBirthday), errors.Is(err, service.ErrTooYoung):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
