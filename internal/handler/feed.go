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
	items      *[]postFeedDTO
	nextCursor string
	hasNext    bool
}

type postFeedDTO struct {
	id        uuid.UUID
	text      string
	author    authorFeedDTO
	createdAt time.Time
	likes     int
	comments  int
	medias    *[]mediaFeedDTO
	documents *[]documentFeedDTO
}

type authorFeedDTO struct {
	id         uuid.UUID
	username   string
	avatarLink string
}

type mediaFeedDTO struct {
	id        uuid.UUID
	mimeType  string
	link      string
	thumbnail string
}

type documentFeedDTO struct {
	id   uuid.UUID
	link string
	name string
	size int
}

type FeedHandler struct {
	service service.FeedServide
}

func NewHandler(service service.FeedServide) *FeedHandler {
	return &FeedHandler{service: service}
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

	feedService := h.service

	result, err := feedService.GetFeed(r.Context(), rawCursor, limit)
	if err != nil {
		fmt.Println("Feed error", err)
		return
	}

	fmt.Println("Result Feed: ")
	for i, p := range result.Posts {
		fmt.Println(i, p)
	}

	//var posts []postFeedDTO

	response := feedResponse{

		nextCursor: result.Cursor,
		hasNext:    result.HasMore,
	}

	json.NewEncoder(w).Encode(response)
}
