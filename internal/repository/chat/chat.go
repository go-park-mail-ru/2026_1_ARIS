package chat

import (
	"context"
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SQLChatRepo struct {
	db *pgxpool.Pool
}

func NewSQLChatRepo(db *pgxpool.Pool) *SQLChatRepo {
	return &SQLChatRepo{db: db}
}

func (r *SQLChatRepo) Save(ctx context.Context, chat *models.Chat) error {
	query := `INSERT INTO chat (uid, chat_type, title, avatar_id) VALUES ($1, $2, $3, $4) RETURNING id`
	var id int64
	err := r.db.QueryRow(ctx, query, chat.Uid, chat.Type, chat.Title, chat.AvatarID).Scan(&id)
	if err != nil {
		return err
	}
	chat.ID = id
	return nil
}

func (r *SQLChatRepo) GetByID(ctx context.Context, id int64) (*models.Chat, error) {
	query := `SELECT id, uid, chat_type, title, avatar_id, is_active, created_at, updated_at FROM chat WHERE id=$1 AND is_active=true`
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

func (r *SQLChatRepo) Delete(ctx context.Context, id int64) error {
	query := `UPDATE chat SET is_active=false WHERE id=$1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
