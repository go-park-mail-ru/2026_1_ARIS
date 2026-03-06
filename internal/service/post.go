package service

import (
	"context"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/cursor"
	"github.com/google/uuid"
)

type postService struct {
	PostRepo    repository.PostRepo
	ProfileRepo repository.ProfileRepo
}

type PostService interface {
	GetPostAuthor(postID uuid.UUID) (models.Profile, error)
	GetFeed(ctx context.Context, rawCursor string, limit int) (FeedResult, error)
}

func NewPostService(postRepo repository.PostRepo, profileRepo repository.ProfileRepo) *postService {
	return &postService{
		PostRepo:    postRepo,
		ProfileRepo: profileRepo,
	}
}

func (service *postService) GetFeed(ctx context.Context, rawCursor string, limit int) (FeedResult, error) {

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
	posts, err := service.PostRepo.GetFeed(ctx, params)
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

func (r *postService) GetPostAuthor(postID uuid.UUID) (models.Profile, error) {
	post, err := r.PostRepo.GetPostByID(postID)
	fmt.Println("Post get in PostService:", post)
	if err != nil {
		return models.Profile{}, err
	}

	profileID := post.AuthorID

	profile, err := r.ProfileRepo.GetProfileByID(profileID)

	if err != nil {
		return models.Profile{}, err
	}

	return profile, nil
}
