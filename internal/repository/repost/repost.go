package repost

//go:generate mockgen -destination=./../mocks/repost_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/repost RepostRepo

import (
	"context"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RepostRepo interface {
	Save(ctx context.Context, repost models.Repost) (int64, error)
	GetRepostCount(ctx context.Context, postID int64) int
}

type RepostStorage struct {
	db repostDB
	// logger
}

type repostDB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewRepostStorage(db repostDB) RepostRepo {
	return &RepostStorage{
		db: db,
	}
}

func (storage *RepostStorage) Save(ctx context.Context, repost models.Repost) (int64, error) {
	query := `INSERT INTO repost (uid, author_id, chat_id, post_id) VALUES ($1, $2, $3, $4) RETURNING id`

	row := storage.db.QueryRow(ctx, query, uuid.New(), repost.AuthorID, repost.ChatID, repost.PostID)

	var repostID int64

	if err := row.Scan(&repostID); err != nil {
		return 0, err
	}

	return repostID, nil
}

func (storage *RepostStorage) GetRepostCount(ctx context.Context, postID int64) int {
	query := `SELECT COUNT(*) FROM repost WHERE post_id=$1`

	row := storage.db.QueryRow(ctx, query, postID)

	var repostCount int64

	if err := row.Scan(&repostCount); err != nil {
		return 0
	}

	return int(repostCount)
}
