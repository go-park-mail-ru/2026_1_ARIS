package media

//go:generate mockgen -destination=./../mocks/media_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/media MediaRepo

import (
	"context"
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type MediaRepo interface {
	Get(ctx context.Context, id int64) (*models.Media, error)
	Save(ctx context.Context, media models.Media) (int64, error)
	GetLink(ctx context.Context, id int64) (string, error)
	UpdateLink(ctx context.Context, id int64, newLink string) error
}

type mediaStorage struct {
	db mediaDB
	// logger
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
	query := `SELECT * FROM media WHERE id=$1`

	var media models.Media

	err := pgxscan.Get(ctx, storage.db, &media, query, id)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, xerrors.MediaNotFound
		}
		return nil, err
	}

	return &media, err
}

func (storage *mediaStorage) Save(ctx context.Context, media models.Media) (int64, error) {
	query := `INSERT INTO media (uid, media_name, extension, mime_type, size, link, author_id) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	row := storage.db.QueryRow(ctx, query, media.Uid, media.Name, media.Extension, media.MimeType, media.Size, media.Link, media.AuthorID)

	var mediaID int64

	if err := row.Scan(&mediaID); err != nil {
		return 0, err
	}

	return mediaID, nil
}

func (storage *mediaStorage) GetLink(ctx context.Context, id int64) (string, error) {
	query := `SELECT link FROM media WHERE id=$1`

	row := storage.db.QueryRow(ctx, query, id)

	var link string

	if err := row.Scan(&link); err != nil {
		return "", err
	}

	return link, nil
}

func (storage *mediaStorage) UpdateLink(ctx context.Context, id int64, newLink string) error {
	query := `UPDATE table media SET link=$1 WHERE id=$2`

	res, err := storage.db.Exec(ctx, query, newLink, id)
	if err != nil {
		return err
	}

	if res.RowsAffected() != 1 {
		return errors.New("affected not on 1 row")
	}

	return nil
}
