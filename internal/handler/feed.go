package handlers

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/utils"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/cursor"
	"github.com/google/uuid"
)

type FeedResponse struct {
	Items      []postFeedDTO `json:"posts"`
	NextCursor string        `json:"nextCursor"`
	HasMore    bool          `json:"hasMore"`
}

type postFeedDTO struct {
	Id        uuid.UUID      `json:"id"`
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
	Id         uuid.UUID `json:"id"`
	FirstName  string    `json:"firstName"`
	LastName   string    `json:"lastName"`
	Username   string    `json:"username"`
	AvatarLink string    `json:"avatarLink"`
}

type mediaFeedDTO struct {
	Id       uuid.UUID `json:"id"`
	MimeType string    `json:"mimeType"`
	Link     string    `json:"mediaLink"`
}

type FeedHandler struct {
	PostService        service.PostService
	MediaService       service.MediaService
	UserProfileService service.UserService
}

func NewFeedHandler(postService service.PostService, mediaService service.MediaService, userProfileService service.UserService) *FeedHandler {
	return &FeedHandler{
		PostService:        postService,
		MediaService:       mediaService,
		UserProfileService: userProfileService,
	}
}

// @Description Getting feed
// @ID get-feed
// @Summary Get feed
// @Tags feed
// @Security SessionAuth
// @Accept json
// @Produce json
// @Success 200 {object} FeedResponse
// @Failure 400 {object} CommonResponse
// @Failure 405 {object} CommonResponse
// @Failure 500 {object} CommonResponse
// @Param limit query int false "number of posts"
// @Param cursor query string false "cursor string responded by feed request"
// @Router /feed [get]
func (h *FeedHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, "Required method GET", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	rawCursor := r.URL.Query().Get("cursor")

	limit := 20

	if l := r.URL.Query().Get("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil {
			utils.WriteError(w, "Cant parse limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	feed, err := h.PostService.GetFeed(r.Context(), rawCursor, limit)
	if err != nil {
		utils.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var posts []postFeedDTO

	// сборка каждого поста в DTO
	for _, post := range feed.Posts {

		postAuthor, err := h.PostService.GetPostAuthor(post.ID)
		if err != nil {
			utils.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var authorAvatarLink string

		if postAuthor.AvatarID != nil {
			authorAvatar, err := h.MediaService.GetAvatarByID(*postAuthor.AvatarID)
			if err != nil {
				utils.WriteError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			authorAvatarLink = authorAvatar.Link
		}

		authorProfile, err := h.UserProfileService.GetUserProfileByProfile(r.Context(), postAuthor.ID)

		author := authorFeedDTO{
			Id:         postAuthor.ID,
			FirstName:  authorProfile.FirstName,
			LastName:   authorProfile.LastName,
			Username:   postAuthor.Username,
			AvatarLink: authorAvatarLink,
		}

		medias := h.MediaService.GetMediaByPost(post.ID)

		var mediasDTO []mediaFeedDTO

		for _, media := range medias {
			mediasDTO = append(mediasDTO, mediaFeedDTO{
				Id:       media.ID,
				MimeType: media.MimeType,
				Link:     media.Link,
			})
		}

		likeCount := h.PostService.GetLikeCount(r.Context(), post.ID)

		commentCount := h.PostService.GetCommentCount(r.Context(), post.ID)

		repostCount := h.PostService.GetRepostCount(r.Context(), post.ID)

		posts = append(posts, postFeedDTO{
			Id:        post.ID,
			Text:      *post.Text,
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

	json.NewEncoder(w).Encode(response)
}

func (h *FeedHandler) GetPopularPosts(w http.ResponseWriter, r *http.Request) {

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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(popularPostsResponse{
		Items: items,
	})
}

func (h *FeedHandler) GetPublicPopularPosts(w http.ResponseWriter, r *http.Request) {

	items := []popularPostDTO{
		{Title: "Как научиться подтягиваться 20 раз?"},
		{Title: "Почему Rust заменяет C++"},
		{Title: "Лучшие книги по машинному обучению"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(popularPostsResponse{
		Items: items,
	})
}

func (h *FeedHandler) GetPublicFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, "Required method GET", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	rawCursor := r.URL.Query().Get("cursor")

	limit := 20

	if l := r.URL.Query().Get("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil {
			utils.WriteError(w, "Cant parse limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	params := repository.FeedParams{Limit: limit}

	if rawCursor != "" {
		decoded, err := cursor.Decode(rawCursor)
		if err != nil {
			utils.WriteError(w, err.Error(), http.StatusBadRequest)
			return
		}
		params.Cursor = &decoded
	}

	postsData, err := h.PostService.GetPublicFeed(r.Context(), params)
	if err != nil {
		utils.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var posts []postFeedDTO

	for _, post := range postsData {
		postAuthor, err := h.PostService.GetPostAuthor(post.ID)
		if err != nil {
			utils.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var authorAvatarLink string

		if postAuthor.AvatarID != nil {
			authorAvatar, err := h.MediaService.GetAvatarByID(*postAuthor.AvatarID)
			if err != nil {
				utils.WriteError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			authorAvatarLink = authorAvatar.Link
		}

		authorProfile, err := h.UserProfileService.GetUserProfileByProfile(r.Context(), postAuthor.ID)
		if err != nil {
			utils.WriteError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		author := authorFeedDTO{
			Id:         postAuthor.ID,
			FirstName:  authorProfile.FirstName,
			LastName:   authorProfile.LastName,
			Username:   postAuthor.Username,
			AvatarLink: authorAvatarLink,
		}

		medias := h.MediaService.GetMediaByPost(post.ID)
		var mediasDTO []mediaFeedDTO

		for _, media := range medias {
			mediasDTO = append(mediasDTO, mediaFeedDTO{
				Id:       media.ID,
				MimeType: media.MimeType,
				Link:     media.Link,
			})
		}

		likeCount := h.PostService.GetLikeCount(r.Context(), post.ID)
		commentCount := h.PostService.GetCommentCount(r.Context(), post.ID)
		repostCount := h.PostService.GetRepostCount(r.Context(), post.ID)

		posts = append(posts, postFeedDTO{
			Id:        post.ID,
			Text:      *post.Text,
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

	json.NewEncoder(w).Encode(response)
}
