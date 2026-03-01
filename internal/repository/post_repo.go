package repository

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/model"
)

type PostRepo interface {
	Save(ctx context.Context, post model.Post)
	Delete(ctx context.Context, id model.PostID) error

	List(ctx context.Context, ofset, limit int) ([]model.Post, error)
}

type inmemoryPostRepo struct {
	posts []model.Post
}

func (r *inmemoryPostRepo) Save(ctx context.Context, post model.Post) {
	r.posts = append(r.posts, post)
}

func (r *inmemoryPostRepo) Delete(ctx context.Context, id model.PostID) error {
	for i, p := range r.posts {
		if p.ID == id {
			r.posts = append(r.posts[:i], r.posts[i+1:]...)
			return nil
		}
	}
	return errors.New("post not found")
}

func (r *inmemoryPostRepo) List(ctx context.Context, ofset, limit int) ([]model.Post, error) {
	if ofset+limit > len(r.posts) {
		return nil, errors.New("out of range")
	}
	return r.posts[ofset : ofset+limit], nil
}
