package http

import (
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/search/service"
)

type Handler struct {
	search *service.Service
}

func New(search *service.Service) *Handler {
	return &Handler{search: search}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/search", h.Search)
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		query = r.URL.Query().Get("query")
	}
	limit := parseLimit(r, "limit", 10)

	result, err := h.search.Search(r.Context(), query, limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapResult(result))
}

func parseLimit(r *http.Request, name string, fallback int) int {
	value := r.URL.Query().Get(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: xerrors.InternalServerErrorStr})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func mapResult(result *service.Result) response {
	resp := response{
		Users:       make([]userResult, 0, len(result.Users)),
		Communities: make([]communityResult, 0, len(result.Communities)),
	}

	for _, user := range result.Users {
		resp.Users = append(resp.Users, userResult{
			ProfileID:     user.ProfileID,
			UserAccountID: user.UserAccountID,
			Username:      html.EscapeString(user.Username),
			FirstName:     html.EscapeString(user.FirstName),
			LastName:      html.EscapeString(user.LastName),
			AvatarID:      user.AvatarID,
			AvatarURL:     user.AvatarURL,
		})
	}

	for _, community := range result.Communities {
		resp.Communities = append(resp.Communities, communityResult{
			ID:           community.ID,
			ProfileID:    community.ProfileID,
			Username:     html.EscapeString(community.Username),
			Title:        html.EscapeString(community.Title),
			Bio:          escapePtr(community.Bio),
			Type:         community.Type,
			AvatarID:     community.AvatarID,
			AvatarURL:    community.AvatarURL,
			CoverMediaID: community.CoverMediaID,
			CoverURL:     community.CoverURL,
		})
	}

	return resp
}

func escapePtr(value *string) *string {
	if value == nil {
		return nil
	}
	escaped := html.EscapeString(*value)
	return &escaped
}
