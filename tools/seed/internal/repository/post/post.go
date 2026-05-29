package post

//go:generate mockgen -destination=./../mocks/post_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/repository/post PostRepo

import (
	"context"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/models/xerrors"
	pgerrors "github.com/go-park-mail-ru/2026_1_ARIS/tools/seed/internal/utils/pg_errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type PostRepo interface {
	Save(ctx context.Context, post models.Post) (int64, error)
	Delete(ctx context.Context, id int64) error
	Update(ctx context.Context, post models.Post) error
	SetOnlyPublicDemo(ctx context.Context, ids []int64) error

	GetByAuthorID(ctx context.Context, authorID int64) ([]models.Post, error)
	GetByCommunityID(ctx context.Context, communityID int64) ([]models.Post, error)
	List(ctx context.Context, offset, limit int) ([]models.Post, error)
	Get(ctx context.Context, id int64) (*models.Post, error)
	GetAll(ctx context.Context) ([]models.Post, error)
}

type postStorage struct {
	db postDB
	// logger
}

type postDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func NewPostStorage(db postDB) PostRepo {
	return &postStorage{
		db: db,
	}
}

func (storage *postStorage) Save(ctx context.Context, post models.Post) (int64, error) {
	logger := logger.FromContext(ctx)
	query := `
		INSERT INTO post (
			uid, post_text, author_id, community_id,
			is_public_demo, allow_comments, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	start := time.Now()
	row := storage.db.QueryRow(ctx, query,
		post.Uid,
		post.Text,
		post.AuthorID,
		post.CommunityID,
		post.IsPublicDemo,
		post.AllowComments,
		post.CreatedAt,
		post.UpdatedAt,
	)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "PostStorage.Save"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	var postID int64

	if err := row.Scan(&postID); err != nil {
		return 0, pgerrors.MapPgError(err)
	}

	return postID, nil
}

func (storage *postStorage) SetOnlyPublicDemo(ctx context.Context, ids []int64) error {
	logger := logger.FromContext(ctx)
	query := `
		UPDATE post
		SET is_public_demo = (id = ANY($1)),
		    updated_at = CASE WHEN is_public_demo <> (id = ANY($1)) THEN NOW() ELSE updated_at END
	`

	start := time.Now()
	_, err := storage.db.Exec(ctx, query, ids)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "PostStorage.SetOnlyPublicDemo"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil {
		return pgerrors.MapPgError(err)
	}

	return nil
}

func (storage *postStorage) Delete(ctx context.Context, id int64) error {
	logger := logger.FromContext(ctx)
	query := `DELETE FROM post WHERE id=$1`

	start := time.Now()
	res, err := storage.db.Exec(ctx, query, id)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "PostStorage.Delete"),
			zap.Duration("duration_ms", time.Since(start)))
	}
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
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM post ORDER BY id LIMIT $1 OFFSET $2`

	start := time.Now()
	rows, err := storage.db.Query(ctx, query, limit, offset)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "PostStorage.List"),
			zap.Duration("duration_ms", time.Since(start)))
	}
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
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM post WHERE id=$1`

	var post models.Post

	start := time.Now()
	if err := pgxscan.Get(ctx, storage.db, &post, query, id); err != nil {
		if pgxscan.NotFound(err) {
			return nil, xerrors.PostNotFound
		}
		return nil, err
	}
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "PostStorage.Get"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	return &post, nil
}

func (storage *postStorage) GetAll(ctx context.Context) ([]models.Post, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM post`

	start := time.Now()
	rows, err := storage.db.Query(ctx, query)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "PostStorage.GetAll"),
			zap.Duration("duration_ms", time.Since(start)))
	}
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
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM post WHERE author_id=$1 ORDER BY created_at DESC`

	start := time.Now()
	rows, err := storage.db.Query(ctx, query, authorID)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "PostStorage.GetByAuthorID"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil {
		return nil, err
	}

	posts, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Post])
	if err != nil {
		return nil, err
	}

	return posts, nil
}

func (storage *postStorage) GetByCommunityID(ctx context.Context, communityID int64) ([]models.Post, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT * FROM post WHERE community_id=$1 ORDER BY created_at DESC`

	start := time.Now()
	rows, err := storage.db.Query(ctx, query, communityID)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "PostStorage.GetByCommunityID"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil {
		return nil, err
	}

	posts, err := pgx.CollectRows(rows, pgx.RowToStructByName[models.Post])
	if err != nil {
		return nil, err
	}

	return posts, nil
}

func (storage *postStorage) Update(ctx context.Context, post models.Post) error {
	logger := logger.FromContext(ctx)
	query := `UPDATE post SET post_text=$1, updated_at=$2 WHERE id=$3`

	start := time.Now()
	res, err := storage.db.Exec(ctx, query, post.Text, post.UpdatedAt, post.ID)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "PostStorage.Update"),
			zap.Duration("duration_ms", time.Since(start)))
	}
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
