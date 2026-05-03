package feed

import (
	"encoding/json"
	"html"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/media"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/post"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service/user"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type FeedResponse struct {
	Items      []postFeedDTO `json:"posts"`
	NextCursor string        `json:"nextCursor"`
	HasMore    bool          `json:"hasMore"`
}

type postFeedDTO struct {
	Id        int64          `json:"id"`
	Text      string         `json:"text"`
	Author    authorFeedDTO  `json:"author"`
	CreatedAt time.Time      `json:"createdAt"`
	Likes     int            `json:"likes"`
	Comments  int            `json:"comments"`
	Reposts   int            `json:"reposts"`
	Medias    []mediaFeedDTO `json:"medias"`
}

type popularPostDTO struct {
	Title string `json:"title"`
}

type popularPostsResponse struct {
	Items []popularPostDTO `json:"items"`
}

type authorFeedDTO struct {
	Id         string `json:"id"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	Username   string `json:"username"`
	AvatarLink string `json:"avatarLink"`
}

type mediaFeedDTO struct {
	Id       uuid.UUID `json:"id"`
	MimeType string    `json:"mimeType"`
	Link     string    `json:"mediaLink"`
}

type FeedHandler struct {
	PostService        post.PostService
	MediaService       media.MediaService
	UserProfileService user.UserService
}

func NewFeedHandler(postService post.PostService, mediaService media.MediaService, userProfileService user.UserService) *FeedHandler {
	return &FeedHandler{
		PostService:        postService,
		MediaService:       mediaService,
		UserProfileService: userProfileService,
	}
}

// @Description	Getting feed
// @ID			get-feed
// @Summary		Get feed
// @Tags		feed
// @Security	SessionAuth
// @Accept		json
// @Produce		json
// @Success		200		{object}	FeedResponse
// @Failure		400		{object}	dto.CommonErrorResponse
// @Failure		405		{object}	dto.CommonErrorResponse
// @Failure		500		{object}	dto.CommonErrorResponse
// @Param		limit	query		int		false	"number of posts"
// @Param		cursor	query		string	false	"cursor string responded by feed request"
// @Router		/feed [get]
func (h *FeedHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	rawCursor := r.URL.Query().Get("cursor")

	limit := 20

	if l := r.URL.Query().Get("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil {
			log.Warn("invalid_limit_param", zap.String("limit", l), zap.String("path", r.URL.Path), zap.Error(err))
			utils.WriteError(w, "Cant parse limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	feed, err := h.PostService.GetFeed(r.Context(), rawCursor, limit)
	if err != nil {
		log.Error("get_feed_failed",
			zap.String("cursor", rawCursor),
			zap.Int("limit", limit),
			zap.Error(err),
		)
		utils.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var posts []postFeedDTO

	// сборка каждого поста в DTO
	for _, post := range feed.Posts {
		postAuthor, err := h.PostService.GetPostAuthor(r.Context(), post.ID)
		if err != nil {
			log.Error("get_post_author_failed",
				zap.Int64("post_id", post.ID),
				zap.Error(err),
			)
			utils.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var authorAvatarLink string

		if postAuthor.AvatarID != nil {
			authorAvatar, err := h.MediaService.GetAvatarByID(r.Context(), postAuthor.AvatarID)
			if err != nil {
				log.Warn("get_author_avatar_failed",
					zap.Int64("author_id", postAuthor.ID),
					zap.Error(err),
				)
				utils.WriteError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			authorAvatarLink = authorAvatar.Link
		}

		authorUserProfile, err := h.UserProfileService.GetUserProfileByProfileID(r.Context(), postAuthor.ID)
		if err != nil {

			utils.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		authorUserAccount, err := h.UserProfileService.GetUserAccountByUserProfileID(r.Context(), authorUserProfile.ID)
		if err != nil {

			utils.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		author := authorFeedDTO{
			Id:         strconv.FormatInt(postAuthor.ID, 10),
			FirstName:  authorUserProfile.FirstName,
			LastName:   authorUserProfile.LastName,
			Username:   authorUserAccount.Username,
			AvatarLink: authorAvatarLink,
		}

		medias := h.MediaService.GetMediasByPostID(r.Context(), post.ID)
		var mediasDTO []mediaFeedDTO

		for _, media := range medias {
			mediasDTO = append(mediasDTO, mediaFeedDTO{
				Id:       media.Uid,
				MimeType: media.MimeType,
				Link:     media.Link,
			})
		}

		likeCount := h.PostService.GetLikeCount(r.Context(), post.ID)
		commentCount := h.PostService.GetCommentCount(r.Context(), post.ID)
		repostCount := h.PostService.GetRepostCount(r.Context(), post.ID)
		postText := ""
		if post.Text != nil {
			postText = *post.Text
		}

		posts = append(posts, postFeedDTO{
			Id:        post.ID,
			Text:      html.EscapeString(postText),
			Author:    author,
			CreatedAt: post.CreatedAt,
			Likes:     likeCount,
			Comments:  commentCount,
			Reposts:   repostCount,
			Medias:    mediasDTO,
		})
	}

	response := FeedResponse{
		Items:      posts,
		NextCursor: feed.Cursor,
		HasMore:    feed.HasMore,
	}

	log.Info("get_feed_success",
		zap.Int("posts", len(posts)),
		zap.String("cursor", feed.Cursor),
		zap.Bool("has_more", feed.HasMore),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	w.WriteHeader(http.StatusOK)
}

// @Description	Getting popular posts
// @ID			get-popular-users
// @Summary		Get popular posts
// @Tags		feed
// @Security	SessionAuth
// @Produce		json
// @Success		200		{object}	popularPostsResponse
// @Router		/posts/popular [get]
func (h *FeedHandler) GetPopularPosts(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	all := []popularPostDTO{
		{Title: "Как научиться подтягиваться 20 раз?"},
		{Title: "Почему Rust заменяет C++"},
		{Title: "Лучшие книги по машинному обучению"},
		{Title: "Как устроены рекомендательные алгоритмы"},
		{Title: "Стоит ли изучать Go в 2026 году"},
	}

	rand.Shuffle(len(all), func(i, j int) {
		all[i], all[j] = all[j], all[i]
	})

	items := all[:3]

	log.Info("get_popular_posts_success",
		zap.Int("items", len(items)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(popularPostsResponse{
		Items: items,
	})
}

// @Description	Getting public popular posts
// @ID			get-public-popular-users
// @Summary		Get public popular posts
// @Tags		feed
// @Produce		json
// @Success		200		{object}	popularPostsResponse
// @Router		/public/popular-posts [get]
func (h *FeedHandler) GetPublicPopularPosts(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	items := []popularPostDTO{
		{Title: "Как научиться подтягиваться 20 раз?"},
		{Title: "Почему Rust заменяет C++"},
		{Title: "Лучшие книги по машинному обучению"},
	}

	log.Info("get_public_popular_posts_success",
		zap.Int("items", len(items)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(popularPostsResponse{
		Items: items,
	})
}

// @Description	Getting public feed
// @ID			get-public-feed
// @Summary		Get public feed
// @Tags		feed
// @Accept		json
// @Produce		json
// @Success		200		{object}	FeedResponse
// @Failure		400		{object}	dto.CommonErrorResponse
// @Failure		405		{object}	dto.CommonErrorResponse
// @Failure		500		{object}	dto.CommonErrorResponse
// @Param		limit	query		int		false	"number of posts"
// @Param		cursor	query		string	false	"cursor string responded by feed request"
// @Router		/public/feed [get]
func (h *FeedHandler) GetPublicFeed(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context())

	rawCursor := r.URL.Query().Get("cursor")

	limit := 20

	if l := r.URL.Query().Get("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil {
			log.Warn("invalid_limit_param", zap.String("limit", l), zap.String("path", r.URL.Path), zap.Error(err))
			utils.WriteError(w, "Cant parse limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	feed, err := h.PostService.GetPublicFeed(r.Context(), rawCursor, limit)
	if err != nil {
		log.Error("get_public_feed_failed",
			zap.String("cursor", rawCursor),
			zap.Int("limit", limit),
			zap.Error(err),
		)
		utils.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var posts []postFeedDTO

	for _, post := range feed.Posts {
		postAuthor, err := h.PostService.GetPostAuthor(r.Context(), post.ID)
		if err != nil {

			utils.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var authorAvatarLink string

		if postAuthor.AvatarID != nil {
			authorAvatar, err := h.MediaService.GetAvatarByID(r.Context(), postAuthor.AvatarID)
			if err != nil {

				utils.WriteError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			authorAvatarLink = authorAvatar.Link
		}

		authorUserProfile, err := h.UserProfileService.GetUserProfileByProfileID(r.Context(), postAuthor.ID)
		if err != nil {

			utils.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		authorUserAccount, err := h.UserProfileService.GetUserAccountByUserProfileID(r.Context(), authorUserProfile.ID)
		if err != nil {

			utils.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		author := authorFeedDTO{
			Id:         strconv.FormatInt(postAuthor.ID, 10),
			FirstName:  authorUserProfile.FirstName,
			LastName:   authorUserProfile.LastName,
			Username:   authorUserAccount.Username,
			AvatarLink: authorAvatarLink,
		}

		medias := h.MediaService.GetMediasByPostID(r.Context(), post.ID)
		var mediasDTO []mediaFeedDTO

		for _, media := range medias {
			mediasDTO = append(mediasDTO, mediaFeedDTO{
				Id:       media.Uid,
				MimeType: media.MimeType,
				Link:     media.Link,
			})
		}

		likeCount := h.PostService.GetLikeCount(r.Context(), post.ID)
		commentCount := h.PostService.GetCommentCount(r.Context(), post.ID)
		repostCount := h.PostService.GetRepostCount(r.Context(), post.ID)
		postText := ""
		if post.Text != nil {
			postText = *post.Text
		}

		posts = append(posts, postFeedDTO{
			Id:        post.ID,
			Text:      html.EscapeString(postText),
			Author:    author,
			CreatedAt: post.CreatedAt,
			Likes:     likeCount,
			Comments:  commentCount,
			Reposts:   repostCount,
			Medias:    mediasDTO,
		})
	}

	response := FeedResponse{
		Items:      posts,
		NextCursor: "",
		HasMore:    false,
	}

	log.Info("get_public_feed_success",
		zap.Int("posts", len(posts)),
		zap.String("cursor", rawCursor),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	w.WriteHeader(http.StatusOK)
}
