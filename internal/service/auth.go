package service

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	mu       sync.RWMutex
	userRepo repository.UserRepo
	nextID   models.UserID
}

func NewAuthService(repo repository.UserRepo) *AuthService {
	return &AuthService{
		userRepo: repo,
		nextID:   1,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password, username, phone string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	email = strings.ToLower(email)

	existingUser, _ := s.userRepo.GetByEmail(ctx, email)
	if existingUser.Email != "" {
		return nil, errors.New("пользователь с таким email уже существует")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("ошибка при обработке пароля")
	}

	user := models.NewUser(
		s.nextID,
		username,
		email,
		phone,
		string(hashedPassword),
	)

	s.userRepo.Save(ctx, user)

	s.nextID++
	return &user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*models.User, error) {
	email = strings.ToLower(email)

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, errors.New("недействительные учётные данные")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("недействительные учётные данные")
	}

	return &user, nil
}
