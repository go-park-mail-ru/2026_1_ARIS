package http

import (
	"errors"
	"html"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/search/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
	"go.uber.org/zap"
)

type Handler struct {
	search *usecase.Service
	log    *zap.Logger
}

func New(search *usecase.Service, log *zap.Logger) *Handler {
	return &Handler{search: search, log: log}
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
		h.log.Error("search failed", zap.Error(err))
		writeServiceError(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, mapResult(result))
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
	case errors.Is(err, usecase.ErrInvalidInput):
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	default:
		utils.WriteJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func mapResult(result *usecase.Result) response {
	resp := response{
		Users:       make([]userResult, 0, len(result.Users)),
		Communities: make([]communityResult, 0, len(result.Communities)),
		Posts:       make([]postResult, 0, len(result.Posts)),
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
			Type:         html.EscapeString(community.Type),
			AvatarID:     community.AvatarID,
			AvatarURL:    community.AvatarURL,
			CoverMediaID: community.CoverMediaID,
			CoverURL:     community.CoverURL,
		})
	}

	for _, post := range result.Posts {
		resp.Posts = append(resp.Posts, postResult{
			ID:              post.ID,
			Text:            html.EscapeString(post.Text),
			AuthorID:        post.AuthorID,
			AuthorProfileID: post.AuthorProfileID,
			AuthorUsername:  html.EscapeString(post.AuthorUsername),
			AuthorFirstName: html.EscapeString(post.AuthorFirstName),
			AuthorLastName:  html.EscapeString(post.AuthorLastName),
			AuthorAvatarID:  post.AuthorAvatarID,
			AuthorAvatarURL: post.AuthorAvatarURL,
			CommunityID:     post.CommunityID,
			CreatedAt:       post.CreatedAt,
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
