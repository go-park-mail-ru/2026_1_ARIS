package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	mu              sync.RWMutex
	userRepo        repository.UserRepo
	userProfileRepo repository.UserProfileRepo
	profileRepo     repository.ProfileRepo
}

type AuthService interface {
	CreateRealUserProfile(ctx context.Context, password_hash, username, firstName, lastName string, email, phone *string, isActive bool, birthdayDate *time.Time, gender *models.Gender, avatar *models.Media) (*models.Profile, error)
	Register(ctx context.Context, firstName, lastName, login, password1, birthday string) (*models.Profile, error)
	Login(ctx context.Context, email, password string) (*models.User, error)
}

func NewAuthService(userRepo repository.UserRepo, profileRepo repository.ProfileRepo, userProfileRepo repository.UserProfileRepo) AuthService {
	return &authService{
		userRepo:        userRepo,
		profileRepo:     profileRepo,
		userProfileRepo: userProfileRepo,
	}
}

func (s *authService) Register(ctx context.Context, firstName, lastName, login, password1, birthday string) (*models.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.profileRepo.GetProfileByUsername(login); err == nil {
		return nil, errors.New("пользователь с таким login уже существует")
	}

	birthdayDate, err := time.Parse("02/01/2006", birthday)
	if err != nil {
		return nil, errors.New("invalid birthday date")
	}
	if birthdayDate.AddDate(12, 0, 0).After(time.Now()) {
		return nil, errors.New("you are too young, buddy")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password1), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("ошибка при обработке пароля")
	}

	profile, err := s.CreateRealUserProfile(ctx, string(hashedPassword), login, firstName, lastName, nil, nil, true, &birthdayDate, nil, nil)

	return profile, nil
}

func (s *authService) Login(ctx context.Context, login, password string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profile, err := s.profileRepo.GetProfileByUsername(login)
	if err != nil {
		return nil, errors.New("недействительные учётные данные")
	}

	userProfile, err := s.userProfileRepo.GetUserProfileByProfileID(profile.ID)
	if err != nil {
		return nil, errors.New("недействительные учётные данные")
	}

	user, err := s.userRepo.GetByID(ctx, userProfile.UserID)
	if err != nil {
		return nil, errors.New("недействительные учётные данные")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.New("недействительные учётные данные")
	}

	return user, nil
}

func (s *authService) CreateRealUserProfile(ctx context.Context, password_hash, username, firstName, lastName string, email, phone *string, isActive bool, birthdayDate *time.Time, gender *models.Gender, avatar *models.Media) (*models.Profile, error) {
	user := models.NewUser(password_hash, phone, email)
	profile := models.NewProfile(username, avatar, isActive)
	userProfile := models.NewUserProfile(user, profile, firstName, lastName, nil, birthdayDate, gender)

	s.userRepo.Save(ctx, user)
	s.profileRepo.Save(ctx, profile)
	s.userProfileRepo.Save(ctx, userProfile)

	return &profile, nil
}
