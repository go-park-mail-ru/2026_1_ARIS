package post

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type PostRepo interface {
	Save(ctx context.Context, post models.Post) (int64, error)
	Delete(ctx context.Context, id int64) error

	List(ctx context.Context, offset, limit int) ([]models.Post, error)
	Get(ctx context.Context, id int64) (*models.Post, error)
	GetAll(ctx context.Context) ([]models.Post, error)
}

type inmemoryPostRepo struct {
	mu    sync.RWMutex
	Posts map[int64]models.Post
}

func NewPostRepo() PostRepo {
	repo := inmemoryPostRepo{}
	repo.Posts = make(map[int64]models.Post)
	return &repo
}

func (r *inmemoryPostRepo) Save(ctx context.Context, post models.Post) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.Posts[post.ID]
	if !ok {
		r.Posts[post.ID] = post
	}

	return post.ID, nil
}

func (r *inmemoryPostRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.Posts[id]

	if !ok {
		return nil //errors.New("post not found")
	}

	delete(r.Posts, id)
	return nil
}

func (r *inmemoryPostRepo) List(ctx context.Context, offset, limit int) ([]models.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if offset >= len(r.Posts) {
		return []models.Post{}, nil
	}

	if offset+limit > len(r.Posts) {
		return slices.Collect(maps.Values(r.Posts))[offset:], nil
	}

	return slices.Collect(maps.Values(r.Posts))[offset:offset:limit], nil
}

func (r *inmemoryPostRepo) Get(ctx context.Context, id int64) (*models.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profile, ok := r.Posts[id]
	if !ok {
		return nil, errors.New("Profile not found")
	}

	return &profile, nil
}

func (r *inmemoryPostRepo) GetAll(ctx context.Context) ([]models.Post, error) {
	return slices.Collect(maps.Values(r.Posts)), nil
}
