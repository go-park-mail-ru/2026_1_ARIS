package repository

import (
	"context"
	"errors"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/post/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

var (
	ErrPostNotFound = errors.New("post not found")
)

type DB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Store struct {
	Posts     PostRepo
	PostMedia PostMediaRepo
	Comments  CommentRepo
	Likes     LikeRepo
	Reposts   RepostRepo
}

func NewStore(db DB) Store {
	return Store{
		Posts:     NewPostStorage(db),
		PostMedia: NewPostMediaStorage(db),
		Comments:  NewCommentStorage(db),
		Likes:     NewLikeStorage(db),
		Reposts:   NewRepostStorage(db),
	}
}

type PostRepo interface {
	Save(ctx context.Context, post model.Post) (int64, error)
	Delete(ctx context.Context, id int64) error
	Update(ctx context.Context, post model.Post) error
	Get(ctx context.Context, id int64) (*model.Post, error)
	GetAll(ctx context.Context) ([]model.Post, error)
	GetByAuthorID(ctx context.Context, authorID int64) ([]model.Post, error)
	GetByCommunityID(ctx context.Context, communityID int64) ([]model.Post, error)
}

type postStorage struct {
	db DB
}

func NewPostStorage(db DB) PostRepo {
	return &postStorage{db: db}
}

func (s *postStorage) Save(ctx context.Context, post model.Post) (int64, error) {
	start := time.Now()
	row := s.db.QueryRow(ctx, `
		INSERT INTO post (uid, post_text, author_id, community_id, is_public_demo, allow_comments)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, uuid.New(), post.Text, post.AuthorID, post.CommunityID, post.IsPublicDemo, post.AllowComments)
	logQuery(ctx, "postStorage.Save", start)

	var postID int64
	if err := row.Scan(&postID); err != nil {
		return 0, err
	}
	return postID, nil
}

func (s *postStorage) Delete(ctx context.Context, id int64) error {
	start := time.Now()
	if _, err := s.db.Exec(ctx, `DELETE FROM like_record WHERE post_id=$1 OR comment_id IN (SELECT id FROM comment WHERE post_id=$1)`, id); err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx, `DELETE FROM post WHERE id=$1`, id)
	logQuery(ctx, "postStorage.Delete", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPostNotFound
	}
	return nil
}

func (s *postStorage) Update(ctx context.Context, post model.Post) error {
	start := time.Now()
	tag, err := s.db.Exec(ctx, `UPDATE post SET post_text=$1, updated_at=$2 WHERE id=$3`, post.Text, post.UpdatedAt, post.ID)
	logQuery(ctx, "postStorage.Update", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPostNotFound
	}
	return nil
}

func (s *postStorage) Get(ctx context.Context, id int64) (*model.Post, error) {
	var post model.Post
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &post, `SELECT * FROM post WHERE id=$1`, id)
	logQuery(ctx, "postStorage.Get", start)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	return &post, nil
}

func (s *postStorage) GetAll(ctx context.Context) ([]model.Post, error) {
	start := time.Now()
	rows, err := s.db.Query(ctx, `SELECT * FROM post WHERE is_active=TRUE`)
	logQuery(ctx, "postStorage.GetAll", start)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Post])
}

func (s *postStorage) GetByAuthorID(ctx context.Context, authorID int64) ([]model.Post, error) {
	start := time.Now()
	rows, err := s.db.Query(ctx, `SELECT * FROM post WHERE author_id=$1 AND is_active=TRUE ORDER BY created_at DESC`, authorID)
	logQuery(ctx, "postStorage.GetByAuthorID", start)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Post])
}

func (s *postStorage) GetByCommunityID(ctx context.Context, communityID int64) ([]model.Post, error) {
	start := time.Now()
	rows, err := s.db.Query(ctx, `SELECT * FROM post WHERE community_id=$1 AND is_active=TRUE ORDER BY created_at DESC`, communityID)
	logQuery(ctx, "postStorage.GetByCommunityID", start)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.Post])
}

type PostMediaRepo interface {
	GetMediaByPostID(ctx context.Context, postID int64) []int64
	Save(ctx context.Context, postWithMedia model.PostWithMedia) error
	DeleteByPostID(ctx context.Context, postID int64) error
}

type postMediaStorage struct {
	db DB
}

func NewPostMediaStorage(db DB) PostMediaRepo {
	return &postMediaStorage{db: db}
}

func (s *postMediaStorage) GetMediaByPostID(ctx context.Context, postID int64) []int64 {
	start := time.Now()
	rows, err := s.db.Query(ctx, `SELECT media_id FROM post_with_media WHERE post_id=$1 ORDER BY sort_order`, postID)
	logQuery(ctx, "postMediaStorage.GetMediaByPostID", start)
	if err != nil {
		return nil
	}
	defer rows.Close()

	mediaIDs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (int64, error) {
		var mediaID int64
		return mediaID, row.Scan(&mediaID)
	})
	if err != nil {
		return nil
	}
	return mediaIDs
}

func (s *postMediaStorage) Save(ctx context.Context, postWithMedia model.PostWithMedia) error {
	start := time.Now()
	tag, err := s.db.Exec(ctx, `INSERT INTO post_with_media (post_id, media_id, sort_order) VALUES ($1, $2, $3)`, postWithMedia.PostID, postWithMedia.MediaID, postWithMedia.Order)
	logQuery(ctx, "postMediaStorage.Save", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("affected not on 1 row")
	}
	return nil
}

func (s *postMediaStorage) DeleteByPostID(ctx context.Context, postID int64) error {
	start := time.Now()
	_, err := s.db.Exec(ctx, `DELETE FROM post_with_media WHERE post_id=$1`, postID)
	logQuery(ctx, "postMediaStorage.DeleteByPostID", start)
	return err
}

type CommentRepo interface {
	GetCommentCount(ctx context.Context, postID int64) int
}

type commentStorage struct {
	db DB
}

func NewCommentStorage(db DB) CommentRepo {
	return &commentStorage{db: db}
}

func (s *commentStorage) GetCommentCount(ctx context.Context, postID int64) int {
	return countByQuery(ctx, s.db, `SELECT COUNT(*) FROM comment WHERE post_id=$1 AND is_active=TRUE`, "commentStorage.GetCommentCount", postID)
}

type LikeRepo interface {
	Save(ctx context.Context, like model.Like) (int64, error)
	GetLikeCountOnPost(ctx context.Context, postID int64) int
	GetPostLikeByAuthor(ctx context.Context, postID, authorID int64) (*model.Like, error)
	SetActive(ctx context.Context, likeID int64, active bool) error
	HasActivePostLike(ctx context.Context, postID, authorID int64) bool
}

type likeStorage struct {
	db DB
}

func NewLikeStorage(db DB) LikeRepo {
	return &likeStorage{db: db}
}

func (s *likeStorage) Save(ctx context.Context, like model.Like) (int64, error) {
	start := time.Now()
	row := s.db.QueryRow(ctx, `
		INSERT INTO like_record (uid, post_id, comment_id, author_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, uuid.New(), like.PostID, like.CommentID, like.AuthorID)
	logQuery(ctx, "likeStorage.Save", start)

	var likeID int64
	if err := row.Scan(&likeID); err != nil {
		return 0, err
	}
	return likeID, nil
}

func (s *likeStorage) GetLikeCountOnPost(ctx context.Context, postID int64) int {
	return countByQuery(ctx, s.db, `SELECT COUNT(*) FROM like_record WHERE post_id=$1 AND is_active=TRUE`, "likeStorage.GetLikeCountOnPost", postID)
}

func (s *likeStorage) GetPostLikeByAuthor(ctx context.Context, postID, authorID int64) (*model.Like, error) {
	var like model.Like
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &like, `SELECT * FROM like_record WHERE post_id=$1 AND author_id=$2`, postID, authorID)
	logQuery(ctx, "likeStorage.GetPostLikeByAuthor", start)
	if err != nil {
		return nil, err
	}
	return &like, nil
}

func (s *likeStorage) SetActive(ctx context.Context, likeID int64, active bool) error {
	start := time.Now()
	tag, err := s.db.Exec(ctx, `UPDATE like_record SET is_active=$1, updated_at=NOW() WHERE id=$2`, active, likeID)
	logQuery(ctx, "likeStorage.SetActive", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("like not found")
	}
	return nil
}

func (s *likeStorage) HasActivePostLike(ctx context.Context, postID, authorID int64) bool {
	like, err := s.GetPostLikeByAuthor(ctx, postID, authorID)
	return err == nil && like.IsActive
}

type RepostRepo interface {
	GetRepostCount(ctx context.Context, postID int64) int
}

type repostStorage struct {
	db DB
}

func NewRepostStorage(db DB) RepostRepo {
	return &repostStorage{db: db}
}

func (s *repostStorage) GetRepostCount(ctx context.Context, postID int64) int {
	return countByQuery(ctx, s.db, `SELECT COUNT(*) FROM repost WHERE post_id=$1 AND is_active=TRUE`, "repostStorage.GetRepostCount", postID)
}

func countByQuery(ctx context.Context, db DB, query string, label string, args ...any) int {
	start := time.Now()
	row := db.QueryRow(ctx, query, args...)
	logQuery(ctx, label, start)

	var count int64
	if err := row.Scan(&count); err != nil {
		return 0
	}
	return int(count)
}

func logQuery(ctx context.Context, query string, start time.Time) {
	logg := logger.FromContext(ctx)
	if logg == nil {
		return
	}
	logg.Debug("db query", zap.String("query", query), zap.Duration("duration_ms", time.Since(start)))
}
