package media

import (
	"context"
	"errors"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type MediaRepo interface {
	Get(ctx context.Context, id int64) (*models.Media, error)
	Save(ctx context.Context, media models.Media) (int64, error)
}

type inmemoryMediaRepo struct {
	mu     sync.RWMutex
	medias map[int64]models.Media
}

func NewMediaRepo() MediaRepo {
	repo := inmemoryMediaRepo{}
	repo.medias = make(map[int64]models.Media)
	return &repo
}

func (r *inmemoryMediaRepo) Get(ctx context.Context, id int64) (*models.Media, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	media, ok := r.medias[id]
	if !ok {
		return nil, errors.New("Media not found")
	}

	return &media, nil
}

func (r *inmemoryMediaRepo) Save(ctx context.Context, media models.Media) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.medias[media.ID] = media
	return media.ID, nil
}
