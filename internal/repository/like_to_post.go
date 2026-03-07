package repository

import (
	"fmt"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
)

type inmemoryLIkeToPostRepo struct {
	likesToPost []models.LikeToPost
}

type LikeToPostRepo interface {
	GetLikeCountOnPost(postID uuid.UUID) int
}

func NewLikeToPostRepo() LikeToPostRepo {
	return &inmemoryLIkeToPostRepo{}
}

func (r *inmemoryLIkeToPostRepo) GetLikeCountOnPost(postID uuid.UUID) int {
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
