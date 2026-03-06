package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
	"github.com/google/uuid"
)

// func randHex(n int) string {
// 	if n < 0 || n > 64 {
// 		n = 32
// 	}
// 	b := make([]byte, n)
// 	rand.Read(b)

// 	return hex.EncodeToString(b)
// }

// type session struct {
// 	token     string
// 	UserID    int
// 	CreatedAt int64
// 	ExpiresAt int64
// }

// type sessionRepo interface {
// 	Create(ctx context.Context, session *session) error
// 	FindByToken(ctx context.Context, token string) (*session, error)
// 	Delete(ctx context.Context, token string) error
// }

// type InmemorySessionRepo struct {
// 	Sessions []session
// }

// func (r *InmemorySessionRepo) Create(ctx context.Context, session *session) error {
// 	r.Sessions = append(r.Sessions, *session)
// 	return nil
// }

// func (r *InmemorySessionRepo) FindByToken(ctx context.Context, token string) (*session, error) {
// 	for _, s := range r.Sessions {
// 		if token == s.token {
// 			return &s, nil
// 		}
// 	}
// 	return nil, errors.New("Can't find session")
// }

// func (r *InmemorySessionRepo) Delete(ctx context.Context, token string) error {
// 	return nil
// }

// var cookie_name = "session_token"

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
	Medias    *[]mediaFeedDTO
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
	FeedService  service.FeedServide
	PostService  service.PostService
	MediaService service.MediaService
}

func NewHandler(feedService service.FeedServide, postService service.PostService, mediaService service.MediaService) *FeedHandler {
	return &FeedHandler{
		FeedService:  feedService,
		PostService:  postService,
		MediaService: mediaService,
	}
}

func (h *FeedHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Feed handler")

	if r.Method != http.MethodGet {
		fmt.Println("Required method GET")
		http.Error(w, "Required method GET", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	/*

		// check cookie

		// обернуть в middleware
		cookie, err := r.Cookie(cookie_name)
		if err != nil {
			fmt.Println(err)
			fmt.Println("Redirecting login...")
			return
		}

		// наверное тоже можно обернуть в middleware
		if cookie.Expires.Before(time.Now()) {
			fmt.Println("Cookie expires")
			// redirect login
			return
		}

	*/

	rawCursor := r.URL.Query().Get("cursor")

	fmt.Println("rawCursor =", rawCursor)

	limit := 20

	if l := r.URL.Query().Get("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil {
			fmt.Println(err)
			return
		}
		limit = parsed
	}

	fmt.Println("limit =", limit)

	postService := h.PostService

	result, err := postService.GetFeed(r.Context(), rawCursor, limit)
	if err != nil {
		fmt.Println("Feed error", err)
		return
	}

	fmt.Println("Result Feed: ")
	for i, p := range result.Posts {
		fmt.Println(i, p)
	}

	var posts []postFeedDTO

	for _, post := range result.Posts {

		fmt.Println("Before h.PostService.GetPostAuthor")
		postAuthor, err := h.PostService.GetPostAuthor(post.ID)

		if err != nil {
			fmt.Println("Error in hanlder: ", err)
			return
		}

		fmt.Println("Post Author = ", postAuthor)

		fmt.Println("postAuthor avatar = ", postAuthor.AvatarID)

		var authorAvatarLink string

		if postAuthor.AvatarID != nil {
			authorAvatar, err := h.MediaService.GetAvatarByID(*postAuthor.AvatarID)

			if err == nil {
				authorAvatarLink = authorAvatar.Link
			}
		}

		author := authorFeedDTO{
			Id:         postAuthor.ID,
			Username:   postAuthor.Username,
			AvatarLink: authorAvatarLink,
		}

		fmt.Println("Author = ", author)

		medias := h.MediaService.GetMediaByPost(post.ID)

		fmt.Println("Post's medias = ", medias)

		var mediasDTO []mediaFeedDTO

		for _, m := range medias {
			mediasDTO = append(mediasDTO, mediaFeedDTO{
				Id:        m.ID,
				MimeType:  m.MimeType,
				Link:      m.Link,
				Thumbnail: m.Link,
			})
		}

		fmt.Println("Medias DTO = ", mediasDTO)

		posts = append(posts, postFeedDTO{
			Id:        post.ID,
			Text:      post.Text,
			Author:    author,
			CreatedAt: post.CreatedAt,
			Likes:     0,
			Comments:  0,
			Medias:    &mediasDTO,
		})

		fmt.Println("Posts DTO = ", posts)
	}

	response := feedResponse{
		Items:      posts,
		NextCursor: result.Cursor,
		HasNext:    result.HasMore,
	}

	fmt.Println("Responce = ", response)

	json.NewEncoder(w).Encode(response)
}
