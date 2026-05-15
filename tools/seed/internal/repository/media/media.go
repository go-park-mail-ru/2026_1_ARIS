package media

//go:generate mockgen -destination=./../mocks/media_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/repository/media MediaRepo

import (
	"context"
	"errors"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/models/xerrors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type MediaRepo interface {
	Get(ctx context.Context, id int64) (*models.Media, error)
	GetIDByName(ctx context.Context, name string) (int64, error)
	Save(ctx context.Context, media models.Media) (int64, error)
	GetLink(ctx context.Context, id int64) (string, error)
	UpdateLink(ctx context.Context, id int64, newLink string) error
}

type mediaStorage struct {
	db mediaDB
}

type mediaDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func NewMediaStorage(db mediaDB) MediaRepo {
	return &mediaStorage{
		db: db,
	}
}

func (storage *mediaStorage) Get(ctx context.Context, id int64) (*models.Media, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM media WHERE id=$1`

	var media models.Media

	start := time.Now()
	err := pgxscan.Get(ctx, storage.db, &media, query, id)

	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "GetMediaByID"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, xerrors.MediaNotFound
		}
		return nil, err
	}

	return &media, err
}

func (storage *mediaStorage) GetIDByName(ctx context.Context, name string) (int64, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT id FROM media WHERE media_name=$1 AND is_active=TRUE ORDER BY id LIMIT 1`

	start := time.Now()
	row := storage.db.QueryRow(ctx, query, name)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "GetMediaIDByName"),
			zap.Duration("duration_ms", time.Since(start)))
	}

	var mediaID int64
	if err := row.Scan(&mediaID); err != nil {
		return 0, err
	}
	return mediaID, nil
}

func (storage *mediaStorage) Save(ctx context.Context, media models.Media) (int64, error) {
	logger := logger.FromContext(ctx)
	query := `INSERT INTO media (uid, media_name, extension, mime_type, size, link, author_id) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	start := time.Now()
	row := storage.db.QueryRow(ctx, query, media.Uid, media.Name, media.Extension, media.MimeType, media.Size, media.Link, media.AuthorID)

	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "SaveMedia"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	var mediaID int64

	if err := row.Scan(&mediaID); err != nil {
		return 0, err
	}

	return mediaID, nil
}

func (storage *mediaStorage) GetLink(ctx context.Context, id int64) (string, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT link FROM media WHERE id=$1`

	start := time.Now()
	row := storage.db.QueryRow(ctx, query, id)

	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "GetMediaLinkByID"),
			zap.Duration("duration_ms", time.Since(start)))
	}

	var link string

	if err := row.Scan(&link); err != nil {
		return "", err
	}

	return link, nil
}

func (storage *mediaStorage) UpdateLink(ctx context.Context, id int64, newLink string) error {
	logger := logger.FromContext(ctx)
	query := `UPDATE table media SET link=$1 WHERE id=$2`

	start := time.Now()
	res, err := storage.db.Exec(ctx, query, newLink, id)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "UpdateMediaLinkByID"),
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
