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

type authService struct {
	mu       sync.RWMutex
	userRepo repository.UserRepo
}

type AuthService interface {
	Register(email, password, username, phone string) (*models.User, error)
	Login(email, password string) (*models.User, error)
}

func NewAuthService(userRepo repository.UserRepo) AuthService {
	return &authService{
		userRepo: userRepo,
	}
}

func (s *authService) Register(email, password, username, phone string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	email = strings.ToLower(email)

	if _, err := s.userRepo.GetByEmail(context.Background(), email); err == nil {
		return nil, errors.New("пользователь с таким email уже существует")
	}

	if _, err := s.userRepo.GetByPhone(context.Background(), phone); err == nil {
		return nil, errors.New("пользователь с таким номером телефона уже существует")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("ошибка при обработке пароля")
	}

	user := models.NewUser(
		email,
		phone,
		string(hashedPassword),
	)
	s.userRepo.Save(context.Background(), user)
	return &user, nil
}

func (s *authService) Login(email, password string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	email = strings.ToLower(email)
	user, err := s.userRepo.GetByEmail(context.Background(), email)
	if err != nil {
		return nil, errors.New("недействительные учётные данные")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.New("недействительные учётные данные")
	}

	return &user, nil
}
