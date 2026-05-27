package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/model"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var ErrSessionNotFound = errors.New("session not found")

//go:generate mockgen -destination=mocks/session_repo_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/services/auth/internal/repository SessionRepo
type SessionRepo interface {
	Save(ctx context.Context, session model.Session) error
	Delete(ctx context.Context, id model.SessionID) error
	GetByID(ctx context.Context, id model.SessionID) (*model.Session, error)
}

type sessionRedis struct {
	client *redis.Client
}

func NewSessionStorage(client *redis.Client) SessionRepo {
	return &sessionRedis{client: client}
}

func (r *sessionRedis) Save(ctx context.Context, session model.Session) error {
	marshaled, err := json.Marshal(session)
	if err != nil {
		return err
	}

	start := time.Now()
	res := r.client.Set(ctx, string(session.SessionID), marshaled, time.Until(session.ExpiredAt))
	if res.Err() != nil {
		return res.Err()
	}
	logQuery(ctx, "sessionRepo.Save", start)
	return nil
}

func (r *sessionRedis) Delete(ctx context.Context, id model.SessionID) error {
	start := time.Now()
	res := r.client.Del(ctx, string(id))
	if res.Err() != nil {
		return res.Err()
	}
	logQuery(ctx, "sessionRepo.Delete", start)
	return nil
}

func (r *sessionRedis) GetByID(ctx context.Context, id model.SessionID) (*model.Session, error) {
	start := time.Now()
	res, err := r.client.Get(ctx, string(id)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	logQuery(ctx, "sessionRepo.GetByID", start)

	var session model.Session
	if err := json.Unmarshal(res, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func logQuery(ctx context.Context, query string, start time.Time) {
	logg := logger.FromContext(ctx)
	if logg == nil {
		return
	}
	logg.Debug("db query", zap.String("query", query), zap.Duration("duration_ms", time.Since(start)))
}
