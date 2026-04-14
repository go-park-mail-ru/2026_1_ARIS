package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/redis/go-redis/v9"
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
	sessionID := session.SessionID

	marshaled, err := json.Marshal(session)
	if err != nil {
		return err
	}

	res := r.client.Set(ctx, string(sessionID), marshaled, time.Until(session.ExpiredAt))
	if res.Err() != nil {
		return res.Err()
	}

	fmt.Println("Session Saved:", sessionID, string(marshaled))

	return nil
}

func (r *sessionRedis) Delete(ctx context.Context, id models.SessionID) error {
	res := r.client.Del(ctx, string(id))
	if res.Err() != nil {
		return res.Err()
	}

	return nil
}

func (r *sessionRedis) GetByID(ctx context.Context, id models.SessionID) (*models.Session, error) {

	res, err := r.client.Get(ctx, string(id)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, xerrors.SessionNotFound
		}
		return nil, err
	}

	var session models.Session

	if err := json.Unmarshal(res, &session); err != nil {
		fmt.Println("json.Unmarshal error", string(res))
		return nil, err
	}

	return &session, nil
}
