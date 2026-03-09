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
	//Register(ctx context.Context, email, password, username, phone string) (*models.User, error)
	CreateRealUserProfile(ctx context.Context, password_hash, username, firstName, lastName string, email, phone *string, isActive bool, birthdayDate *time.Time, gender *models.Gender, avatar *models.Media) models.Profile
	Register(ctx context.Context, firstName, lastName, login, password1, password2 string, birthday *time.Time) (models.Profile, error)
	Login(ctx context.Context, email, password string) (*models.User, error)
}

func NewAuthService(userRepo repository.UserRepo, profileRepo repository.ProfileRepo, userProfileRepo repository.UserProfileRepo) AuthService {
	return &authService{
		userRepo:        userRepo,
		profileRepo:     profileRepo,
		userProfileRepo: userProfileRepo,
	}
}

// func (s *authService) Register(ctx context.Context, firstName, lastName, birthday, login, password1, password2 string) (*models.User, error) {
func (s *authService) Register(ctx context.Context, firstName, lastName, login, password1, password2 string, birthday *time.Time) (models.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	//email = strings.ToLower(email)

	if _, err := s.profileRepo.GetProfileByUsername(login); err == nil {
		return models.Profile{}, errors.New("пользователь с таким login уже существует")
	}

	// if _, err := s.userRepo.GetByEmail(context.Background(), email); err == nil {
	// 	return nil, errors.New("пользователь с таким email уже существует")
	// }

	// if _, err := s.userRepo.GetByPhone(context.Background(), phone); err == nil {
	// 	return nil, errors.New("пользователь с таким номером телефона уже существует")
	// }

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password1), bcrypt.DefaultCost)
	if err != nil {
		return models.Profile{}, errors.New("ошибка при обработке пароля")
	}

	profile := s.CreateRealUserProfile(ctx, string(hashedPassword), login, firstName, lastName, nil, nil, true, birthday, nil, nil)

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

	user, err := s.userRepo.GetByID(ctx, userProfile.ID)
	if err != nil {
		return nil, errors.New("недействительные учётные данные")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.New("недействительные учётные данные")
	}

	return &user, nil
}

func (s *authService) CreateRealUserProfile(ctx context.Context, password_hash, username, firstName, lastName string, email, phone *string, isActive bool, birthdayDate *time.Time, gender *models.Gender, avatar *models.Media) models.Profile {
	user := models.NewUser(password_hash, phone, email)
	profile := models.NewProfile(username, avatar, isActive)
	userProfile := models.NewUserProfile(user, profile, firstName, lastName, nil, birthdayDate, gender)

	s.userRepo.Save(ctx, user)
	s.profileRepo.Save(ctx, profile)
	s.userProfileRepo.Save(ctx, userProfile)

	return profile
}
