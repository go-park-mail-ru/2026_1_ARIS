package profile

import (
	"context"
	"errors"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type ProfileRepo interface {
	Get(ctx context.Context, profileID int64) (*models.Profile, error)
	//GetProfileByUsername(username string) (*models.Profile, error)
	Save(ctx context.Context, profile models.Profile) (int64, error)
	GetAll(ctx context.Context) ([]models.Profile, error)
}

type inmemoryProfileRepo struct {
	mu       sync.RWMutex
	Profiles map[int64]models.Profile
}

func NewProfileRepo() ProfileRepo {
	repo := inmemoryProfileRepo{}
	repo.Profiles = make(map[int64]models.Profile)
	return &repo
}

func (r *inmemoryProfileRepo) Get(ctx context.Context, profileID int64) (*models.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profile, ok := r.Profiles[profileID]
	if !ok {
		return nil, errors.New("Profile not found")
	}
	return &profile, nil
}

func (r *inmemoryProfileRepo) Save(ctx context.Context, profile models.Profile) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Profiles[profile.ID] = profile
	return profile.ID, nil
}

// func (r *inmemoryProfileRepo) GetProfileByUsername(username string) (*models.Profile, error) {
// 	r.mu.RLock()
// 	defer r.mu.RUnlock()
// 	for _, p := range r.Profiles {
// 		if p.Username == username {
// 			return &p, nil
// 		}
// 	}
// 	return nil, errors.New("Profile not found")
// }

func (r *inmemoryProfileRepo) GetAll(ctx context.Context) ([]models.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profiles := make([]models.Profile, 0, len(r.Profiles))
	for _, p := range r.Profiles {
		profiles = append(profiles, p)
	}

	return profiles, nil
}
