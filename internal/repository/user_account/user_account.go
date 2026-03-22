package useraccount

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
)

type UserAccountRepo interface {
	Save(ctx context.Context, user models.UserAccount) (int64, error)
	Delete(ctx context.Context, id int64) error

	Get(ctx context.Context, id int64) (*models.UserAccount, error)
	GetByEmail(ctx context.Context, email string) (*models.UserAccount, error)
	GetByPhone(ctx context.Context, phone string) (*models.UserAccount, error)
	GetByUsername(ctx context.Context, username string) (*models.UserAccount, error)
	GetByUid(ctx context.Context, uid uuid.UUID) (*models.UserAccount, error)

	List(ctx context.Context, offset, limit int) []models.UserAccount
}

type inmemoryUserRepo struct {
	mu           sync.RWMutex
	userAccounts map[int64]models.UserAccount
}

func NewUserRepo() UserAccountRepo {
	repo := inmemoryUserRepo{}
	repo.userAccounts = make(map[int64]models.UserAccount)
	return &repo
}

func (r *inmemoryUserRepo) Save(ctx context.Context, user models.UserAccount) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.userAccounts[user.ID] = user
	return user.ID, nil
}

func (r *inmemoryUserRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.userAccounts[id]

	if ok {
		delete(r.userAccounts, id)
		return nil
	}

	return errors.New("user not found")
}

func (r *inmemoryUserRepo) Get(ctx context.Context, id int64) (*models.UserAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.userAccounts[id]

	if !ok {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

func (r *inmemoryUserRepo) GetByEmail(ctx context.Context, email string) (*models.UserAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.userAccounts {
		if *u.Email == email {
			return &u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (r *inmemoryUserRepo) GetByPhone(ctx context.Context, phone string) (*models.UserAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.userAccounts {
		if *u.Phone == phone {
			return &u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (r *inmemoryUserRepo) List(ctx context.Context, offset, limit int) []models.UserAccount {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if offset >= len(r.userAccounts) {
		return []models.UserAccount{}
	}
	if offset+limit > len(r.userAccounts) {
		return slices.Collect(maps.Values(r.userAccounts))[offset:]
	}

	return slices.Collect(maps.Values(r.userAccounts))[offset : offset+limit]
}

func (r *inmemoryUserRepo) GetByUsername(ctx context.Context, username string) (*models.UserAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.userAccounts {
		if u.Username == username {
			return &u, nil
		}
	}
	return nil, errors.New("User not found")
}

func (r *inmemoryUserRepo) GetByUid(ctx context.Context, uid uuid.UUID) (*models.UserAccount, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.userAccounts {
		if u.Uid == uid {
			return &u, nil
		}
	}
	return nil, errors.New("User not found")
}
