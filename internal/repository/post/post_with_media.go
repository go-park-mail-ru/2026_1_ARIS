package post

import (
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type inmemoryPostWithMediaRepo struct {
	mu             sync.RWMutex
	postWithMedias []models.PostWithMedia
}

type PostWithMediaRepo interface {
	GetMediaByPostID(ctx context.Context, postID int64) []int64
	Save(ctx context.Context, postWithMedia models.PostWithMedia) error
	DeleteByPostID(ctx context.Context, postID int64) error
}

type postWithMediaStorage struct {
	db *pgxpool.Pool
	// logger
}

func NewPostWithMediaStorage(db *pgxpool.Pool) PostWithMediaRepo {
	return &postWithMediaStorage{
		db: db,
	}
}

func (storage *postWithMediaStorage) GetMediaByPostID(ctx context.Context, postID int64) []int64 {
	query := `SELECT media_id FROM post_with_media WHERE post_id=$1`

	rows, err := storage.db.Query(ctx, query, postID)
	if err != nil {
		return nil
	}

	medias, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (int64, error) {
		var mediaID int64
		err := row.Scan(&mediaID)
		return mediaID, err
	})

	return medias
}

func (storage *postWithMediaStorage) Save(ctx context.Context, postWithMedia models.PostWithMedia) error {
	query := `INSERT INTO post_with_media (post_id, media_id, sort_order) VALUES ($1, $2, $3)`

	res, err := storage.db.Exec(ctx, query, postWithMedia.PostID, postWithMedia.MediaID, postWithMedia.Order)
	if err != nil {
		return err
	}

	if res.RowsAffected() != 1 {
		return errors.New("affected not on 1 row")
	}
	return nil
}

func (storage *postWithMediaStorage) DeleteByPostID(ctx context.Context, postID int64) error {
	query := `DELETE FROM post_with_media WHERE post_id=$1`

	_, err := storage.db.Exec(ctx, query, postID)
	return err
}

func NewPostWithMediaRepo() PostWithMediaRepo {
	return &inmemoryPostWithMediaRepo{}
}

// убрать отсюда, переложить в сервис
func (r *inmemoryPostWithMediaRepo) GetMediaByPostID(ctx context.Context, postID int64) []int64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	var mediaIDs []int64

	slices.SortFunc(r.postWithMedias, func(i, j models.PostWithMedia) int {
		if i.Order < j.Order {
			return -1
		} else if i.Order > j.Order {
			return 1
		}
		return 0
	})

	for _, p := range r.postWithMedias {
		if p.PostID == postID {
			mediaIDs = append(mediaIDs, p.MediaID)
		}
	}

	return mediaIDs
}

func (r *inmemoryPostWithMediaRepo) Save(ctx context.Context, postWithMedia models.PostWithMedia) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.postWithMedias = append(r.postWithMedias, postWithMedia)
	return nil
}

func (r *inmemoryPostWithMediaRepo) DeleteByPostID(ctx context.Context, postID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	filtered := r.postWithMedias[:0]
	for _, item := range r.postWithMedias {
		if item.PostID != postID {
			filtered = append(filtered, item)
		}
	}

	r.postWithMedias = filtered
	return nil
}
