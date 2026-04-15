package like

import (
	"context"
	"errors"
	"sync"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type likeStorage struct {
	db *pgxpool.Pool
	// logger
}

type inmemoryLikeRepo struct {
	mu    sync.RWMutex
	likes map[int64]models.Like
}

type LikeRepo interface {
	Get(ctx context.Context, likeID int64) (*models.Like, error)
	Save(ctx context.Context, like models.Like) (int64, error)
	GetLikeCountOnPost(ctx context.Context, postID int64) int
}

func NewLikeStorage(db *pgxpool.Pool) LikeRepo {
	return &likeStorage{
		db: db,
	}
}

func (storage *likeStorage) Save(ctx context.Context, like models.Like) (int64, error) {
	query := `INSERT INTO like_record (uid, post_id, comment_id, author_id) 
	VALUES ($1, $2, $3, $4)
	RETURNING id;`

	rows := storage.db.QueryRow(ctx, query,
		uuid.New(),
		like.PostID,
		like.CommentID,
		like.AuthorID)

	var likeID int64

	err := rows.Scan(&likeID)
	if err != nil {
		return 0, err
	}

	return likeID, nil
}

func (storage *likeStorage) Get(ctx context.Context, likeID int64) (*models.Like, error) {
	query := `SELECT * FROM like_record WHERE id=$1`

	var like models.Like

	err := pgxscan.Get(ctx, storage.db, &like, query, likeID)
	if err != nil {
		return nil, err
	}

	return &like, nil
}

func (storage *likeStorage) GetLikeCountOnPost(ctx context.Context, postID int64) int {
	return 0
}

func NewLikeRepo() LikeRepo {
	repo := inmemoryLikeRepo{}
	repo.likes = make(map[int64]models.Like)
	return &repo
}

func (r *inmemoryLikeRepo) Get(ctx context.Context, likeID int64) (*models.Like, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	like, ok := r.likes[likeID]
	if !ok {
		return nil, errors.New("Like not found")
	}
	return &like, nil
}

func (r *inmemoryLikeRepo) Save(ctx context.Context, like models.Like) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.likes[like.ID]
	if !ok {
		r.likes[like.ID] = like
	}
	return like.ID, nil
}

func (r *inmemoryLikeRepo) GetLikeCountOnPost(ctx context.Context, postID int64) int {
	var counter int

	for _, l := range r.likes {
		if l.PostID != nil && *l.PostID == postID {
			counter++
		}
	}
	return counter
}
