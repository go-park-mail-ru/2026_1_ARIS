package comment

import (
	"context"
	"errors"
	"sync"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type inmemoryCommentRepo struct {
	mu       sync.RWMutex
	comments map[int64]models.Comment
}

type commentStorage struct {
	db *pgxpool.Pool
	// logger
}

func NewCommentStorage(db *pgxpool.Pool) CommentRepo {
	return &commentStorage{
		db: db,
	}
}

type CommentRepo interface {
	GetCommentCount(ctx context.Context, postID int64) int
	Save(ctx context.Context, comment models.Comment) (int64, error)
}

func (storage *commentStorage) GetCommentCount(ctx context.Context, postID int64) int {
	query := `SELECT COUNT(*) FROM comment WHERE post_id=$1 AND is_active=true;`

	comments := storage.db.QueryRow(ctx, query, postID)

	var commentCount int64
	if err := comments.Scan(&commentCount); err != nil {
		return 0
	}
	return int(commentCount)
}

func (storage *commentStorage) Save(ctx context.Context, comment models.Comment) (int64, error) {
	query := `INSERT INTO comment (uid, comment_text, post_id, parent_comment_id, sticker_id, author_id) VALUES
	($1, $2, $3, $4, $5, $6)
	RETURNING id;`

	rows, err := storage.db.Query(ctx, query,
		uuid.New(),
		comment.Text,
		comment.TargetPostID,
		comment.ParentCommentID,
		comment.StickerID,
		comment.AuthorID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	} else {
		if err = rows.Err(); err != nil {
			return 0, err
		}
	}
	return 0, errors.New("Bad query")
}

func NewCommentRepo() CommentRepo {
	repo := inmemoryCommentRepo{}
	repo.comments = make(map[int64]models.Comment)
	return &repo
}

func (r *inmemoryCommentRepo) GetCommentCount(ctx context.Context, postID int64) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	commentsCount := 0

	for _, c := range r.comments {
		if c.TargetPostID == postID {
			commentsCount++
		}
	}

	return commentsCount
}

func (r *inmemoryCommentRepo) Save(ctx context.Context, comment models.Comment) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.comments[comment.ID]
	if !ok {
		r.comments[comment.ID] = comment
	}
	return comment.ID, nil
}
