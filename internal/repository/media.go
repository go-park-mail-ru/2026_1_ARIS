package repository

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
)

type MediaRepo interface {
	GetMediaByID(id uuid.UUID) (models.Media, error)
	Save(ctx context.Context, media models.Media) error
}

type inmemoryMediaRepo struct {
	medias map[uuid.UUID]models.Media
}

func NewMediaRepo() *inmemoryMediaRepo {
	repo := inmemoryMediaRepo{}
	repo.medias = make(map[uuid.UUID]models.Media)
	return &repo
}

func (r *inmemoryMediaRepo) GetMediaByID(id uuid.UUID) (models.Media, error) {
	media, ok := r.medias[id]
	if !ok {
		return models.Media{}, errors.New("Media not found")
	}

	return media, nil
}

func (r *inmemoryMediaRepo) Save(ctx context.Context, media models.Media) error {
	r.medias[media.ID] = media
	return nil
}
