package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
)

const sessionCookieName = "session_id"

type Handler struct {
	authUsecase  *usecase.Service
	cookieSecure bool
	roles        roleProvider
}

type roleProvider interface {
	GetProfileRole(ctx context.Context, profileID int64) (model.SupportRole, error)
}

func New(authUsecase *usecase.Service, cookieSecure bool, roles ...roleProvider) *Handler {
	var provider roleProvider
	if len(roles) > 0 {
		provider = roles[0]
	}
	return &Handler{authUsecase: authUsecase, cookieSecure: cookieSecure, roles: provider}
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
	if !utils.DecodeJSON(w, r, &req) {
		return
	}

	err := h.authUsecase.RegisterStepOne(r.Context(), usecase.RegisterStepOneInput{
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
	if !utils.DecodeJSON(w, r, &req) {
		return
	}

	result, err := h.authUsecase.Register(r.Context(), usecase.RegisterInput{
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
	utils.WriteJSON(w, http.StatusCreated, h.mapUser(r.Context(), result.User))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}

	result, err := h.authUsecase.Login(r.Context(), usecase.LoginInput{Login: req.Login, Password: req.Password})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	h.setSessionCookie(w, result.Session.SessionID, result.Session.ExpiredAt)
	utils.WriteJSON(w, http.StatusOK, h.mapUser(r.Context(), result.User))
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		_ = h.authUsecase.Logout(r.Context(), cookie.Value)
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
	utils.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		utils.WriteJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	user, err := h.authUsecase.GetMe(r.Context(), cookie.Value)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, meResponse{
		ID:         strconv.FormatInt(user.ProfileID, 10),
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Login:      user.Login,
		Email:      utils.DerefString(user.Email),
		Role:       string(h.supportRole(r.Context(), user.ProfileID)),
		AvatarLink: user.AvatarURL,
	})
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, id model.SessionID, expiresAt time.Time) {
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

func parseGender(value int) model.Gender {
	if value == 1 {
		return model.Male
	}
	return model.Female
}

func (h *Handler) mapUser(ctx context.Context, user usecase.User) userResponse {
	return userResponse{
		ID:            user.ProfileID,
		UserAccountID: user.UserAccountID,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Login:         user.Login,
		Email:         utils.DerefString(user.Email),
		Role:          string(h.supportRole(ctx, user.ProfileID)),
		AvatarURL:     user.AvatarURL,
		AvatarLink:    user.AvatarURL,
		CreatedAt:     user.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (h *Handler) supportRole(ctx context.Context, profileID int64) model.SupportRole {
	if h.roles == nil {
		return model.SupportRoleUser
	}
	role, err := h.roles.GetProfileRole(ctx, profileID)
	if err != nil {
		return model.SupportRoleUser
	}
	return role
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrLoginAlreadyExists):
		utils.WriteJSON(w, http.StatusConflict, errorResponse{Error: "login already exists"})
	case errors.Is(err, usecase.ErrInvalidCredentials), errors.Is(err, usecase.ErrSessionNotFound):
		utils.WriteJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
	case errors.Is(err, usecase.ErrInvalidInput), errors.Is(err, usecase.ErrInvalidBirthday), errors.Is(err, usecase.ErrTooYoung):
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	default:
		utils.WriteJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}
