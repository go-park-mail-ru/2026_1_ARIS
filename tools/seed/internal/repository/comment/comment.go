package comment

//go:generate mockgen -destination=./../mocks/comment_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/repository/comment CommentRepo

import (
	"context"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type commentStorage struct {
	db commentDB
	// logger
}

type commentDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewCommentStorage(db commentDB) CommentRepo {
	return &commentStorage{
		db: db,
	}
}

type CommentRepo interface {
	GetCommentCount(ctx context.Context, postID int64) int
	Save(ctx context.Context, comment models.Comment) (int64, error)
}

func (storage *commentStorage) GetCommentCount(ctx context.Context, postID int64) int {
	logger := logger.FromContext(ctx)
	query := `SELECT COUNT(*) FROM comment WHERE post_id=$1;`

	start := time.Now()
	comments := storage.db.QueryRow(ctx, query, postID)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "GetUserByID"),
			zap.Duration("duration_ms", time.Since(start)),
		)
	}

	var commentCount int64
	if err := comments.Scan(&commentCount); err != nil {
		return 0
	}
	return int(commentCount)
}

func (storage *commentStorage) Save(ctx context.Context, comment models.Comment) (int64, error) {
	logger := logger.FromContext(ctx)
	query := `INSERT INTO comment (uid, comment_text, post_id, parent_comment_id, sticker_id, author_id) VALUES
	($1, $2, $3, $4, $5, $6)
	RETURNING id;`

	start := time.Now()
	rows, err := storage.db.Query(ctx, query,
		uuid.New(),
		comment.Text,
		comment.TargetPostID,
		comment.ParentCommentID,
		comment.StickerID,
		comment.AuthorID)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "GetUserByID"),
			zap.Duration("duration_ms", time.Since(start)),
		)
	}
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	} else {
		if err = rows.Err(); err != nil {
			return 0, err
		}
	}
	return 0, errors.New("Bad query")
}
