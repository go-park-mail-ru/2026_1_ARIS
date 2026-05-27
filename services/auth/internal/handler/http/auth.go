package http

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/model"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
)

const (
	sessionCookieName    = "session_id"
	vkidStateCookieName  = "vkid_oauth_state"
	defaultVKIDAuthorize = "https://id.vk.com/authorize"
)

type Handler struct {
	authUsecase  *usecase.Service
	cookieSecure bool
	roles        roleProvider
	oauthStates  repository.OAuthStateRepo
	vkid         VKIDConfig
}

type VKIDConfig struct {
	ClientID            string
	AuthorizeURL        string
	RedirectURI         string
	Scope               string
	FrontendSuccessPath string
	FrontendErrorPath   string
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

func (h *Handler) ConfigureVKID(config VKIDConfig) {
	h.vkid = config
}

func (h *Handler) ConfigureOAuthStates(repo repository.OAuthStateRepo) {
	h.oauthStates = repo
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/register/step-one", h.RegisterStepOne)
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Get("/vkid/login", h.VKIDLogin)
	r.Get("/vkid/callback", h.VKIDCallback)
	r.Post("/password", h.ChangePassword)
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

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		utils.WriteJSON(w, http.StatusUnauthorized, errorResponse{Error: "unauthorized"})
		return
	}

	var req changePasswordRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}

	if err := h.authUsecase.ChangePassword(r.Context(), cookie.Value, usecase.ChangePasswordInput{
		OldPassword:  req.OldPassword,
		NewPassword1: req.NewPassword1,
		NewPassword2: req.NewPassword2,
	}); err != nil {
		writeServiceError(w, err)
		return
	}

	h.clearSessionCookie(w)
	utils.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) VKIDLogin(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(h.vkid.ClientID) == "" || strings.TrimSpace(h.vkid.RedirectURI) == "" {
		utils.WriteJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "vkid oauth is not configured"})
		return
	}

	state, err := randomURLToken(32)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	codeVerifier, err := randomURLToken(64)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	oauthState := repository.OAuthState{
		State:        state,
		CodeVerifier: codeVerifier,
		RedirectURI:  h.vkid.RedirectURI,
		ReturnTo:     sanitizeReturnTo(r.URL.Query().Get("returnTo"), h.vkid.FrontendSuccessPath),
	}
	stateTTL := 10 * time.Minute
	if h.oauthStates != nil {
		if err := h.oauthStates.Save(r.Context(), oauthState, stateTTL); err != nil {
			writeServiceError(w, err)
			return
		}
	}
	h.setVKIDStateCookie(w, oauthState, stateTTL)

	authURL := strings.TrimSpace(h.vkid.AuthorizeURL)
	if authURL == "" {
		authURL = defaultVKIDAuthorize
	}
	scope := strings.TrimSpace(h.vkid.Scope)
	if scope == "" {
		scope = "email"
	}

	authURL, query := splitURLQuery(authURL)
	query.Set("response_type", "code")
	query.Set("client_id", h.vkid.ClientID)
	query.Set("redirect_uri", h.vkid.RedirectURI)
	query.Set("scope", scope)
	query.Set("state", state)
	query.Set("code_challenge", codeChallenge(codeVerifier))
	query.Set("code_challenge_method", "S256")
	http.Redirect(w, r, authURL+"?"+query.Encode(), http.StatusFound)
}

func (h *Handler) VKIDCallback(w http.ResponseWriter, r *http.Request) {
	oauthState, ok := h.resolveVKIDState(r)
	h.clearVKIDStateCookie(w)
	if !ok {
		h.redirectOAuthError(w, r, "invalid_state")
		return
	}
	if r.URL.Query().Get("error") != "" {
		h.redirectOAuthError(w, r, r.URL.Query().Get("error"))
		return
	}

	code := r.URL.Query().Get("code")
	if strings.TrimSpace(code) == "" {
		h.redirectOAuthError(w, r, "missing_code")
		return
	}

	result, err := h.authUsecase.LoginWithVKID(r.Context(), usecase.VKIDCallbackInput{
		Code:         code,
		DeviceID:     r.URL.Query().Get("device_id"),
		CodeVerifier: oauthState.CodeVerifier,
		RedirectURI:  oauthState.RedirectURI,
		State:        oauthState.State,
	})
	if err != nil {
		h.redirectOAuthError(w, r, "provider_error")
		return
	}

	h.setSessionCookie(w, result.Session.SessionID, result.Session.ExpiredAt)
	http.Redirect(w, r, sanitizeReturnTo(oauthState.ReturnTo, h.vkid.FrontendSuccessPath), http.StatusFound)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		_ = h.authUsecase.Logout(r.Context(), cookie.Value)
	}

	h.clearSessionCookie(w)
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

func (h *Handler) setVKIDStateCookie(w http.ResponseWriter, state repository.OAuthState, ttl time.Duration) {
	payload, _ := utils.MarshalJSON(state)
	http.SetCookie(w, &http.Cookie{
		Name:     vkidStateCookieName,
		Value:    base64.RawURLEncoding.EncodeToString(payload),
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.cookieSecure,
		Path:     "/",
	})
}

func (h *Handler) resolveVKIDState(r *http.Request) (repository.OAuthState, bool) {
	queryState := strings.TrimSpace(r.URL.Query().Get("state"))
	if queryState == "" {
		return repository.OAuthState{}, false
	}

	if h.oauthStates != nil {
		state, err := h.oauthStates.Pop(r.Context(), queryState)
		if err == nil && state != nil && state.State == queryState {
			return *state, true
		}
	}

	cookieState, ok := h.readVKIDStateCookie(r)
	if !ok || cookieState.State == "" || cookieState.State != queryState {
		return repository.OAuthState{}, false
	}
	return cookieState, true
}

func (h *Handler) readVKIDStateCookie(r *http.Request) (repository.OAuthState, bool) {
	cookie, err := r.Cookie(vkidStateCookieName)
	if err != nil {
		return repository.OAuthState{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return repository.OAuthState{}, false
	}
	var state repository.OAuthState
	if err := utils.UnmarshalJSON(payload, &state); err != nil {
		return repository.OAuthState{}, false
	}
	return state, true
}

func (h *Handler) clearVKIDStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     vkidStateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   h.cookieSecure,
	})
}

func (h *Handler) redirectOAuthError(w http.ResponseWriter, r *http.Request, reason string) {
	target := sanitizeReturnTo(h.vkid.FrontendErrorPath, "/login")
	target, query := splitURLQuery(target)
	query.Set("oauth", "vkid")
	query.Set("error", reason)
	http.Redirect(w, r, target+"?"+query.Encode(), http.StatusFound)
}

func randomURLToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func codeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func sanitizeReturnTo(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = strings.TrimSpace(fallback)
	}
	if value == "" {
		return "/"
	}
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.Contains(value, "://") {
		return "/"
	}
	return value
}

func splitURLQuery(rawURL string) (string, url.Values) {
	parts := strings.SplitN(rawURL, "?", 2)
	if len(parts) != 2 {
		return rawURL, url.Values{}
	}
	query, err := url.ParseQuery(parts[1])
	if err != nil {
		return parts[0], url.Values{}
	}
	return parts[0], query
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

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   h.cookieSecure,
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
	case errors.Is(err, usecase.ErrInvalidInput), errors.Is(err, usecase.ErrInvalidBirthday), errors.Is(err, usecase.ErrTooYoung),
		errors.Is(err, usecase.ErrPasswordMismatch), errors.Is(err, usecase.ErrPasswordReuse), errors.Is(err, usecase.ErrWeakPassword):
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, usecase.ErrOAuthUnavailable):
		utils.WriteJSON(w, http.StatusServiceUnavailable, errorResponse{Error: err.Error()})
	case errors.Is(err, usecase.ErrOAuthProvider):
		utils.WriteJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
	default:
		utils.WriteJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}
