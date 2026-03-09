package service

import (
	"context"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/google/uuid"
)

type SessionService interface {
	Create(ctx context.Context, userID uuid.UUID) (models.Session, error)
	Get(ctx context.Context, sessionID models.SessionID) (models.Session, error)
}

type sessionService struct {
	repo repository.SessionRepo
}

func NewSessionService(repo repository.SessionRepo) SessionService {
	return &sessionService{
		repo: repo,
	}
}

func (s *sessionService) validateSession(userID uuid.UUID) error {
	if userID == uuid.Nil {
		return errors.New("invalid user id")
	}
	return nil
}

const sessionTTL = 24 * time.Hour

func (s *sessionService) Create(ctx context.Context, userID uuid.UUID) (models.Session, error) {
	if err := s.validateSession(userID); err != nil {
		return models.Session{}, err
	}

	sess := models.Session{
		SessionID: models.SessionID(uuid.New().String()),
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiredAt: time.Now().Add(sessionTTL),
	}

	s.repo.Save(ctx, sess)
	return sess, nil
}

func (s *sessionService) Get(ctx context.Context, sessionID models.SessionID) (models.Session, error) {
	return s.repo.GetByID(ctx, sessionID)
}
