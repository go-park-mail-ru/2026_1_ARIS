package http

import (
	"encoding/json"
	"errors"
	"html"
	"math/rand"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/post/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
)

type Handler struct {
	post *service.Service
}

func New(post *service.Service) *Handler {
	return &Handler{post: post}
}

func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Get("/public/feed", h.GetPublicFeed)
	r.Get("/public/popular-posts", h.GetPublicPopularPosts)

	r.Group(func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		r.Get("/feed", h.GetFeed)
		r.Get("/posts/popular", h.GetPopularPosts)
	})

	r.Route("/post", func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		r.Get("/me", h.GetMyPosts)
		r.Get("/profile/{profileID}", h.GetProfilePosts)
		r.Get("/community/{communityID}", h.GetCommunityPosts)
		r.Get("/community/{communityID}/official", h.GetCommunityOfficialPosts)
		r.Post("/upload", h.CreatePost)
		r.Delete("/{id}", h.DeletePost)
		r.Get("/{id}", h.GetPost)
		r.Patch("/{id}", h.UpdatePost)
		r.Post("/{id}/likes", h.LikePost)
		r.Delete("/{id}/likes", h.UnlikePost)
	})
}

func (h *Handler) GetCommunityPosts(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}

	communityID, ok := parseID(w, chi.URLParam(r, "communityID"))
	if !ok {
		return
	}

	posts, err := h.post.GetCommunityPosts(r.Context(), communityID, userAccountID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapPostList(posts))
}

func (h *Handler) GetCommunityOfficialPosts(w http.ResponseWriter, r *http.Request) {
	communityID, ok := parseID(w, chi.URLParam(r, "communityID"))
	if !ok {
		return
	}

	posts, err := h.post.GetCommunityOfficialPosts(r.Context(), communityID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapPostList(posts))
}

func (h *Handler) GetFeed(w http.ResponseWriter, r *http.Request) {
	limit, ok := parseLimit(w, r)
	if !ok {
		return
	}

	feed, err := h.post.GetFeed(r.Context(), r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapFeed(feed))
}

func (h *Handler) GetPublicFeed(w http.ResponseWriter, r *http.Request) {
	limit, ok := parseLimit(w, r)
	if !ok {
		return
	}

	feed, err := h.post.GetPublicFeed(r.Context(), r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapFeed(feed))
}

func (h *Handler) GetPopularPosts(w http.ResponseWriter, r *http.Request) {
	all := popularTitles()
	rand.Shuffle(len(all), func(i, j int) {
		all[i], all[j] = all[j], all[i]
	})
	if len(all) > 3 {
		all = all[:3]
	}
	writeJSON(w, http.StatusOK, popularPostsResponse{Items: all})
}

func (h *Handler) GetPublicPopularPosts(w http.ResponseWriter, r *http.Request) {
	all := popularTitles()
	if len(all) > 3 {
		all = all[:3]
	}
	writeJSON(w, http.StatusOK, popularPostsResponse{Items: all})
}

func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}

	var req PostCreationRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}

	created, err := h.post.CreatePost(r.Context(), userAccountID, createInput(req))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, mapPostDetails(created))
}

func (h *Handler) GetMyPosts(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}

	posts, err := h.post.GetMyPosts(r.Context(), userAccountID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapPostList(posts))
}

func (h *Handler) GetProfilePosts(w http.ResponseWriter, r *http.Request) {
	profileID, ok := parseID(w, chi.URLParam(r, "profileID"))
	if !ok {
		return
	}

	posts, err := h.post.GetProfilePosts(r.Context(), profileID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapPostList(posts))
}

func (h *Handler) DeletePost(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	postID, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	if err := h.post.DeletePost(r.Context(), userAccountID, postID); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	postID, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}

	post, err := h.post.GetPostForViewer(r.Context(), postID, userAccountID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapPostDetails(post))
}

func (h *Handler) LikePost(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	postID, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	post, err := h.post.LikePost(r.Context(), userAccountID, postID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapPostDetails(post))
}

func (h *Handler) UnlikePost(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	postID, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	post, err := h.post.UnlikePost(r.Context(), userAccountID, postID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapPostDetails(post))
}

func (h *Handler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	postID, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}

	var req PostCreationRequest
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, xerrors.InvalidRequestBody, http.StatusBadRequest)
		return
	}

	updated, err := h.post.UpdatePost(r.Context(), userAccountID, postID, createInput(req))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapPostDetails(updated))
}

func userIDFromContext(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok || userAccountID <= 0 {
		utils.WriteError(w, xerrors.InvalidCtxUserAccountValue, http.StatusUnauthorized)
		return 0, false
	}
	return userAccountID, true
}

func parseID(w http.ResponseWriter, value string) (int64, bool) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		utils.WriteError(w, xerrors.InvalidRequest, http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func parseLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Cant parse limit"})
			return 0, false
		}
		limit = parsed
	}
	return limit, true
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput), errors.Is(err, service.ErrPostContentRequired):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrPostNotFound), errors.Is(err, service.ErrProfileNotFound), errors.Is(err, service.ErrCommunityNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrForbidden):
		writeJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: xerrors.InternalServerErrorStr})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func createInput(req PostCreationRequest) service.CreateInput {
	var media []dto.MediaRequestData
	if req.Media != nil {
		media = *req.Media
	}
	return service.CreateInput{Text: req.Text, Media: media, AuthorProfileID: req.AuthorProfileID, CommunityID: req.CommunityID}
}

func mapPostDetails(post *service.PostDetails) PostCreationResponse {
	resp := PostCreationResponse{
		ID:          post.ID,
		ProfileID:   post.Author.ID,
		CommunityID: post.CommunityID,
		Text:        escapeTextPtr(post.Text),
		Author:      mapPostAuthor(post.Author),
		Likes:       post.Likes,
		IsLiked:     post.IsLiked,
	}

	for _, media := range post.Media {
		resp.Media = append(resp.Media, dto.MediaRequestData{MediaID: media.ID, MediaURL: media.URL})
	}

	return resp
}

func mapPostList(posts []service.PostDetails) []PostListItemResponse {
	result := make([]PostListItemResponse, 0, len(posts))
	for _, post := range posts {
		item := PostListItemResponse{
			ID:          post.ID,
			ProfileID:   post.AuthorID,
			CommunityID: post.CommunityID,
			Text:        "",
			Author:      mapPostAuthor(post.Author),
			CreatedAt:   post.CreatedAt,
			Likes:       post.Likes,
			IsLiked:     post.IsLiked,
		}
		if post.Text != nil {
			item.Text = html.EscapeString(*post.Text)
		}
		if !post.UpdatedAt.IsZero() {
			updatedAt := post.UpdatedAt
			item.UpdatedAt = &updatedAt
		}
		for _, media := range post.Media {
			item.Media = append(item.Media, dto.MediaRequestData{MediaID: media.ID, MediaURL: media.URL})
		}
		result = append(result, item)
	}
	return result
}

func mapPostAuthor(author service.Author) postAuthorDTO {
	return postAuthorDTO{
		ProfileID:     author.ID,
		FirstName:     html.EscapeString(author.FirstName),
		LastName:      html.EscapeString(author.LastName),
		Username:      html.EscapeString(author.Username),
		UserAccountID: author.UserAccountID,
		AvatarURL:     author.AvatarURL,
	}
}

func mapFeed(feed service.FeedResult) FeedResponse {
	posts := make([]postFeedDTO, 0, len(feed.Posts))
	for _, post := range feed.Posts {
		medias := make([]mediaFeedDTO, 0, len(post.Medias))
		for _, media := range post.Medias {
			medias = append(medias, mediaFeedDTO{Id: media.UID, MimeType: media.MimeType, Link: media.URL})
		}
		posts = append(posts, postFeedDTO{
			Id:        post.ID, //.String(),
			Text:      post.Text,
			Author:    authorFeedDTO{Id: strconv.FormatInt(post.Author.ID, 10), FirstName: post.Author.FirstName, LastName: post.Author.LastName, Username: post.Author.Username, AvatarLink: derefString(post.Author.AvatarURL)},
			CreatedAt: post.CreatedAt,
			Likes:     post.Likes,
			Comments:  post.Comments,
			Reposts:   post.Reposts,
			Medias:    medias,
		})
	}
	return FeedResponse{Items: posts, NextCursor: feed.Cursor, HasMore: feed.HasMore}
}

func escapeTextPtr(value *string) *string {
	if value == nil {
		return nil
	}
	escaped := html.EscapeString(*value)
	return &escaped
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func popularTitles() []popularPostDTO {
	return []popularPostDTO{
		{Title: "Как научиться подтягиваться 20 раз?"},
		{Title: "Почему Rust заменяет C++"},
		{Title: "Лучшие книги по машинному обучению"},
		{Title: "Как устроены рекомендательные алгоритмы"},
		{Title: "Стоит ли изучать Go в 2026 году"},
	}
}
