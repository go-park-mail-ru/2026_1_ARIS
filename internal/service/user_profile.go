package service

//go:generate mockgen -destination=./mocks/user_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/service UserService
import (
	"context"
	"math/rand"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/google/uuid"
)

type userService struct {
	UserRepo        repository.UserRepo
	ProfileRepo     repository.ProfileRepo
	UserProfileRepo repository.UserProfileRepo
}

type UserService interface {
	CreateRealUserProfile(ctx context.Context, email, phone, password_hash, username, firstName, lastName string, isActive bool, birthdayDate *time.Time, gender models.Gender, avatar *models.Media) (*models.Profile, error)
	GetUserList(ctx context.Context, offset, limit int) []models.User
	GetUserProfileByProfile(ctx context.Context, profileID uuid.UUID) (*models.UserProfile, error)
	GetUserProfileByUserProfileID(userProfileID uuid.UUID) (*models.UserProfile, error)
	GetUserProfileByUser(ctx context.Context, userID uuid.UUID) (*models.UserProfile, error)
	GetSuggestedUsers(ctx context.Context, currentUserID uuid.UUID) ([]models.Profile, error)
	GetPublicPopularUsers(ctx context.Context) ([]models.Profile, error)
	GetLatestEvents(ctx context.Context) ([]LatestEvent, error)
}

type LatestEvent struct {
	Profile models.Profile
	Type    int
}

func NewUserProfileService(userRepo repository.UserRepo, profileRepo repository.ProfileRepo, userProfileRepo repository.UserProfileRepo) UserService {
	return &userService{
		UserRepo:        userRepo,
		ProfileRepo:     profileRepo,
		UserProfileRepo: userProfileRepo,
	}
}

func (s *userService) CreateRealUserProfile(ctx context.Context, email, phone, password_hash, username, firstName, lastName string, isActive bool, birthdayDate *time.Time, gender models.Gender, avatar *models.Media) (*models.Profile, error) {
	user := models.NewUser(password_hash, &phone, &email)
	profile := models.NewProfile(username, avatar, isActive)
	userProfile := models.NewUserProfile(user, profile, firstName, lastName, nil, birthdayDate, &gender)

	s.UserRepo.Save(ctx, user)
	s.ProfileRepo.Save(ctx, profile)
	s.UserProfileRepo.Save(ctx, userProfile)

	return &profile, nil
}

func (s *userService) GetUserList(ctx context.Context, offset, limit int) []models.User {
	return s.UserRepo.List(ctx, offset, limit)
}

func (s *userService) GetUserProfileByProfile(ctx context.Context, profileID uuid.UUID) (*models.UserProfile, error) {
	return s.UserProfileRepo.GetUserProfileByProfileID(profileID)
}

func (s *userService) GetUserProfileByUserProfileID(userProfileID uuid.UUID) (*models.UserProfile, error) {
	return s.UserProfileRepo.GetUserProfileByUserProfileID(userProfileID)
}

func (s *userService) GetUserProfileByUser(ctx context.Context, userID uuid.UUID) (*models.UserProfile, error) {
	return s.UserProfileRepo.GetUserProfileByUserID(userID)
}

func (s *userService) GetSuggestedUsers(ctx context.Context, currentUserID uuid.UUID) ([]models.Profile, error) {
	profiles, err := s.ProfileRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	currentUserProfile, err := s.UserProfileRepo.GetUserProfileByUserProfileID(currentUserID)
	if err != nil {
		return nil, err
	}

	currentProfileID := currentUserProfile.ProfileID

	var filtered []models.Profile

	for _, p := range profiles {
		if p.ID == currentProfileID {
			continue
		}

		if p.Username == "KomandaARIS" {
			continue
		}

		filtered = append(filtered, p)
	}

	rand.Shuffle(len(filtered), func(i, j int) {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	})

	if len(filtered) > 4 {
		filtered = filtered[:4]
	}

	return filtered, nil
}

func (s *userService) GetPublicPopularUsers(ctx context.Context) ([]models.Profile, error) {
	allProfiles, err := s.ProfileRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	targetUsernames := []string{
		"SergeyShulginenko",
		"AnnaOparina",
		"IvanKhvostov",
		"RinatBaikov",
	}

	profilesByUsername := make(map[string]models.Profile)
	for _, profile := range allProfiles {
		profilesByUsername[profile.Username] = profile
	}

	result := make([]models.Profile, 0, len(targetUsernames))
	for _, username := range targetUsernames {
		profile, ok := profilesByUsername[username]
		if ok {
			result = append(result, profile)
		}
	}

	return result, nil
}

func (s *userService) GetLatestEvents(ctx context.Context) ([]LatestEvent, error) {
	allProfiles, err := s.ProfileRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	targets := []struct {
		Username string
		Type     int
	}{
		{Username: "SofiaSitnichenko", Type: 1},
		{Username: "DaniilKhasyanov", Type: 2},
		{Username: "KonstantinGalanin", Type: 3},
	}

	profilesByUsername := make(map[string]models.Profile)
	for _, profile := range allProfiles {
		profilesByUsername[profile.Username] = profile
	}

	result := make([]LatestEvent, 0, len(targets))
	for _, target := range targets {
		profile, ok := profilesByUsername[target.Username]
		if ok {
			result = append(result, LatestEvent{
				Profile: profile,
				Type:    target.Type,
			})
		}
	}

	return result, nil
}
