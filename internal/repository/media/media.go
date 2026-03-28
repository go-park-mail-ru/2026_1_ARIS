package media

import (
	"context"
	"errors"
	"sync"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MediaRepo interface {
	Get(ctx context.Context, id int64) (*models.Media, error)
	Save(ctx context.Context, media models.Media) (int64, error)
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
		return nil, err
	}

	return &media, err
}

func (storage *mediaStorage) Save(ctx context.Context, media models.Media) (int64, error) {
	query := `INSERT INTO media (uid, media_name, extension, mime_type, description, size, link) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	row := storage.db.QueryRow(ctx, query, uuid.New(), media.Name, media.Extension, media.MimeType, media.Description, media.Size, media.Link)

	var mediaID int64

	if err := row.Scan(&mediaID); err != nil {
		return 0, err
	}

	return mediaID, nil
}

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
