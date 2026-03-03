package repository

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type PostRepo interface {
	Save(ctx context.Context, post models.Post)
	Delete(ctx context.Context, id models.PostID) error

	List(ctx context.Context, offset, limit int) ([]models.Post, error)
}

type inmemoryPostRepo struct {
	posts []models.Post
}

func (r *inmemoryPostRepo) Save(ctx context.Context, post models.Post) {
	r.posts = append(r.posts, post)
}

func (r *inmemoryPostRepo) Delete(ctx context.Context, id models.PostID) error {
	for i, p := range r.posts {
		if p.ID == id {
			r.posts = append(r.posts[:i], r.posts[i+1:]...)
			return nil
		}
	}
	return errors.New("post not found")
}

func (r *inmemoryPostRepo) List(ctx context.Context, offset, limit int) ([]models.Post, error) {
	if offset+limit > len(r.posts) {
		return nil, errors.New("out of range")
	}
	return r.posts[offset : offset+limit], nil
}
