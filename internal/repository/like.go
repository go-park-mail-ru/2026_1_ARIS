package repository

import (
	"errors"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
)

type inmemoryLikeRepo struct {
	mu    sync.RWMutex
	likes map[uuid.UUID]models.Like
}

type LikeRepo interface {
	Get(likeID uuid.UUID) (models.Like, error)
	Save(like models.Like) error
}

func NewLikeRepo() LikeRepo {
	repo := inmemoryLikeRepo{}
	repo.likes = make(map[uuid.UUID]models.Like)
	return &repo
}

func (r *inmemoryLikeRepo) Get(likeID uuid.UUID) (models.Like, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	like, ok := r.likes[likeID]
	if !ok {
		return models.Like{}, errors.New("Like not found")
	}
	return like, nil
}

func (r *inmemoryLikeRepo) Save(like models.Like) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.likes[like.ID]
	if !ok {
		r.likes[like.ID] = like
	}
	return nil
}
