package repost

import (
	"context"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RepostRepo interface {
	Save(ctx context.Context, repost models.Repost) (int64, error)
	GetRepostCount(ctx context.Context, postID int64) int
}

type RepostStorage struct {
	db *pgxpool.Pool
	// logger
}

func NewRepostStorage(db *pgxpool.Pool) RepostRepo {
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

type inmemoryRepostRepo struct {
	reposts map[int64]models.Repost
	mu      sync.RWMutex
}

func NewRepostRepo() RepostRepo {
	return &inmemoryRepostRepo{
		reposts: make(map[int64]models.Repost),
	}
}

func (r *inmemoryRepostRepo) Save(ctx context.Context, repost models.Repost) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.reposts[repost.ID]
	if !ok {
		r.reposts[repost.ID] = repost
	}

	return repost.ID, nil
}

func (r *inmemoryRepostRepo) GetRepostCount(ctx context.Context, postID int64) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0

	for _, repost := range r.reposts {
		if repost.PostID == postID {
			count++
		}
	}

	return count
}
