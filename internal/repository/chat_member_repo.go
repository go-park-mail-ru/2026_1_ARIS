package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type ChatMemberRepo interface {
	Save(ctx context.Context, member models.ChatMember) error
	GetByChatID(ctx context.Context, chatID int64) ([]models.ChatMember, error)
	GetByUserID(ctx context.Context, userID int64) ([]models.ChatMember, error)
	Delete(ctx context.Context, id int64) error
}

type inmemoryChatMemberRepo struct {
	mu      sync.RWMutex
	members map[int64]models.ChatMember
}

func NewChatMemberRepo() ChatMemberRepo {
	return &inmemoryChatMemberRepo{
		members: make(map[int64]models.ChatMember),
	}
}

func (r *inmemoryChatMemberRepo) Save(ctx context.Context, member models.ChatMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.members[member.ID] = member
	return nil
}

func (r *inmemoryChatMemberRepo) GetByChatID(ctx context.Context, chatID int64) ([]models.ChatMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var res []models.ChatMember
	for _, m := range r.members {
		if m.ChatID == chatID {
			res = append(res, m)
		}
	}
	return res, nil
}

func (r *inmemoryChatMemberRepo) GetByUserID(ctx context.Context, userID int64) ([]models.ChatMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var res []models.ChatMember
	for _, m := range r.members {
		if m.MemberID == userID {
			res = append(res, m)
		}
	}
	return res, nil
}

func (r *inmemoryChatMemberRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.members[id]; !ok {
		return errors.New("member not found")
	}
	delete(r.members, id)
	return nil
}
