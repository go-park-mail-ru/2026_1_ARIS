package chat

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepo interface {
	Save(ctx context.Context, chat models.Chat) (int64, error)
	Get(ctx context.Context, chatID int64) (*models.Chat, error)
}

type chatSorage struct {
	db *pgxpool.Pool
	// logger
}

func NewChatStorage(db *pgxpool.Pool) ChatRepo {
	return &chatSorage{
		db: db,
	}
}

func (storage *chatSorage) Save(ctx context.Context, chat models.Chat) (int64, error) {
	query := `INSERT INTO chat (uid, chat_type, title, avatar_id) VALUES ($1, $2, $3, $4) RETURNING id`

	row := storage.db.QueryRow(ctx, query, uuid.New(), chat.Type, chat.Title, chat.AvatarID)

	var chatID int64

	err := row.Scan(&chatID)
	if err != nil {
		return 0, err
	}

	return chatID, nil
}

func (storage *chatSorage) Get(ctx context.Context, chatID int64) (*models.Chat, error) {
	query := `SELECT * FROM chat WHERE id=$1`

	var chat models.Chat

	err := pgxscan.Get(ctx, storage.db, &chat, query, chatID)
	if err != nil {
		return nil, err
	}

	return &chat, nil
}
