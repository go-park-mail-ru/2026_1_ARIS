package repository

import (
	"errors"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"

	"github.com/google/uuid"
)

type UserProfileRepo interface {
	GetUserProfileByID(userProfileID uuid.UUID) (models.UserProfile, error)
	Save(userProfile models.UserProfile) error
}

type inmemoryUserProfileRepo struct {
	userProfiles map[uuid.UUID]models.UserProfile
}

func NewUserProfileRepo() *inmemoryUserProfileRepo {
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

func (r *inmemoryUserProfileRepo) Save(userProfile models.UserProfile) error {
	r.userProfiles[userProfile.ProfileID] = userProfile
	return nil
}
