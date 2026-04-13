package post

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostWithMediaRepo interface {
	GetMediaByPostID(ctx context.Context, postID int64) []int64
	Save(ctx context.Context, Tx pgx.Tx, postWithMedia models.PostWithMedia) error
	DeletePostMedia(ctx context.Context, postID int64, Tx pgx.Tx) error
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

func (storage *postWithMediaStorage) DeletePostMedia(ctx context.Context, postID int64, Tx pgx.Tx) error {
	query := `DELETE FROM post_with_media WHERE post_id=$1`

	res, err := Tx.Exec(ctx, query, postID)
	if err != nil {
		return err
	}

	fmt.Println(res.RowsAffected())

	if res.RowsAffected() == 0 {
		return xerrors.NoRowsAffected
	}

	return nil
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

func (storage *postWithMediaStorage) Save(ctx context.Context, Tx pgx.Tx, postWithMedia models.PostWithMedia) error {
	query := `INSERT INTO post_with_media (post_id, media_id, sort_order) VALUES ($1, $2, $3)`

	res, err := Tx.Exec(ctx, query, postWithMedia.PostID, postWithMedia.MediaID, postWithMedia.Order)
	if err != nil {
		return err
	}

	if res.RowsAffected() != 1 {
		return errors.New("affected not on 1 row")
	}
	return nil
}
