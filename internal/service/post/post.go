package post

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/handler/dto"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/comment"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/like"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/post"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/profile"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/repost"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/cursor"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FeedResult struct {
	Posts   []models.Post `json:"posts"`
	Cursor  string        `json:"cursor,omitempty"`
	HasMore bool          `json:"hasMore"`
}

type postService struct {
	pool              *pgxpool.Pool
	PostRepo          post.PostRepo
	PostWithMediaRepo post.PostWithMediaRepo
	ProfileRepo       profile.ProfileRepo
	CommentRepo       comment.CommentRepo
	RepostRepo        repost.RepostRepo
	LikeRepo          like.LikeRepo
}

type FeedParams struct {
	Cursor *cursor.Cursor
	Limit  int
}

type PostService interface {
	Get(ctx context.Context, postID int64) (*models.Post, error)
	getFeed(ctx context.Context, getCursoredPosts func(ctx context.Context, params FeedParams) ([]models.Post, error), rawCursor string, limit int) (FeedResult, error)
	GetFeed(ctx context.Context, rawCursor string, limit int) (FeedResult, error)
	GetPublicFeed(ctx context.Context, rawCursor string, limit int) (FeedResult, error)
	GetPostAuthor(ctx context.Context, postID int64) (*models.Profile, error)
	Save(ctx context.Context, post models.Post, profileID int64, attachedMedia *[]dto.MediaRequestData) (int64, error)
	GetLikeCount(ctx context.Context, postID int64) int
	GetCommentCount(ctx context.Context, postID int64) int
	GetRepostCount(ctx context.Context, postID int64) int
	GetPublicPopularPosts(ctx context.Context) ([]models.Post, error)
	GetPopularPosts(ctx context.Context) ([]models.Post, error)
	AttachMedia(ctx context.Context, postID, authorID int64, Tx pgx.Tx, mediaID []dto.MediaRequestData) error
	detachMedia(ctx context.Context, postID, authorID int64, Tx pgx.Tx) error
	Delete(ctx context.Context, postID int64) error
	Update(ctx context.Context, postID, authorID int64, dto dto.PostRequestDTO) error
}

func NewPostService(postRepo post.PostRepo,
	postWithMediaRepo post.PostWithMediaRepo,
	profileRepo profile.ProfileRepo,
	commentRepo comment.CommentRepo,
	repostRepo repost.RepostRepo,
	likeRepo like.LikeRepo,
	pool *pgxpool.Pool) PostService {

	return &postService{
		PostRepo:          postRepo,
		ProfileRepo:       profileRepo,
		CommentRepo:       commentRepo,
		RepostRepo:        repostRepo,
		LikeRepo:          likeRepo,
		PostWithMediaRepo: postWithMediaRepo,
		pool:              pool,
	}
}

// type attachmentError struct {
// 	Err error `json:"error"`
// 	Pos int   `json:"position"`
// }

// type mediaErrors struct {
// 	Errs []attachmentError `json:"errors"`
// }

func (s *postService) detachMedia(ctx context.Context, postID, authorID int64, Tx pgx.Tx) error {
	post, err := s.Get(ctx, int64(postID))
	if err != nil {
		return err
	}

	if authorID != post.AuthorID {
		return xerrors.AccessDeny
	}

	if err := s.PostWithMediaRepo.DeletePostMedia(ctx, postID, Tx); err != nil {
		return err
	}

	return nil
}

func (s *postService) Update(ctx context.Context, postID, authorID int64, dto dto.PostRequestDTO) error {
	Tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer Tx.Rollback(ctx)

	post, err := s.Get(ctx, int64(postID))
	if err != nil {
		return err
	}

	if authorID != post.AuthorID {
		return xerrors.AccessDeny
	}

	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if dto.Text != nil && post.Text != nil && *dto.Text != *post.Text {
		setClauses = append(setClauses, fmt.Sprintf("post_text=$%d", argIdx))
		args = append(args, *dto.Text)
		argIdx++
	}

	if len(args) != 0 {
		args = append(args, postID)
		query := fmt.Sprintf("UPDATE post SET %s WHERE id=$%d", strings.Join(setClauses, ", "), argIdx)

		err := s.PostRepo.Update(ctx, query, args)
		if err != nil {
			return err
		}
	}

	if err := s.detachMedia(ctx, postID, authorID, Tx); err != nil {
		if !errors.Is(err, xerrors.NoRowsAffected) {
			return err
		}
	}

	if dto.Media != nil {
		if err := s.AttachMedia(ctx, postID, authorID, Tx, *dto.Media); err != nil {
			return err
		}
	}

	return Tx.Commit(ctx)
}

func (s *postService) Get(ctx context.Context, postID int64) (*models.Post, error) {
	return s.PostRepo.Get(ctx, postID)
}

func (s *postService) Delete(ctx context.Context, postID int64) error {
	return s.PostRepo.Delete(ctx, postID)
}

func (s *postService) AttachMedia(ctx context.Context, postID, authorID int64, Tx pgx.Tx, mediaID []dto.MediaRequestData) error {
	mediaIDs := make([]int64, len(mediaID))
	for i, media := range mediaID {
		mediaIDs[i] = media.MediaID
	}

	for i, media := range mediaID {
		postWithMedia := models.NewPostWithMedia(postID, media.MediaID, i)
		err := s.PostWithMediaRepo.Save(ctx, Tx, *postWithMedia)
		if err != nil {
			// Определить тип ошибки
			return fmt.Errorf("can't save media:%d", i)
		}
	}
	return nil
}

// получение ленты, будь то публичных или нет постов (унификация чезер callbach-функцию getCursoredPosts)
func (s *postService) getFeed(ctx context.Context, getCursoredPosts func(ctx context.Context, params FeedParams) ([]models.Post, error), rawCursor string, limit int) (FeedResult, error) {
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

	params := FeedParams{Cursor: cur, Limit: limit}

	posts, err := getCursoredPosts(ctx, params)
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
			ID:        lastPost.Uid,
		})
	}

	return FeedResult{
		Posts:   posts,
		Cursor:  nextCursor,
		HasMore: hasMore,
	}, nil
}

// получение ленты
func (s *postService) GetFeed(ctx context.Context, rawCursor string, limit int) (FeedResult, error) {
	return s.getFeed(ctx, s.getCursoredPosts, rawCursor, limit)
}

// получение публичной ленты
func (s *postService) GetPublicFeed(ctx context.Context, rawCursor string, limit int) (FeedResult, error) {
	return s.getFeed(ctx, s.getCursoredPublicPosts, rawCursor, limit)
}

func (s *postService) GetPostAuthor(ctx context.Context, postID int64) (*models.Profile, error) {
	post, err := s.PostRepo.Get(ctx, postID)
	if err != nil {
		return nil, err
	}

	profileID := post.AuthorID

	profile, err := s.ProfileRepo.Get(ctx, profileID)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *postService) Save(ctx context.Context, post models.Post, profileID int64, attachedMedia *[]dto.MediaRequestData) (int64, error) {
	Tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer Tx.Rollback(ctx)

	postID, err := s.PostRepo.Save(ctx, Tx, post)
	if err != nil {
		return 0, err
	}

	if attachedMedia != nil {
		// проверка на тип
		err := s.AttachMedia(ctx, postID, profileID, Tx, *attachedMedia)
		if err != nil {
			return 0, err
		}
	}

	return postID, Tx.Commit(ctx)
}

func (s *postService) GetLikeCount(ctx context.Context, postID int64) int {
	return s.LikeRepo.GetLikeCountOnPost(ctx, postID)
}

func (s *postService) GetCommentCount(ctx context.Context, postID int64) int {
	return s.CommentRepo.GetCommentCount(ctx, postID)
}

func (s *postService) GetRepostCount(ctx context.Context, postID int64) int {
	return s.RepostRepo.GetRepostCount(ctx, postID)
}

// тут пусто
func (s *postService) GetPublicPopularPosts(ctx context.Context) ([]models.Post, error) {
	return nil, nil
}

// тут пусто
func (s *postService) GetPopularPosts(ctx context.Context) ([]models.Post, error) {
	return nil, nil
}

// получение постов с учётом курсора
func (s *postService) getCursoredPosts(ctx context.Context, params FeedParams) ([]models.Post, error) {
	allPosts, err := s.PostRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	feedSlice := make([]models.Post, 0, len(allPosts))

	for _, p := range allPosts {
		if p.IsPublicDemo {
			continue
		}
		feedSlice = append(feedSlice, p)
	}

	slices.SortFunc(feedSlice, func(a, b models.Post) int {
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		} else if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		return 0
	})

	limit := params.Limit + 1

	if params.Cursor == nil {
		if limit > len(feedSlice) {
			return feedSlice[:], nil
		} else {
			return feedSlice[:limit], nil
		}
	}

	for i, p := range feedSlice {
		if p.CreatedAt.After(params.Cursor.CreatedAt) && p.Uid.String() != params.Cursor.ID.String() {
			if i+limit > len(feedSlice) {
				return feedSlice[i:], nil
			}
			return feedSlice[i : i+limit], nil
		}
	}

	return nil, errors.New("No more posts")
}

// получение публичных постов с учётом курсора
func (s *postService) getCursoredPublicPosts(ctx context.Context, params FeedParams) ([]models.Post, error) {
	var feedSlice []models.Post

	allPosts, err := s.PostRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, p := range allPosts {
		if p.IsPublicDemo {
			feedSlice = append(feedSlice, p)
		}
	}

	slices.SortFunc(feedSlice, func(a, b models.Post) int {
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		} else if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		return 0
	})

	limit := params.Limit + 1

	if params.Cursor == nil {
		if limit > len(feedSlice) {
			return feedSlice[:], nil
		} else {
			return feedSlice[:limit], nil
		}
	}

	for i, p := range feedSlice {
		if p.CreatedAt.After(params.Cursor.CreatedAt) && p.Uid.String() != params.Cursor.ID.String() {
			if i+limit > len(feedSlice) {
				return feedSlice[i:], nil
			}
			return feedSlice[i : i+limit], nil
		}
	}

	return nil, errors.New("No more posts")
}
