package repository

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/model"
)

type UserRepo interface {
	Save(ctx context.Context, user model.User)
	Delete(ctx context.Context, id model.UserID) error
	Update(ctx context.Context, id model.UserID, user model.User) error

	GetByID(ctx context.Context, id model.UserID) (model.User, error)
	GetByEmail(ctx context.Context, email string) (model.User, error)
	GetByUsername(ctx context.Context, username string) (model.User, error)
	GetByPhone(ctx context.Context, phone string) (model.User, error)

	List(ctx context.Context, ofset, limit int) ([]model.User, error)
}

type inmemoryUserRepo struct {
	users []model.User
}

func (r *inmemoryUserRepo) Save(ctx context.Context, user model.User) {
	r.users = append(r.users, user)
}

func (r *inmemoryUserRepo) Delete(ctx context.Context, id model.UserID) error {
	for i, u := range r.users {
		if u.ID == id {
			r.users = append(r.users[:i], r.users[i+1:]...)
			return nil
		}
	}
	return errors.New("user not found")
}

func (r *inmemoryUserRepo) GetByID(ctx context.Context, id model.UserID) (model.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return model.User{}, errors.New("user not found")
}

func (r *inmemoryUserRepo) GetByEmail(ctx context.Context, email string) (model.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return model.User{}, errors.New("user not found")
}

func (r *inmemoryUserRepo) GetByUsername(ctx context.Context, username string) (model.User, error) {
	for _, u := range r.users {
		if u.Username == username {
			return u, nil
		}
	}
	return model.User{}, errors.New("user not found")
}

func (r *inmemoryUserRepo) GetByPhone(ctx context.Context, phone string) (model.User, error) {
	for _, u := range r.users {
		if u.Phone == phone {
			return u, nil
		}
	}
	return model.User{}, errors.New("user not found")
}

func (r *inmemoryUserRepo) Update(ctx context.Context, id model.UserID, user model.User) error {
	for i, u := range r.users {
		if u.ID == id {
			r.users[i] = user
			r.users[i].ID = id
			return nil
		}
	}
	return errors.New("user not found")
}

func (r *inmemoryUserRepo) List(ctx context.Context, ofset, limit int) ([]model.User, error) {
	if ofset+limit > len(r.users) {
		return nil, errors.New("out of range")
	}
	return r.users[ofset : ofset+limit], nil
}
