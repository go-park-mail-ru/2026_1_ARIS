package session

//go:generate mockgen -destination=./../mocks/session_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/session SessionRepo
import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type SessionRepo interface {
	Save(ctx context.Context, session models.Session) error
	Delete(ctx context.Context, id models.SessionID) error
	GetByID(ctx context.Context, id models.SessionID) (*models.Session, error)
}

type sessionRedis struct {
	client *redis.Client
	// logger
}

func NewSessionStorage(client *redis.Client) SessionRepo {
	return &sessionRedis{
		client: client,
	}
}

func (r *sessionRedis) Save(ctx context.Context, session models.Session) error {
	logger := logger.FromContext(ctx)
	sessionID := session.SessionID

	marshaled, err := json.Marshal(session)
	if err != nil {
		return err
	}

	start := time.Now()
	res := r.client.Set(ctx, string(sessionID), marshaled, time.Until(session.ExpiredAt))
	if res.Err() != nil {
		return res.Err()
	}
	logger.Debug("db query",
		zap.String("query", "sessionRepo.Save"),
		zap.Duration("duration_ms", time.Since(start)))

	return nil
}

func (r *sessionRedis) Delete(ctx context.Context, id models.SessionID) error {
	logger := logger.FromContext(ctx)
	start := time.Now()
	res := r.client.Del(ctx, string(id))
	if res.Err() != nil {
		return res.Err()
	}
	logger.Debug("db query",
		zap.String("query", "sessionRepo.Delete"),
		zap.Duration("duration_ms", time.Since(start)))

	return nil
}

func (r *sessionRedis) GetByID(ctx context.Context, id models.SessionID) (*models.Session, error) {
	logger := logger.FromContext(ctx)
	start := time.Now()
	res, err := r.client.Get(ctx, string(id)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, xerrors.SessionNotFound
		}
		return nil, err
	}
	logger.Debug("db query",
		zap.String("query", "sessionRepo.GetByID"),
		zap.Duration("duration_ms", time.Since(start)))

	var session models.Session

	if err := json.Unmarshal(res, &session); err != nil {
		fmt.Println("json.Unmarshal error", string(res))
		return nil, err
	}

	return &session, nil
}
