package service

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
)

type userService struct {
	UserRepo        repository.UserRepo
	ProfileRepo     repository.ProfileRepo
	UserProfileRepo repository.UserProfileRepo
}

type UserService interface {
	CreateRealUserProfile(ctx context.Context, email, phone, password_hash, username, firstName, lastName string, isActive bool, birthdayDate *time.Time, gender models.Gender, avatar *models.Media) models.Profile
	GetUserList(ctx context.Context, offset, limit int) ([]models.User, error)
}

func NewUserProfileService(userRepo repository.UserRepo, profileRepo repository.ProfileRepo, userProfileRepo repository.UserProfileRepo) UserService {
	return &userService{
		UserRepo:        userRepo,
		ProfileRepo:     profileRepo,
		UserProfileRepo: userProfileRepo,
	}
}

func (s *userService) CreateRealUserProfile(ctx context.Context, email, phone, password_hash, username, firstName, lastName string, isActive bool, birthdayDate *time.Time, gender models.Gender, avatar *models.Media) models.Profile {
	user := models.NewUser(email, phone, password_hash)
	profile := models.NewProfile(username, avatar, isActive)
	userProfile := models.NewUserProfile(user, profile, firstName, lastName, birthdayDate, gender)

	s.UserRepo.Save(ctx, user)
	s.ProfileRepo.Save(ctx, profile)
	s.UserProfileRepo.Save(ctx, userProfile)

	return profile
}

func (s *userService) GetUserList(ctx context.Context, offset, limit int) ([]models.User, error) {
	return s.UserRepo.List(ctx, offset, limit)
}
