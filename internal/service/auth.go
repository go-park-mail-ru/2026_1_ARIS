package service

import (
	"errors"
	"strings"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	mu     sync.RWMutex
	users  map[string]models.User
	nextID models.UserID
}

func NewAuthService() *AuthService {
	return &AuthService{
		users:  make(map[string]models.User),
		nextID: 1,
	}
}

func (s *AuthService) Register(email, password, username, phone string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	email = strings.ToLower(email)

	if _, exists := s.users[email]; exists {
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
	s.users[email] = user
	s.nextID++
	return &user, nil
}

func (s *AuthService) Login(email, password string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	email = strings.ToLower(email)
	user, exists := s.users[email]
	if !exists {
		return nil, errors.New("недействительные учётные данные")
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("недействительные учётные данные")
	}

	return &user, nil
}
