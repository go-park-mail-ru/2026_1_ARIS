package repost

import (
	"context"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type RepostRepo interface {
	Save(ctx context.Context, repost models.Repost) (int64, error)
	GetRepostCount(ctx context.Context, postID int64) int
}

type inmemoryRepostRepo struct {
	reposts map[int64]models.Repost
	mu      sync.RWMutex
}

func NewRepostRepo() RepostRepo {
	return &inmemoryRepostRepo{
		reposts: make(map[int64]models.Repost),
	}
}

func (r *inmemoryRepostRepo) Save(ctx context.Context, repost models.Repost) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.reposts[repost.ID]
	if !ok {
		r.reposts[repost.ID] = repost
	}

	return repost.ID, nil
}

func (r *inmemoryRepostRepo) GetRepostCount(ctx context.Context, postID int64) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0

	for _, repost := range r.reposts {
		if repost.PostID == postID {
			count++
		}
	}

	return count
}
