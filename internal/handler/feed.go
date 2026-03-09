package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
	"github.com/google/uuid"
)

type feedResponse struct {
	Items      []postFeedDTO
	NextCursor string
	HasNext    bool
}

type postFeedDTO struct {
	Id        uuid.UUID
	Text      string
	Author    authorFeedDTO
	CreatedAt time.Time
	Likes     int
	Comments  int
	Medias    []mediaFeedDTO
}

type authorFeedDTO struct {
	Id         uuid.UUID
	Username   string
	AvatarLink string
}

type mediaFeedDTO struct {
	Id        uuid.UUID
	MimeType  string
	Link      string
	Thumbnail string
}

type FeedHandler struct {
	PostService  service.PostService
	MediaService service.MediaService
}

func NewFeedHandler(postService service.PostService, mediaService service.MediaService) *FeedHandler {
	return &FeedHandler{
		PostService:  postService,
		MediaService: mediaService,
	}
}

func (h *FeedHandler) GetFeed(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		fmt.Println("Required method GET")
		http.Error(w, "Required method GET", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	rawCursor := r.URL.Query().Get("cursor")

	limit := 20

	if l := r.URL.Query().Get("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil {
			fmt.Println(err)
			return
		}
		limit = parsed
	}

	feed, err := h.PostService.GetFeed(r.Context(), rawCursor, limit)
	if err != nil {
		fmt.Println("Feed error", err)
		return
	}

	var posts []postFeedDTO

	// сборка каждого поста в DTO
	for _, post := range feed.Posts {

		postAuthor, err := h.PostService.GetPostAuthor(post.ID)
		if err != nil {
			fmt.Println("Error in hanlder: ", err)
			return
		}

		var authorAvatarLink string

		if postAuthor.AvatarID != nil {
			authorAvatar, err := h.MediaService.GetAvatarByID(*postAuthor.AvatarID)
			if err != nil {
				fmt.Println("Error in feed handler: ", err)
				return
			}
			authorAvatarLink = authorAvatar.Link
		}

		author := authorFeedDTO{
			Id:         postAuthor.ID,
			Username:   postAuthor.Username,
			AvatarLink: authorAvatarLink,
		}

		medias := h.MediaService.GetMediaByPost(post.ID)

		var mediasDTO []mediaFeedDTO

		for _, media := range medias {
			mediasDTO = append(mediasDTO, mediaFeedDTO{
				Id:        media.ID,
				MimeType:  media.MimeType,
				Link:      media.Link,
				Thumbnail: media.Link,
			})
		}

		likeCount := h.PostService.GetLikeCount(r.Context(), post.ID)

		commentCount := h.PostService.GetCommentCount(r.Context(), post.ID)

		posts = append(posts, postFeedDTO{
			Id:        post.ID,
			Text:      *post.Text,
			Author:    author,
			CreatedAt: post.CreatedAt,
			Likes:     likeCount,
			Comments:  commentCount,
			Medias:    mediasDTO,
		})

	}

	response := feedResponse{
		Items:      posts,
		NextCursor: feed.Cursor,
		HasNext:    feed.HasMore,
	}

	json.NewEncoder(w).Encode(response)
}
