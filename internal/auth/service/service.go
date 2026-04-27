package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/auth/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const SessionTTL = 24 * time.Hour

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidBirthday    = errors.New("invalid birthday")
	ErrTooYoung           = errors.New("user must be at least 12 years old")
	ErrLoginAlreadyExists = errors.New("login already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionNotFound    = errors.New("session not found")
)

type Service struct {
	store repository.Store
	now   func() time.Time
}

func New(store repository.Store) *Service {
	return &Service{store: store, now: time.Now}
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (*AuthResult, error) {
	login := normalizeLogin(in.Login)
	if in.FirstName == "" || in.LastName == "" || login == "" || in.Password == "" {
		return nil, ErrInvalidInput
	}

	if _, err := s.store.Accounts.GetByUsername(ctx, login); err == nil {
		return nil, ErrLoginAlreadyExists
	}

	birthday, err := time.Parse("2006-01-02", in.Birthday)
	if err != nil {
		return nil, ErrInvalidBirthday
	}
	if birthday.AddDate(12, 0, 0).After(s.now()) {
		return nil, ErrTooYoung
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	accountID, err := s.store.Accounts.Save(ctx, *models.NewUserAccount(login, nil, nil, string(hash)))
	if err != nil {
		return nil, err
	}

	profileID, err := s.store.Profiles.Save(ctx, *models.NewProfile(nil))
	if err != nil {
		return nil, err
	}

	userProfile := models.NewUserProfile(accountID, profileID, in.FirstName, in.LastName, nil, birthday, normalizeGender(in.Gender))
	if _, err := s.store.UserProfiles.Save(ctx, *userProfile); err != nil {
		return nil, err
	}

	return s.issueAuthResult(ctx, accountID)
}

func (s *Service) Login(ctx context.Context, in LoginInput) (*AuthResult, error) {
	account, err := s.store.Accounts.GetByUsername(ctx, normalizeLogin(in.Login))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(in.Password)) != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issueAuthResult(ctx, account.ID)
}

func (s *Service) ValidateSession(ctx context.Context, sessionID string) (*models.Session, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, ErrSessionNotFound
	}

	session, err := s.store.Sessions.GetByID(ctx, models.SessionID(sessionID))
	if err != nil {
		return nil, ErrSessionNotFound
	}

	if session.ExpiredAt.Before(s.now()) {
		_ = s.store.Sessions.Delete(ctx, session.SessionID)
		return nil, ErrSessionNotFound
	}

	return session, nil
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return nil
	}
	return s.store.Sessions.Delete(ctx, models.SessionID(sessionID))
}

func (s *Service) issueAuthResult(ctx context.Context, accountID int64) (*AuthResult, error) {
	userProfile, err := s.store.UserProfiles.GetByUserAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	session := models.Session{
		SessionID: models.SessionID(uuid.NewString()),
		UserID:    accountID,
		CreatedAt: s.now(),
		ExpiredAt: s.now().Add(SessionTTL),
	}
	if err := s.store.Sessions.Save(ctx, session); err != nil {
		return nil, err
	}

	return &AuthResult{
		User: User{
			UserAccountID: accountID,
			ProfileID:     userProfile.ProfileID,
			FirstName:     userProfile.FirstName,
			LastName:      userProfile.LastName,
			CreatedAt:     userProfile.CreatedAt,
		},
		Session: session,
	}, nil
}

func normalizeLogin(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeGender(value models.Gender) models.Gender {
	if value == models.Male {
		return models.Male
	}
	return models.Female
}
