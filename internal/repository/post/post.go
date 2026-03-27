package post

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostRepo interface {
	Save(ctx context.Context, post models.Post) (int64, error)
	Delete(ctx context.Context, id int64) error

	List(ctx context.Context, offset, limit int) ([]models.Post, error)
	Get(ctx context.Context, id int64) (*models.Post, error)
	GetAll(ctx context.Context) ([]models.Post, error)
}

type postStorage struct {
	db *pgxpool.Pool
	// logger
}

func NewPostStorage(db *pgxpool.Pool) PostRepo {
	return &postStorage{
		db: db,
	}
}

func (storage *postStorage) Save(ctx context.Context, post models.Post) (int64, error) {
	query := `INSERT INTO post (uid, post_text, author_id, is_public_demo) VALUES ($1, $2, $3, $4) RETURNING id`

	row := storage.db.QueryRow(ctx, query, uuid.New(), post.Text, post.AuthorID, post.IsPublicDemo)

	var postID int64

	if err := row.Scan(&postID); err != nil {
		return 0, err
	}

	return postID, nil
}

func (storage *postStorage) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM post WHERE id=$1`

	res, err := storage.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if res.RowsAffected() != 1 {
		return errors.New("DELETE affected not on 1 row")
	}

	return nil
}

func (storage *postStorage) List(ctx context.Context, offset, limit int) ([]models.Post, error) {
	query := `SELECT * FROM post ORDER BY id LIMIT $1 OFFSET $2`

	rows, err := storage.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}

	posts, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Post])
	if err != nil {
		return nil, err
	}

	return posts, nil
}

func (storage *postStorage) Get(ctx context.Context, id int64) (*models.Post, error) {
	query := `SELECT * FROM post WHERE id=$1`

	var post models.Post

	if err := pgxscan.Get(ctx, storage.db, &post, query, id); err != nil {
		return nil, err
	}

	return &post, nil
}

func (storage *postStorage) GetAll(ctx context.Context) ([]models.Post, error) {
	query := `SELECT * FROM post`

	rows, err := storage.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	posts, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Post])
	if err != nil {
		return nil, err
	}

	return posts, nil
}

type inmemoryPostRepo struct {
	mu    sync.RWMutex
	Posts map[int64]models.Post
}

func NewPostRepo() PostRepo {
	repo := inmemoryPostRepo{}
	repo.Posts = make(map[int64]models.Post)
	return &repo
}

func (r *inmemoryPostRepo) Save(ctx context.Context, post models.Post) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.Posts[post.ID]
	if !ok {
		r.Posts[post.ID] = post
	}

	return post.ID, nil
}

func (r *inmemoryPostRepo) Delete(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.Posts[id]

	if !ok {
		return nil //errors.New("post not found")
	}

	delete(r.Posts, id)
	return nil
}

func (r *inmemoryPostRepo) List(ctx context.Context, offset, limit int) ([]models.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if offset >= len(r.Posts) {
		return []models.Post{}, nil
	}

	if offset+limit > len(r.Posts) {
		return slices.Collect(maps.Values(r.Posts))[offset:], nil
	}

	return slices.Collect(maps.Values(r.Posts))[offset:offset:limit], nil
}

func (r *inmemoryPostRepo) Get(ctx context.Context, id int64) (*models.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profile, ok := r.Posts[id]
	if !ok {
		return nil, errors.New("Profile not found")
	}

	return &profile, nil
}

func (r *inmemoryPostRepo) GetAll(ctx context.Context) ([]models.Post, error) {
	return slices.Collect(maps.Values(r.Posts)), nil
}
