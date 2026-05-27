package http

import (
	"errors"
	"html"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/analytics"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/usecase"
	"github.com/go-park-mail-ru/2026_1_ARIS/utils"
)

type Handler struct {
	post PostService
}

func New(post PostService) *Handler {
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
		r.Post("/feed/events", h.PostFeedEvents)
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
		r.Get("/{id}/comments", h.GetPostComments)
		r.Post("/{id}/comments", h.CreateComment)
		r.Get("/{id}/comments/replies", h.GetCommentRepliesBatch)
		r.Get("/{id}/comments/{commentID}/replies", h.GetCommentReplies)
		r.Patch("/{id}/comments/{commentID}", h.UpdateComment)
		r.Delete("/{id}/comments/{commentID}", h.DeleteComment)
		r.Post("/{id}/comments/{commentID}/likes", h.LikeComment)
		r.Delete("/{id}/comments/{commentID}/likes", h.UnlikeComment)
	})
}

func (h *Handler) PostFeedEvents(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	var req feedEventsRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	if len(req.Events) > 50 {
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: "too many events"})
		return
	}
	h.post.RecordFeedEvents(r.Context(), userAccountID, mapFeedEvents(req.Events))
	w.WriteHeader(http.StatusNoContent)
}

func mapFeedEvents(items []feedEventItem) []analytics.PostEvent {
	result := make([]analytics.PostEvent, 0, len(items))
	for _, item := range items {
		var t analytics.EventType
		switch item.Type {
		case "view":
			t = analytics.EventPostView
		case "hide":
			t = analytics.EventPostHide
		case "report":
			t = analytics.EventPostReport
		default:
			continue
		}
		result = append(result, analytics.PostEvent{
			PostID:   item.PostID,
			Type:     t,
			DwellMs:  item.DwellMs,
			Position: item.Position,
			Source:   item.Source,
		})
	}
	return result
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
	utils.WriteJSON(w, http.StatusOK, mapPostList(posts))
}

func (h *Handler) GetCommunityOfficialPosts(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	communityID, ok := parseID(w, chi.URLParam(r, "communityID"))
	if !ok {
		return
	}
	posts, err := h.post.GetCommunityOfficialPosts(r.Context(), communityID, userAccountID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapPostList(posts))
}

func (h *Handler) GetFeed(w http.ResponseWriter, r *http.Request) {
	limit, ok := parseLimit(w, r)
	if !ok {
		return
	}
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	mode := r.URL.Query().Get("mode")
	if mode != "by-time" && mode != "for-you" {
		mode = "by-time"
	}
	feed, err := h.post.GetFeed(r.Context(), userAccountID, r.URL.Query().Get("cursor"), mode, limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapFeed(feed))
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
	utils.WriteJSON(w, http.StatusOK, mapFeed(feed))
}

func (h *Handler) GetPopularPosts(w http.ResponseWriter, r *http.Request) {
	all := popularTitles()
	rand.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
	if len(all) > 3 {
		all = all[:3]
	}
	utils.WriteJSON(w, http.StatusOK, popularPostsResponse{Items: all})
}

func (h *Handler) GetPublicPopularPosts(w http.ResponseWriter, r *http.Request) {
	all := popularTitles()
	if len(all) > 3 {
		all = all[:3]
	}
	utils.WriteJSON(w, http.StatusOK, popularPostsResponse{Items: all})
}

func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	var req postCreationRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	created, err := h.post.CreatePost(r.Context(), userAccountID, createInput(req))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, mapPostDetails(created))
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
	utils.WriteJSON(w, http.StatusOK, mapPostList(posts))
}

func (h *Handler) GetProfilePosts(w http.ResponseWriter, r *http.Request) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return
	}
	profileID, ok := parseID(w, chi.URLParam(r, "profileID"))
	if !ok {
		return
	}
	posts, err := h.post.GetProfilePosts(r.Context(), profileID, userAccountID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapPostList(posts))
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
	utils.WriteJSON(w, http.StatusOK, mapPostDetails(post))
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
	utils.WriteJSON(w, http.StatusOK, mapPostDetails(post))
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
	utils.WriteJSON(w, http.StatusOK, mapPostDetails(post))
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
	var req postCreationRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	updated, err := h.post.UpdatePost(r.Context(), userAccountID, postID, createInput(req))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapPostDetails(updated))
}

func (h *Handler) LikeComment(w http.ResponseWriter, r *http.Request) {
	userAccountID, postID, ok := h.userAndPostID(w, r)
	if !ok {
		return
	}
	commentID, ok := parseID(w, chi.URLParam(r, "commentID"))
	if !ok {
		return
	}
	comment, err := h.post.LikeComment(r.Context(), userAccountID, postID, commentID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapComment(*comment))
}

func (h *Handler) UnlikeComment(w http.ResponseWriter, r *http.Request) {
	userAccountID, postID, ok := h.userAndPostID(w, r)
	if !ok {
		return
	}
	commentID, ok := parseID(w, chi.URLParam(r, "commentID"))
	if !ok {
		return
	}
	comment, err := h.post.UnlikeComment(r.Context(), userAccountID, postID, commentID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapComment(*comment))
}

func (h *Handler) GetPostComments(w http.ResponseWriter, r *http.Request) {
	userAccountID, postID, ok := h.userAndPostID(w, r)
	if !ok {
		return
	}
	limit := parseBoundedQueryInt(r, "limit", 50, 1, 100)
	offset := parseBoundedQueryInt(r, "offset", 0, 0, 1<<30)
	comments, err := h.post.GetPostComments(r.Context(), userAccountID, postID, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapComments(comments))
}

func (h *Handler) GetCommentReplies(w http.ResponseWriter, r *http.Request) {
	userAccountID, postID, ok := h.userAndPostID(w, r)
	if !ok {
		return
	}
	commentID, ok := parseID(w, chi.URLParam(r, "commentID"))
	if !ok {
		return
	}
	limit := parseBoundedQueryInt(r, "limit", 50, 1, 100)
	offset := parseBoundedQueryInt(r, "offset", 0, 0, 1<<30)
	comments, err := h.post.GetCommentReplies(r.Context(), userAccountID, postID, commentID, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapComments(comments))
}

func (h *Handler) GetCommentRepliesBatch(w http.ResponseWriter, r *http.Request) {
	userAccountID, postID, ok := h.userAndPostID(w, r)
	if !ok {
		return
	}
	parentIDs, ok := parseParentIDs(w, r.URL.Query().Get("parentIds"))
	if !ok {
		return
	}
	limit := parseBoundedQueryInt(r, "limit", 50, 1, 100)
	offset := parseBoundedQueryInt(r, "offset", 0, 0, 1<<30)
	grouped, err := h.post.GetCommentRepliesBatch(r.Context(), userAccountID, postID, parentIDs, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapCommentsByParent(grouped))
}

func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	userAccountID, postID, ok := h.userAndPostID(w, r)
	if !ok {
		return
	}
	var req commentRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	comment, err := h.post.CreateComment(r.Context(), userAccountID, postID, req.Text, req.ParentCommentID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusCreated, mapComment(*comment))
}

func (h *Handler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	userAccountID, postID, ok := h.userAndPostID(w, r)
	if !ok {
		return
	}
	commentID, ok := parseID(w, chi.URLParam(r, "commentID"))
	if !ok {
		return
	}
	var req commentRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	comment, err := h.post.UpdateComment(r.Context(), userAccountID, postID, commentID, req.Text)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, mapComment(*comment))
}

func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	userAccountID, postID, ok := h.userAndPostID(w, r)
	if !ok {
		return
	}
	commentID, ok := parseID(w, chi.URLParam(r, "commentID"))
	if !ok {
		return
	}
	if err := h.post.DeleteComment(r.Context(), userAccountID, postID, commentID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) userAndPostID(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	userAccountID, ok := userIDFromContext(w, r)
	if !ok {
		return 0, 0, false
	}
	postID, ok := parseID(w, chi.URLParam(r, "id"))
	return userAccountID, postID, ok
}

func userIDFromContext(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userAccountID, ok := r.Context().Value("user_id").(int64)
	if !ok || userAccountID <= 0 {
		utils.WriteJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid user context"})
		return 0, false
	}
	return userAccountID, true
}

func parseID(w http.ResponseWriter, value string) (int64, bool) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request"})
		return 0, false
	}
	return id, true
}

func parseLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: "Cant parse limit"})
			return 0, false
		}
		limit = parsed
	}
	return limit, true
}

func parseBoundedQueryInt(r *http.Request, name string, fallback, minValue, maxValue int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minValue || value > maxValue {
		return fallback
	}
	return value
}

func parseParentIDs(w http.ResponseWriter, raw string) ([]int64, bool) {
	if strings.TrimSpace(raw) == "" {
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request"})
		return nil, false
	}
	parts := strings.Split(raw, ",")
	if len(parts) > 50 {
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request"})
		return nil, false
	}
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id <= 0 {
			utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request"})
			return nil, false
		}
		ids = append(ids, id)
	}
	return ids, true
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidInput), errors.Is(err, usecase.ErrPostContentRequired), errors.Is(err, usecase.ErrCommentsDisabled):
		utils.WriteJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, usecase.ErrPostNotFound), errors.Is(err, usecase.ErrCommentNotFound), errors.Is(err, usecase.ErrProfileNotFound), errors.Is(err, usecase.ErrCommunityNotFound):
		utils.WriteJSON(w, http.StatusNotFound, errorResponse{Error: err.Error()})
	case errors.Is(err, usecase.ErrForbidden):
		utils.WriteJSON(w, http.StatusForbidden, errorResponse{Error: err.Error()})
	default:
		utils.WriteJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func createInput(req postCreationRequest) usecase.CreateInput {
	var media []usecase.MediaRequestData
	if req.Media != nil {
		media = make([]usecase.MediaRequestData, 0, len(*req.Media))
		for _, item := range *req.Media {
			media = append(media, usecase.MediaRequestData{MediaID: item.MediaID, MediaURL: item.MediaURL})
		}
	}
	var files []usecase.MediaRequestData
	if req.Files != nil {
		files = make([]usecase.MediaRequestData, 0, len(*req.Files))
		for _, item := range *req.Files {
			files = append(files, usecase.MediaRequestData{MediaID: item.MediaID, MediaURL: item.MediaURL})
		}
	}
	return usecase.CreateInput{Text: req.Text, Media: media, Files: files, AuthorProfileID: req.AuthorProfileID, CommunityID: req.CommunityID}
}

func mapPostDetails(post *usecase.PostDetails) postCreationResponse {
	resp := postCreationResponse{
		ID:          post.ID,
		ProfileID:   post.Author.ID,
		CommunityID: post.CommunityID,
		Text:        escapeTextPtr(post.Text),
		Author:      mapPostAuthor(post.Author),
		Likes:       post.Likes,
		Comments:    post.Comments,
		IsLiked:     post.IsLiked,
	}
	for _, media := range post.Media {
		resp.Media = append(resp.Media, mediaRequestData{MediaID: media.ID, Name: media.Name, MediaURL: media.URL})
	}
	for _, file := range post.Files {
		resp.Files = append(resp.Files, mediaRequestData{MediaID: file.ID, Name: file.Name, MediaURL: file.URL})
	}
	return resp
}

func mapPostList(posts []usecase.PostDetails) []postListItemResponse {
	result := make([]postListItemResponse, 0, len(posts))
	for _, post := range posts {
		item := postListItemResponse{
			ID:          post.ID,
			ProfileID:   post.AuthorID,
			CommunityID: post.CommunityID,
			Author:      mapPostAuthor(post.Author),
			CreatedAt:   post.CreatedAt,
			Likes:       post.Likes,
			Comments:    post.Comments,
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
			item.Media = append(item.Media, mediaRequestData{MediaID: media.ID, Name: media.Name, MediaURL: media.URL})
		}
		for _, file := range post.Files {
			item.Files = append(item.Files, mediaRequestData{MediaID: file.ID, Name: file.Name, MediaURL: file.URL})
		}
		result = append(result, item)
	}
	return result
}

func mapPostAuthor(author usecase.Author) postAuthorDTO {
	return postAuthorDTO{
		ProfileID:     author.ID,
		FirstName:     html.EscapeString(author.FirstName),
		LastName:      html.EscapeString(author.LastName),
		Username:      html.EscapeString(author.Username),
		UserAccountID: author.UserAccountID,
		AvatarURL:     author.AvatarURL,
	}
}

func mapFeed(feed usecase.FeedResult) feedResponse {
	posts := make([]postFeedDTO, 0, len(feed.Posts))
	for _, post := range feed.Posts {
		medias := make([]mediaFeedDTO, 0, len(post.Medias))
		for _, media := range post.Medias {
			medias = append(medias, mediaFeedDTO{ID: media.UID, Name: media.Name, MimeType: media.MimeType, Link: media.URL})
		}
		files := make([]mediaFeedDTO, 0, len(post.Files))
		for _, file := range post.Files {
			files = append(files, mediaFeedDTO{ID: file.UID, Name: file.Name, MimeType: file.MimeType, Link: file.URL})
		}
		posts = append(posts, postFeedDTO{
			ID:        post.ID,
			Text:      post.Text,
			Author:    authorFeedDTO{ID: strconv.FormatInt(post.Author.ID, 10), FirstName: post.Author.FirstName, LastName: post.Author.LastName, Username: post.Author.Username, AvatarLink: derefString(post.Author.AvatarURL)},
			CreatedAt: post.CreatedAt,
			Likes:     post.Likes,
			IsLiked:   post.IsLiked,
			Comments:  post.Comments,
			Reposts:   post.Reposts,
			Medias:    medias,
			Files:     files,
		})
	}
	return feedResponse{Items: posts, NextCursor: feed.Cursor, HasMore: feed.HasMore}
}

func mapComments(comments []usecase.Comment) []commentResponse {
	result := make([]commentResponse, 0, len(comments))
	for _, comment := range comments {
		result = append(result, mapComment(comment))
	}
	return result
}

func mapCommentsByParent(grouped map[int64][]usecase.Comment) map[string][]commentResponse {
	result := make(map[string][]commentResponse, len(grouped))
	for parentID, comments := range grouped {
		result[strconv.FormatInt(parentID, 10)] = mapComments(comments)
	}
	return result
}

func mapComment(comment usecase.Comment) commentResponse {
	return commentResponse{
		ID:              strconv.FormatInt(comment.ID, 10),
		Uid:             comment.UID.String(),
		Text:            comment.Text,
		PostID:          strconv.FormatInt(comment.PostID, 10),
		ParentCommentID: int64PtrString(comment.ParentCommentID),
		Author:          mapPostAuthor(comment.Author),
		CreatedAt:       comment.CreatedAt,
		UpdatedAt:       comment.UpdatedAt,
		RepliesCount:    comment.RepliesCount,
		Likes:           comment.Likes,
		IsLiked:         comment.IsLiked,
	}
}

func int64PtrString(value *int64) *string {
	if value == nil {
		return nil
	}
	out := strconv.FormatInt(*value, 10)
	return &out
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
