package repository

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/cursor"
)

type FeedParams struct {
	Cursor *cursor.Cursor
	Limit  int
}

type inmemoryFeedRepo struct {
	feedRepo []models.Post
}

type FeedRepo interface {
	GetFeed(ctx context.Context, params FeedParams) ([]models.Post, error)
	Save(ctx context.Context, post models.Post) error
}

func NewFeedRepo() *inmemoryFeedRepo {
	return &inmemoryFeedRepo{}
}

func (r *inmemoryFeedRepo) GetFeed(ctx context.Context, params FeedParams) ([]models.Post, error) {

	fmt.Println("Feed repo start")

	slices.SortFunc(r.feedRepo, func(a, b models.Post) int {
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		} else if a.CreatedAt.After(a.CreatedAt) {
			return 1
		}
		return 0
	})

	limit := params.Limit + 1

	if params.Cursor == nil {
		return r.feedRepo[:limit], nil
	}

	for i, p := range r.feedRepo {
		if p.CreatedAt.After(params.Cursor.CreatedAt) {
			if i+limit > len(r.feedRepo) {
				return r.feedRepo[i:], nil
			}
			return r.feedRepo[i : i+limit], nil
		}
	}

	fmt.Println("Feed repo return")

	return nil, errors.New("No more posts")
}

func (r *inmemoryFeedRepo) Save(ctx context.Context, post models.Post) error {
	r.feedRepo = append(r.feedRepo, post)
	return nil
}
