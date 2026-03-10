package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"

	"github.com/google/uuid"
)

type UserProfileRepo interface {
	GetUserProfileByID(userProfileID uuid.UUID) (*models.UserProfile, error)
	GetUserProfileByProfileID(profileID uuid.UUID) (*models.UserProfile, error)
	GetUserProfileByUserProfileID(userProfileID uuid.UUID) (*models.UserProfile, error)
	Save(ctx context.Context, userProfile models.UserProfile) error
}

type inmemoryUserProfileRepo struct {
	mu           sync.RWMutex
	userProfiles map[uuid.UUID]models.UserProfile
}

func NewUserProfileRepo() UserProfileRepo {
	repo := inmemoryUserProfileRepo{}
	repo.userProfiles = make(map[uuid.UUID]models.UserProfile)
	return &repo
}

func (r *inmemoryUserProfileRepo) GetUserProfileByID(userProfileID uuid.UUID) (*models.UserProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userProfile, ok := r.userProfiles[userProfileID]
	if !ok {
		return nil, errors.New("UserProfile not found")
	}

	return &userProfile, nil
}

func (r *inmemoryUserProfileRepo) Save(tx context.Context, userProfile models.UserProfile) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.userProfiles[userProfile.ProfileID] = userProfile
	return nil
}

func (r *inmemoryUserProfileRepo) GetUserProfileByProfileID(profileID uuid.UUID) (*models.UserProfile, error) {
	for _, p := range r.userProfiles {
		if p.ProfileID == profileID {
			return &p, nil
		}
	}
	return nil, errors.New("UserProfile not found")
}

func (r *inmemoryUserProfileRepo) GetUserProfileByUserProfileID(userProfileID uuid.UUID) (*models.UserProfile, error) {
	for _, p := range r.userProfiles {
		if p.ID == userProfileID {
			return &p, nil
		}
	}
	return nil, errors.New("UserProfile not found")
}
