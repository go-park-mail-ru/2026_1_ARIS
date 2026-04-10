package media

import (
	"context"
	"errors"
	"sync"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MediaRepo interface {
	Get(ctx context.Context, id int64) (*models.Media, error)
	Save(ctx context.Context, media models.Media) (int64, error)
	GetLink(ctx context.Context, id int64) (string, error)
	UpdateLink(ctx context.Context, id int64, newLink string) error
}

type mediaStorage struct {
	db *pgxpool.Pool
	// logger
}

func NewMediaStorage(db *pgxpool.Pool) MediaRepo {
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

// func (storage *mediaStorage) GetBatch(ctx context.Context, ids []int64) error {
// 	query := `SELECT id, uid, mime_type, link
// 	FROM media WHERE id=ANY($1)`
// }

type inmemoryMediaRepo struct {
	mu     sync.RWMutex
	medias map[int64]models.Media
}

func NewMediaRepo() MediaRepo {
	repo := inmemoryMediaRepo{}
	repo.medias = make(map[int64]models.Media)
	return &repo
}

func (r *inmemoryMediaRepo) Get(ctx context.Context, id int64) (*models.Media, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	media, ok := r.medias[id]
	if !ok {
		return nil, errors.New("Media not found")
	}

	return &media, nil
}

func (r *inmemoryMediaRepo) Save(ctx context.Context, media models.Media) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.medias[media.ID] = media
	return media.ID, nil
}

// заглушки
func (r *inmemoryMediaRepo) GetLink(ctx context.Context, id int64) (string, error) {
	return "", nil
}

func (r *inmemoryMediaRepo) UpdateLink(ctx context.Context, id int64, newLink string) error {
	return nil
}
