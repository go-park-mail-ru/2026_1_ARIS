package repository

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/cursor"

	"github.com/google/uuid"
)

type FeedParams struct {
	Cursor *cursor.Cursor
	Limit  int
}

type PostRepo interface {
	Save(ctx context.Context, post models.Post) error
	Delete(ctx context.Context, id uuid.UUID) error

	List(ctx context.Context, offset, limit int) ([]models.Post, error)
	GetPostByID(id uuid.UUID) (models.Post, error)

	GetFeed(ctx context.Context, params FeedParams) ([]models.Post, error)
}

type inmemoryPostRepo struct {
	Posts map[uuid.UUID]models.Post
}

func NewPostRepo() PostRepo {
	repo := inmemoryPostRepo{}
	repo.Posts = make(map[uuid.UUID]models.Post)
	return &repo
}

func (r *inmemoryPostRepo) GetFeed(ctx context.Context, params FeedParams) ([]models.Post, error) {

	fmt.Println("Feed repo start")

	feedSlice := slices.Collect(maps.Values(r.Posts))

	slices.SortFunc(feedSlice, func(a, b models.Post) int {
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		} else if a.CreatedAt.After(a.CreatedAt) {
			return 1
		}
		return 0
	})

	limit := params.Limit + 1

	if params.Cursor == nil {
		fmt.Println("Пустой курсор")
		fmt.Println("Возвращено из репо", feedSlice)
		if limit > len(feedSlice) {
			return feedSlice[:], nil
		} else {
			return feedSlice[:limit], nil
		}
	}

	for i, p := range feedSlice {
		if p.CreatedAt.After(params.Cursor.CreatedAt) {
			if i+limit > len(feedSlice) {
				return feedSlice[i:], nil
			}
			return feedSlice[i : i+limit], nil
		}
	}

	fmt.Println("Feed repo return")

	return nil, errors.New("No more posts")
}

func (r *inmemoryPostRepo) Save(ctx context.Context, post models.Post) error {
	_, ok := r.Posts[post.ID]
	if ok {
		return nil
	}

	r.Posts[post.ID] = post

	return nil
}

func (r *inmemoryPostRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, ok := r.Posts[id]

	if !ok {
		return errors.New("post not found")
	}

	delete(r.Posts, id)
	return nil
}

func (r *inmemoryPostRepo) List(ctx context.Context, offset, limit int) ([]models.Post, error) {
	if offset+limit > len(r.Posts) {
		return nil, errors.New("out of range")
	}

	return slices.Collect(maps.Values(r.Posts))[offset:offset:limit], nil
}

func (r *inmemoryPostRepo) GetPostByID(id uuid.UUID) (models.Post, error) {
	fmt.Println("Все посты из репо: ", r.Posts)
	fmt.Println(id)
	profile, ok := r.Posts[id]
	if !ok {
		return models.Post{}, errors.New("Profile not found")
	}

	return profile, nil
}
