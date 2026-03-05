package repository

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"

	"github.com/google/uuid"
)

type UserRepo interface {
	Save(ctx context.Context, user models.User)
	Delete(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, id uuid.UUID, user models.User) error

	GetByID(ctx context.Context, id uuid.UUID) (models.User, error)
	GetByEmail(ctx context.Context, email string) (models.User, error)
	GetByPhone(ctx context.Context, phone string) (models.User, error)

	List(ctx context.Context, offset, limit int) ([]models.User, error)
}

type inmemoryUserRepo struct {
	users []models.User
}

func (r *inmemoryUserRepo) Save(ctx context.Context, user models.User) {
	r.users = append(r.users, user)
}

func (r *inmemoryUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	for i, u := range r.users {
		if u.ID == id {
			r.users = append(r.users[:i], r.users[i+1:]...)
			return nil
		}
	}
	return errors.New("user not found")
}

func (r *inmemoryUserRepo) GetByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return models.User{}, errors.New("user not found")
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

func (r *inmemoryUserRepo) Update(ctx context.Context, id uuid.UUID, user models.User) error {
	for i, u := range r.users {
		if u.ID == id {
			r.users[i] = user
			r.users[i].ID = id
			return nil
		}
	}
	return errors.New("user not found")
}

func (r *inmemoryUserRepo) List(ctx context.Context, offset, limit int) ([]models.User, error) {
	if offset+limit > len(r.users) {
		return nil, errors.New("out of range")
	}
	return r.users[offset : offset+limit], nil
}
