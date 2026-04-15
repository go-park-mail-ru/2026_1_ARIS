package like

//go:generate mockgen -destination=./../mocks/like_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/like LikeRepo

import (
	"context"
	"fmt"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type likeStorage struct {
	db likeDB
	// logger
}

type LikeRepo interface {
	Get(ctx context.Context, likeID int64) (*models.Like, error)
	Save(ctx context.Context, like models.Like) (int64, error)
	GetLikeCountOnPost(ctx context.Context, postID int64) int
}

type likeDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewLikeStorage(db likeDB) LikeRepo {
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
		fmt.Println("Error : ", err)
		return nil, err
	}

	return &like, nil
}

func (storage *likeStorage) GetLikeCountOnPost(ctx context.Context, postID int64) int {
	return 0
}
