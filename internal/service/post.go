package service

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/cursor"
	"github.com/google/uuid"
)

type FeedResult struct {
	Posts   []models.Post `json:"posts"`
	Cursor  string        `json:"cursor,omitempty"`
	HasMore bool          `json:"hasMore"`
}

type postService struct {
	PostRepo       repository.PostRepo
	ProfileRepo    repository.ProfileRepo
	LikeToPostRepo repository.LikeToPostRepo
	CommentRepo    repository.CommentRepo
	RepostRepo     repository.RepostRepo
}

type PostService interface {
	GetPostAuthor(postID uuid.UUID) (*models.Profile, error)
	GetFeed(ctx context.Context, rawCursor string, limit int) (FeedResult, error)
	Save(ctx context.Context, post models.Post) error
	GetLikeCount(ctx context.Context, postID uuid.UUID) int
	GetCommentCount(ctx context.Context, postID uuid.UUID) int
	GetRepostCount(ctx context.Context, postID uuid.UUID) int
	GetPublicFeed(ctx context.Context, params repository.FeedParams) ([]models.Post, error)
	GetPublicPopularPosts(ctx context.Context) ([]models.Post, error)
	GetPopularPosts(ctx context.Context) ([]models.Post, error)
}

func NewPostService(postRepo repository.PostRepo,
	profileRepo repository.ProfileRepo,
	likeToPostRepo repository.LikeToPostRepo,
	commentRepo repository.CommentRepo,
	repostRepo repository.RepostRepo) PostService {

	return &postService{
		PostRepo:       postRepo,
		ProfileRepo:    profileRepo,
		LikeToPostRepo: likeToPostRepo,
		CommentRepo:    commentRepo,
		RepostRepo:     repostRepo,
	}
}

func (s *postService) GetFeed(ctx context.Context, rawCursor string, limit int) (FeedResult, error) {
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

	params := repository.FeedParams{Cursor: cur, Limit: limit}

	posts, err := s.PostRepo.GetFeed(ctx, params)
	if err != nil {
		return FeedResult{}, err
	}

	filtered := make([]models.Post, 0, len(posts))

	for _, post := range posts {
		if post.IsPublicDemo {
			continue
		}
		filtered = append(filtered, post)
	}

	posts = filtered

	hasMore := len(posts) > limit
	if hasMore {
		posts = posts[:limit]
	}

	var nextCursor string
	if hasMore && len(posts) > 0 {
		lastPost := posts[len(posts)-1]
		nextCursor = cursor.Encode(cursor.Cursor{
			CreatedAt: lastPost.CreatedAt,
			ID:        lastPost.ID,
		})
	}

	return FeedResult{
		Posts:   posts,
		Cursor:  nextCursor,
		HasMore: hasMore,
	}, nil
}

func (s *postService) GetPublicFeed(ctx context.Context, params repository.FeedParams) ([]models.Post, error) {
	return s.PostRepo.GetPublicFeed(ctx, params)
}

func (s *postService) GetPostAuthor(postID uuid.UUID) (*models.Profile, error) {
	post, err := s.PostRepo.GetPostByID(postID)
	if err != nil {
		return nil, err
	}

	profileID := post.AuthorID

	profile, err := s.ProfileRepo.GetProfileByID(profileID)

	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *postService) Save(ctx context.Context, post models.Post) error {
	return s.PostRepo.Save(ctx, post)
}

func (s *postService) GetLikeCount(ctx context.Context, postID uuid.UUID) int {
	return s.LikeToPostRepo.GetLikeCountOnPost(postID)
}

func (s *postService) GetCommentCount(ctx context.Context, postID uuid.UUID) int {
	return s.CommentRepo.GetCommentCount(postID)
}

func (s *postService) GetRepostCount(ctx context.Context, postID uuid.UUID) int {
	return s.RepostRepo.GetRepostCount(ctx, postID)
}

func (s *postService) GetPublicPopularPosts(ctx context.Context) ([]models.Post, error) {
	return nil, nil
}

func (s *postService) GetPopularPosts(ctx context.Context) ([]models.Post, error) {
	return nil, nil
}
