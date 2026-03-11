package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type SessionRepo interface {
	Save(ctx context.Context, session models.Session)
	Delete(ctx context.Context, id models.SessionID) error
	GetByID(ctx context.Context, id models.SessionID) (*models.Session, error)
}

type inmemorySessionRepo struct {
	mu       sync.RWMutex
	sessions map[models.SessionID]models.Session
}

func NewSessionRepo() SessionRepo {
	return &inmemorySessionRepo{
		sessions: make(map[models.SessionID]models.Session),
	}
}

func (r *inmemorySessionRepo) Save(ctx context.Context, session models.Session) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions[session.SessionID] = session
}

func (r *inmemorySessionRepo) Delete(ctx context.Context, id models.SessionID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.sessions[id]
	if !ok {
		return errors.New("session not found")
	}
	delete(r.sessions, id)
	return nil
}

func (r *inmemorySessionRepo) GetByID(ctx context.Context, id models.SessionID) (*models.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	val, ok := r.sessions[id]
	if !ok {
		return nil, errors.New("session not found")
	}
	return &val, nil
}
