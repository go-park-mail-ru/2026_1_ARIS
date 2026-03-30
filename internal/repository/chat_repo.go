package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type ChatRepo interface {
	Save(ctx context.Context, chat models.Chat) error
	GetByID(ctx context.Context, id int64) (*models.Chat, error)
	Delete(ctx context.Context, id int64) error
}

type inmemoryChatRepo struct {
	mu    sync.RWMutex
	chats map[int64]models.Chat
}

func NewChatRepo() ChatRepo {
	return &inmemoryChatRepo{
		chats: make(map[int64]models.Chat),
	}
}

func (r *inmemoryChatRepo) Save(ctx context.Context, chat models.Chat) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.chats[chat.ID] = chat
	return nil
}

func (r *inmemoryChatRepo) GetByID(ctx context.Context, id int64) (*models.Chat, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	chat, ok := r.chats[id]
	if !ok {
		return nil, errors.New("chat not found")
	}
	return &chat, nil
}

func (r *inmemoryChatRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.chats[id]; !ok {
		return errors.New("chat not found")
	}
	delete(r.chats, id)
	return nil
}
