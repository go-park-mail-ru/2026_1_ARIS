package comment

import (
	"context"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type inmemoryCommentRepo struct {
	mu       sync.RWMutex
	comments map[int64]models.Comment
}

type CommentRepo interface {
	GetCommentCount(ctx context.Context, postID int64) int
	Save(ctx context.Context, comment models.Comment) (int64, error)
}

func NewCommentRepo() CommentRepo {
	repo := inmemoryCommentRepo{}
	repo.comments = make(map[int64]models.Comment)
	return &repo
}

func (r *inmemoryCommentRepo) GetCommentCount(ctx context.Context, postID int64) int {
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

func (r *inmemoryCommentRepo) Save(ctx context.Context, comment models.Comment) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.comments[comment.ID]
	if !ok {
		r.comments[comment.ID] = comment
	}
	return comment.ID, nil
}
