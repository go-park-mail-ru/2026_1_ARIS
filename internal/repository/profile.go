package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
)

type ProfileRepo interface {
	GetProfileByID(profileID uuid.UUID) (*models.Profile, error)
	GetProfileByUsername(username string) (*models.Profile, error)
	Save(ctx context.Context, profile models.Profile) error
	GetAll(ctx context.Context) ([]models.Profile, error)
}

type inmemoryProfileRepo struct {
	mu       sync.RWMutex
	Profiles map[uuid.UUID]models.Profile
}

func NewProfileRepo() ProfileRepo {
	repo := inmemoryProfileRepo{}
	repo.Profiles = make(map[uuid.UUID]models.Profile)
	return &repo
}

func (r *inmemoryProfileRepo) GetProfileByID(profileID uuid.UUID) (*models.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profile, ok := r.Profiles[profileID]
	if !ok {
		return nil, errors.New("Profile not found")
	}
	return &profile, nil
}

func (r *inmemoryProfileRepo) Save(ctx context.Context, profile models.Profile) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Profiles[profile.ID] = profile
	return nil
}

func (r *inmemoryProfileRepo) GetProfileByUsername(username string) (*models.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.Profiles {
		if p.Username == username {
			return &p, nil
		}
	}
	return nil, errors.New("Profile not found")
}

func (r *inmemoryProfileRepo) GetAll(ctx context.Context) ([]models.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profiles := make([]models.Profile, 0, len(r.Profiles))
	for _, p := range r.Profiles {
		profiles = append(profiles, p)
	}

	return profiles, nil
}
