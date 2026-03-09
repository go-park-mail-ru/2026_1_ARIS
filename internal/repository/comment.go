package repository

import (
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
)

type inmemoryCommentRepo struct {
	mu       sync.RWMutex
	comments map[uuid.UUID]models.Comment
}

type CommentRepo interface {
	GetCommentCount(postID uuid.UUID) int
	Save(comment models.Comment) error
}

func NewCommentRepo() CommentRepo {
	repo := inmemoryCommentRepo{}
	repo.comments = make(map[uuid.UUID]models.Comment)
	return &repo
}

func (r *inmemoryCommentRepo) GetCommentCount(postID uuid.UUID) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	commentsCount := 0

	for _, c := range r.comments {
		if c.TargetPostID == postID {
			commentsCount++
		}
	}

	return commentsCount
}

func (r *inmemoryCommentRepo) Save(comment models.Comment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.comments[comment.ID]
	if !ok {
		r.comments[comment.ID] = comment
	}
	return nil
}
