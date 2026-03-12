package repository

import (
	"context"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
)

type RepostRepo interface {
	Save(ctx context.Context, repost models.Repost) error
	GetRepostCount(ctx context.Context, postID uuid.UUID) int
}

type inmemoryRepostRepo struct {
	reposts map[uuid.UUID]models.Repost
	mu      sync.RWMutex
}

func NewRepostRepo() RepostRepo {
	return &inmemoryRepostRepo{
		reposts: make(map[uuid.UUID]models.Repost),
	}
}

func (r *inmemoryRepostRepo) Save(ctx context.Context, repost models.Repost) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.reposts[repost.ID]
	if !ok {
		r.reposts[repost.ID] = repost
	}

	return nil
}

func (r *inmemoryRepostRepo) GetRepostCount(ctx context.Context, postID uuid.UUID) int {
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
