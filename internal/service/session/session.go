package session

//go:generate mockgen -destination=./../mocks/session_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/service/session SessionService
import (
	"context"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session"
	"github.com/google/uuid"
)

type SessionService interface {
	Create(ctx context.Context, userID int64) (*models.Session, error)
	Get(ctx context.Context, sessionID models.SessionID) (*models.Session, error)
	Delete(ctx context.Context, sessionID models.SessionID) error
}

type sessionService struct {
	repo session.SessionRepo
}

func NewSessionService(repo session.SessionRepo) SessionService {
	return &sessionService{
		repo: repo,
	}
}

func (s *sessionService) Delete(ctx context.Context, sessionID models.SessionID) error {
	return s.repo.Delete(ctx, sessionID)
}

func (s *sessionService) validateSession(userID int64) error {
	if userID <= 0 {
		return errors.New("invalid user id")
	}
	return nil
}

const sessionTTL = 24 * time.Hour

func (s *sessionService) Create(ctx context.Context, userID int64) (*models.Session, error) {
	if err := s.validateSession(userID); err != nil {
		return nil, err
	}

	sess := models.Session{
		SessionID: models.SessionID(uuid.New().String()),
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiredAt: time.Now().Add(sessionTTL),
	}

	if err := s.repo.Save(ctx, sess); err != nil {
		return nil, err
	}

	return &sess, nil
}

func (s *sessionService) Get(ctx context.Context, sessionID models.SessionID) (*models.Session, error) {
	return s.repo.GetByID(ctx, sessionID)
}
