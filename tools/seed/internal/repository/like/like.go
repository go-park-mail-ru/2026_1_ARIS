package like

//go:generate mockgen -destination=./../mocks/like_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/repository/like LikeRepo

import (
	"context"
	"errors"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type likeStorage struct {
	db likeDB
	// logger
}

type LikeRepo interface {
	Get(ctx context.Context, likeID int64) (*models.Like, error)
	Save(ctx context.Context, like models.Like) (int64, error)
	GetLikeCountOnPost(ctx context.Context, postID int64) int
	GetPostLikeByAuthor(ctx context.Context, postID, authorID int64) (*models.Like, error)
	SetActive(ctx context.Context, likeID int64, active bool) error
	HasActivePostLike(ctx context.Context, postID, authorID int64) bool
}

type likeDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func NewLikeStorage(db likeDB) LikeRepo {
	return &likeStorage{
		db: db,
	}
}

func (storage *likeStorage) Save(ctx context.Context, like models.Like) (int64, error) {
	logger := logger.FromContext(ctx)
	query := `INSERT INTO like_record (uid, post_id, comment_id, author_id) 
	VALUES ($1, $2, $3, $4)
	RETURNING id;`

	start := time.Now()
	rows := storage.db.QueryRow(ctx, query,
		uuid.New(),
		like.PostID,
		like.CommentID,
		like.AuthorID)

	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "GetUserByID"),
			zap.Duration("duration_ms", time.Since(start)),
		)
	}

	var likeID int64

	err := rows.Scan(&likeID)
	if err != nil {
		return 0, err
	}

	return likeID, nil
}

func (storage *likeStorage) Get(ctx context.Context, likeID int64) (*models.Like, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM like_record WHERE id=$1`

	var like models.Like

	start := time.Now()
	err := pgxscan.Get(ctx, storage.db, &like, query, likeID)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "GetUserByID"),
			zap.Duration("duration_ms", time.Since(start)),
		)
	}
	if err != nil {
		return nil, err
	}

	return &like, nil
}

func (storage *likeStorage) GetLikeCountOnPost(ctx context.Context, postID int64) int {
	logger := logger.FromContext(ctx)
	query := `SELECT COUNT(*) FROM like_record WHERE post_id=$1 AND is_active=TRUE`

	start := time.Now()
	row := storage.db.QueryRow(ctx, query, postID)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "LikeStorage.GetLikeCountOnPost"),
			zap.Duration("duration_ms", time.Since(start)),
		)
	}

	var count int64
	if err := row.Scan(&count); err != nil {
		return 0
	}
	return int(count)
}

func (storage *likeStorage) GetPostLikeByAuthor(ctx context.Context, postID, authorID int64) (*models.Like, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM like_record WHERE post_id=$1 AND author_id=$2`

	var like models.Like
	start := time.Now()
	err := pgxscan.Get(ctx, storage.db, &like, query, postID, authorID)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "LikeStorage.GetPostLikeByAuthor"),
			zap.Duration("duration_ms", time.Since(start)),
		)
	}
	if err != nil {
		return nil, err
	}
	return &like, nil
}

func (storage *likeStorage) SetActive(ctx context.Context, likeID int64, active bool) error {
	logger := logger.FromContext(ctx)
	query := `UPDATE like_record SET is_active=$1, updated_at=NOW() WHERE id=$2`

	start := time.Now()
	tag, err := storage.db.Exec(ctx, query, active, likeID)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "LikeStorage.SetActive"),
			zap.Duration("duration_ms", time.Since(start)),
		)
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("like not found")
	}
	return nil
}

func (storage *likeStorage) HasActivePostLike(ctx context.Context, postID, authorID int64) bool {
	like, err := storage.GetPostLikeByAuthor(ctx, postID, authorID)
	return err == nil && like.IsActive
}
