package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/service"
)

type SidebarHandler struct {
	postService service.PostService
}

type popularPostDTO struct {
	Text string `json:"text"`
	Link string `json:"link"`
}

type popularPostsResponse struct {
	Posts []popularPostDTO `json:"posts"`
}

func NewSidebarHandler(postService service.PostService) *SidebarHandler {
	return &SidebarHandler{
		postService: postService,
	}
}

func (h *SidebarHandler) GetPopularPosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Required method GET", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	count := 3

	if c := r.URL.Query().Get("count"); c != "" {
		parsed, err := strconv.Atoi(c)
		if err != nil {
			fmt.Println("Cant parse 'count'")
		}
		count = parsed
	}

	popularPosts := h.postService.GerPopular(r.Context(), count)

	var posts []popularPostDTO

	for _, post := range popularPosts {
		postDTO := popularPostDTO{Text: *post.Text, Link: "still emply"}
		posts = append(posts, postDTO)
	}

	response := popularPostsResponse{Posts: posts}

	json.NewEncoder(w).Encode(response)
}
