package repository

import (
	"fmt"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
)

type inmemoryLIkeToPostRepo struct {
	mu          sync.RWMutex
	likesToPost []models.LikeToPost
}

type LikeToPostRepo interface {
	GetLikeCountOnPost(postID uuid.UUID) int
	Save(likeToPost models.LikeToPost) error
}

func NewLikeToPostRepo() LikeToPostRepo {
	return &inmemoryLIkeToPostRepo{}
}

func (r *inmemoryLIkeToPostRepo) GetLikeCountOnPost(postID uuid.UUID) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	likesCount := 0

	fmt.Println("In GetLikeCountOnPost in Like_To_Post REPO")

	for _, l := range r.likesToPost {
		if l.PostID == postID {
			likesCount++
		}
	}

	fmt.Println("Returned GetLikeCountOnPost in Like_To_Post REPO")

	return likesCount
}

func (r *inmemoryLIkeToPostRepo) Save(likeToPost models.LikeToPost) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.likesToPost = append(r.likesToPost, models.NewLikeToPost(likeToPost.LikeID, likeToPost.PostID))
	return nil
}
