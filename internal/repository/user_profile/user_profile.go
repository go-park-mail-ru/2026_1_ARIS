package userprofile

import (
	"context"
	"errors"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type UserProfileRepo interface {
	Save(ctx context.Context, userProfile models.UserProfile) (int64, error)
	Get(ctx context.Context, userProfileID int64) (*models.UserProfile, error)
	GetByProfileID(ctx context.Context, profileID int64) (*models.UserProfile, error)
	//GetByUserProfileID(ctx context.Context, userProfileID int64) (*models.UserProfile, error)
	GetByUserAccountID(ctx context.Context, userAccountID int64) (*models.UserProfile, error)
}

type inmemoryUserProfileRepo struct {
	mu           sync.RWMutex
	userProfiles map[int64]models.UserProfile
}

func NewUserProfileRepo() UserProfileRepo {
	repo := inmemoryUserProfileRepo{}
	repo.userProfiles = make(map[int64]models.UserProfile)
	return &repo
}

func (r *inmemoryUserProfileRepo) Save(ctx context.Context, userProfile models.UserProfile) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.userProfiles[userProfile.ID] = userProfile
	return userProfile.ID, nil
}

func (r *inmemoryUserProfileRepo) Get(ctx context.Context, userProfileID int64) (*models.UserProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userProfile, ok := r.userProfiles[userProfileID]
	if !ok {
		return nil, errors.New("UserProfile not found")
	}

	return &userProfile, nil
}

func (r *inmemoryUserProfileRepo) GetByProfileID(ctx context.Context, profileID int64) (*models.UserProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.userProfiles {
		if p.ProfileID == profileID {
			return &p, nil
		}
	}
	return nil, errors.New("UserProfile not found")
}

// func (r *inmemoryUserProfileRepo) GetByUserProfileID(ctx context.Context, userProfileID int64) (*models.UserProfile, error) {
// 	r.mu.RLock()
// 	defer r.mu.RUnlock()

// 	for _, p := range r.userProfiles {
// 		if p.ID == userProfileID {
// 			return &p, nil
// 		}
// 	}
// 	return nil, errors.New("UserProfile not found")
// }

func (r *inmemoryUserProfileRepo) GetByUserAccountID(ctx context.Context, userAccountID int64) (*models.UserProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.userProfiles {
		if p.UserAccountID == userAccountID {
			return &p, nil
		}
	}

	return nil, errors.New("UserProfile not found")
}
