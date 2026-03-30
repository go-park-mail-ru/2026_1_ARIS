package repository

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type MessageRepo interface {
	Save(ctx context.Context, msg models.Message) error
	GetByID(ctx context.Context, id int64) (*models.Message, error)
	GetByChatID(ctx context.Context, chatID int64, limit, offset int) ([]models.Message, error)
	Delete(ctx context.Context, id int64) error
}

type inmemoryMessageRepo struct {
	mu       sync.RWMutex
	messages map[int64]models.Message
}

func NewMessageRepo() MessageRepo {
	return &inmemoryMessageRepo{
		messages: make(map[int64]models.Message),
	}
}

func (r *inmemoryMessageRepo) Save(ctx context.Context, msg models.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages[msg.ID] = msg
	return nil
}

func (r *inmemoryMessageRepo) GetByID(ctx context.Context, id int64) (*models.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	msg, ok := r.messages[id]
	if !ok {
		return nil, errors.New("message not found")
	}
	return &msg, nil
}

func (r *inmemoryMessageRepo) GetByChatID(ctx context.Context, chatID int64, limit, offset int) ([]models.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var msgs []models.Message
	for _, msg := range r.messages {
		if msg.ChatID == chatID {
			msgs = append(msgs, msg)
		}
	}

	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].CreatedAt.Before(msgs[j].CreatedAt)
	})
	if offset > len(msgs) {
		return []models.Message{}, nil
	}
	end := offset + limit
	if end > len(msgs) {
		end = len(msgs)
	}
	return msgs[offset:end], nil
}

func (r *inmemoryMessageRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.messages[id]; !ok {
		return errors.New("message not found")
	}
	delete(r.messages, id)
	return nil
}
