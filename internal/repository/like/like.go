package like

//go:generate mockgen -destination=./../mocks/like_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/like LikeRepo

import (
	"context"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
}

type likeDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
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
	logger.Debug("db query",
		zap.String("query", "GetUserByID"),
		zap.Duration("duration_ms", time.Since(start)),
	)

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
	logger.Debug("db query",
		zap.String("query", "GetUserByID"),
		zap.Duration("duration_ms", time.Since(start)),
	)
	if err != nil {
		return nil, err
	}

	return &like, nil
}

func (storage *likeStorage) GetLikeCountOnPost(ctx context.Context, postID int64) int {
	return 0
}
