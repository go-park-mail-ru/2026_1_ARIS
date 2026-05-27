package post

//go:generate mockgen -destination=./../mocks/post_with_media_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/repository/post PostWithMediaRepo

import (
	"context"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type PostWithMediaRepo interface {
	GetMediaByPostID(ctx context.Context, postID int64) []int64
	Save(ctx context.Context, postWithMedia models.PostWithMedia) error
	DeleteByPostID(ctx context.Context, postID int64) error
}

type postWithMediaStorage struct {
	db postWithMediaDB
	// logger
}

type postWithMediaDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func NewPostWithMediaStorage(db postWithMediaDB) PostWithMediaRepo {
	return &postWithMediaStorage{
		db: db,
	}
}

func (storage *postWithMediaStorage) GetMediaByPostID(ctx context.Context, postID int64) []int64 {
	logger := logger.FromContext(ctx)
	query := `SELECT media_id FROM post_with_media WHERE post_id=$1`

	start := time.Now()
	rows, err := storage.db.Query(ctx, query, postID)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "PostWithMediaStorage.GetMediaByPostID"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil {
		return nil
	}

	medias, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (int64, error) {
		var mediaID int64
		err := row.Scan(&mediaID)
		return mediaID, err
	})
	if err != nil {
		return nil
	}

	return medias
}

func (storage *postWithMediaStorage) Save(ctx context.Context, postWithMedia models.PostWithMedia) error {
	logger := logger.FromContext(ctx)
	query := `INSERT INTO post_with_media (post_id, media_id, sort_order) VALUES ($1, $2, $3)`

	start := time.Now()
	res, err := storage.db.Exec(ctx, query, postWithMedia.PostID, postWithMedia.MediaID, postWithMedia.Order)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "PostWithMediaStorage.Save"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil {
		return err
	}

	if res.RowsAffected() != 1 {
		return errors.New("affected not on 1 row")
	}
	return nil
}

func (storage *postWithMediaStorage) DeleteByPostID(ctx context.Context, postID int64) error {
	logger := logger.FromContext(ctx)
	query := `DELETE FROM post_with_media WHERE post_id=$1`

	start := time.Now()
	_, err := storage.db.Exec(ctx, query, postID)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "PostWithMediaStorage.DeleteByPostID"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	return err
}
