package service

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/cursor"
)

type FeedResult struct {
	Posts   []models.Post `json:"posts"`
	Cursor  string        `json:"cursor,omitempty"`
	HasMore bool          `json:"hasMore"`
}

type FeedServide interface {
	GetFeed(ctx context.Context, rawCursor string, limit int) (FeedResult, error)
	Save(ctx context.Context, post models.Post) error
}

type feedService struct {
	repo repository.FeedRepo
}

func NewFeedService(repo repository.FeedRepo) *feedService {
	return &feedService{repo: repo}
}

func (service *feedService) GetFeed(ctx context.Context, rawCursor string, limit int) (FeedResult, error) {

	fmt.Println("Feed service")

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var cur *cursor.Cursor

	if rawCursor != "" {
		decoded, err := cursor.Decode(rawCursor)
		if err != nil {
			return FeedResult{}, err
		}

		cur = &decoded
	}

	_ = cur

	fmt.Println("Before repo in service")
	params := repository.FeedParams{Cursor: cur, Limit: limit}
	fmt.Println(params)
	posts, err := service.repo.GetFeed(ctx, params)
	fmt.Println("Post in service returned")
	if err != nil {
		return FeedResult{}, err
	}

	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}

	var nextCursor string
	if hasMore && len(posts) > 0 {
		lastPost := posts[len(posts)-1]
		nextCursor = cursor.Encode(cursor.Cursor{
			CreatedAt: lastPost.CreatedAt,
			ID:        0, //uuid.Max,
		})
	}

	fmt.Println("Feed service return")

	return FeedResult{
		Posts:   posts,
		Cursor:  nextCursor,
		HasMore: hasMore,
	}, nil
}

func (service *feedService) Save(ctx context.Context, post models.Post) error {
	return service.repo.Save(ctx, post)
}
