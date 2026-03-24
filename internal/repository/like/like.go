package like

import (
	"context"
	"errors"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type inmemoryLikeRepo struct {
	mu    sync.RWMutex
	likes map[int64]models.Like
}

type LikeRepo interface {
	Get(ctx context.Context, likeID int64) (*models.Like, error)
	Save(ctx context.Context, like models.Like) (int64, error)
	GetLikeCountOnPost(ctx context.Context, postID int64) int
}

func NewLikeRepo() LikeRepo {
	repo := inmemoryLikeRepo{}
	repo.likes = make(map[int64]models.Like)
	return &repo
}

func (r *inmemoryLikeRepo) Get(ctx context.Context, likeID int64) (*models.Like, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	like, ok := r.likes[likeID]
	if !ok {
		return nil, errors.New("Like not found")
	}
	return &like, nil
}

func (r *inmemoryLikeRepo) Save(ctx context.Context, like models.Like) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.likes[like.ID]
	if !ok {
		r.likes[like.ID] = like
	}
	return like.ID, nil
}

func (r *inmemoryLikeRepo) GetLikeCountOnPost(ctx context.Context, postID int64) int {
	var counter int

	for _, l := range r.likes {
		if l.PostID != nil && *l.PostID == postID {
			counter++
		}
	}
	return counter
}
