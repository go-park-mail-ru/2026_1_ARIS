package post

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	pgerrors "github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/pg_errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostRepo interface {
	Save(ctx context.Context, post models.Post) (int64, error)
	Delete(ctx context.Context, id int64) error
	Update(ctx context.Context, post models.Post) error

	GetByAuthorID(ctx context.Context, authorID int64) ([]models.Post, error)
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
	query := `INSERT INTO post (uid, post_text, author_id, is_public_demo, allow_comments) VALUES ($1, $2, $3, $4, $5) RETURNING id`

	row := storage.db.QueryRow(ctx, query, uuid.New(), post.Text, post.AuthorID, post.IsPublicDemo, post.AllowComments)

	var postID int64

	if err := row.Scan(&postID); err != nil {
		return 0, pgerrors.MapPgError(err)
	}

	return postID, nil
}

func (storage *postStorage) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM post WHERE id=$1`

	res, err := storage.db.Exec(ctx, query, id)
	if err != nil {
		// подумать...
		return err
	}

	if res.RowsAffected() == 0 {
		return xerrors.PostNotFound
	}

	if res.RowsAffected() > 1 {
		return xerrors.MultipleRowsAffect
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
		if pgxscan.NotFound(err) {
			return nil, xerrors.PostNotFound
		}
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

func (storage *postStorage) GetByAuthorID(ctx context.Context, authorID int64) ([]models.Post, error) {
	query := `SELECT * FROM post WHERE author_id=$1 ORDER BY created_at DESC`

	rows, err := storage.db.Query(ctx, query, authorID)
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

func (storage *postStorage) Update(ctx context.Context, post models.Post) error {
	query := `UPDATE post SET post_text=$1, updated_at=$2 WHERE id=$3`

	res, err := storage.db.Exec(ctx, query, post.Text, post.UpdatedAt, post.ID)
	if err != nil {
		return pgerrors.MapPgError(err)
	}

	if res.RowsAffected() == 0 {
		return xerrors.PostNotFound
	}

	if res.RowsAffected() > 1 {
		return xerrors.MultipleRowsAffect
	}

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

func (r *inmemoryPostRepo) GetByAuthorID(ctx context.Context, authorID int64) ([]models.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	posts := make([]models.Post, 0)

	for _, post := range r.Posts {
		if post.AuthorID == authorID {
			posts = append(posts, post)
		}
	}

	slices.SortFunc(posts, func(a, b models.Post) int {
		if a.CreatedAt.After(b.CreatedAt) {
			return -1
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return 1
		}
		return 0
	})

	return posts, nil
}

func (r *inmemoryPostRepo) Update(ctx context.Context, post models.Post) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.Posts[post.ID]
	if !ok {
		return xerrors.PostNotFound
	}

	r.Posts[post.ID] = post
	return nil
}
