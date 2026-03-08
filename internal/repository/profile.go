package repository

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
)

type ProfileRepo interface {
	GetProfileByID(profileID uuid.UUID) (models.Profile, error)
	Save(ctx context.Context, profile models.Profile) error
}

type inmemoryProfileRepo struct {
	Profiles map[uuid.UUID]models.Profile
}

func NewProfileRepo() ProfileRepo {
	repo := inmemoryProfileRepo{}
	repo.Profiles = make(map[uuid.UUID]models.Profile)
	return &repo
}

func (r *inmemoryProfileRepo) GetProfileByID(profileID uuid.UUID) (models.Profile, error) {
	profile, ok := r.Profiles[profileID]
	if !ok {
		return models.Profile{}, errors.New("Profile not found")
	}
	return profile, nil
}

func (r *inmemoryProfileRepo) Save(ctx context.Context, profile models.Profile) error {
	r.Profiles[profile.ID] = profile
	return nil
}
