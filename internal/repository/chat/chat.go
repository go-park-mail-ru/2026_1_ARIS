package chat

//go:generate mockgen -destination=./../mocks/chat_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/chat ChatRepo

import (
	"context"
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type chatStorage struct {
	db chatDB
}

type ChatRepo interface {
	Save(ctx context.Context, chat *models.Chat) error
	GetByID(ctx context.Context, id int64) (*models.Chat, error)
	Delete(ctx context.Context, id int64) error
}

type chatDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func NewChatStorage(db chatDB) ChatRepo {
	return &chatStorage{db: db}
}

func (r *chatStorage) Save(ctx context.Context, chat *models.Chat) error {
	query := `INSERT INTO chat (uid, chat_type, title, avatar_id) VALUES ($1, $2, $3, $4) RETURNING id`
	var id int64
	err := r.db.QueryRow(ctx, query, chat.Uid, chat.Type, chat.Title, chat.AvatarID).Scan(&id)
	if err != nil {
		return err
	}
	chat.ID = id
	return nil
}

func (r *chatStorage) GetByID(ctx context.Context, id int64) (*models.Chat, error) {
	query := `SELECT * FROM chat WHERE id=$1`
	var chat models.Chat
	err := pgxscan.Get(ctx, r.db, &chat, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("chat not found")
		}
		return nil, err
	}
	return &chat, nil
}

func (r *chatStorage) Delete(ctx context.Context, id int64) error {
	query := `UPDATE chat SET is_active=false WHERE id=$1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
