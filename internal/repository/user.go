package repository

import (
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"

	"github.com/google/uuid"
)

type UserRepo interface {
	Save(ctx context.Context, user models.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	//Update(ctx context.Context, id uuid.UUID, user models.User) error

	GetByID(ctx context.Context, id uuid.UUID) (models.User, error)
	GetByEmail(ctx context.Context, email string) (models.User, error)
	GetByPhone(ctx context.Context, phone string) (models.User, error)

	List(ctx context.Context, offset, limit int) ([]models.User, error)
}

type inmemoryUserRepo struct {
	users map[uuid.UUID]models.User
}

func NewUserRepo() UserRepo {
	repo := inmemoryUserRepo{}
	repo.users = make(map[uuid.UUID]models.User)
	return &repo
}

func (r *inmemoryUserRepo) Save(ctx context.Context, user models.User) error {
	r.users[user.ID] = user
	//r.users = append(r.users, user)
	return nil
}

func (r *inmemoryUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, ok := r.users[id]

	if ok {
		delete(r.users, id)
		return nil
	}

	return errors.New("user not found")
}

func (r *inmemoryUserRepo) GetByID(ctx context.Context, id uuid.UUID) (models.User, error) {

	user, ok := r.users[id]

	if !ok {
		return models.User{}, errors.New("user not found")
	}
	return user, nil
}

func (r *inmemoryUserRepo) GetByEmail(ctx context.Context, email string) (models.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return models.User{}, errors.New("user not found")
}

func (r *inmemoryUserRepo) GetByPhone(ctx context.Context, phone string) (models.User, error) {
	for _, u := range r.users {
		if u.Phone == phone {
			return u, nil
		}
	}
	return models.User{}, errors.New("user not found")
}

func (r *inmemoryUserRepo) List(ctx context.Context, offset, limit int) ([]models.User, error) {
	if offset+limit > len(r.users) {
		return nil, errors.New("out of range")
	}

	return slices.Collect(maps.Values(r.users))[offset : offset+limit], nil
}
