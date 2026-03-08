package repository

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"

	"github.com/google/uuid"
)

type UserProfileRepo interface {
	GetUserProfileByID(userProfileID uuid.UUID) (models.UserProfile, error)
	Save(ctx context.Context, userProfile models.UserProfile) error
}

type inmemoryUserProfileRepo struct {
	userProfiles map[uuid.UUID]models.UserProfile
}

func NewUserProfileRepo() UserProfileRepo {
	repo := inmemoryUserProfileRepo{}
	repo.userProfiles = make(map[uuid.UUID]models.UserProfile)
	return &repo
}

func (r *inmemoryUserProfileRepo) GetUserProfileByID(userProfileID uuid.UUID) (models.UserProfile, error) {
	userProfile, ok := r.userProfiles[userProfileID]
	if !ok {
		return models.UserProfile{}, errors.New("UserProfile not found")
	}

	return userProfile, nil
}

func (r *inmemoryUserProfileRepo) Save(tx context.Context, userProfile models.UserProfile) error {
	r.userProfiles[userProfile.ProfileID] = userProfile
	return nil
}
