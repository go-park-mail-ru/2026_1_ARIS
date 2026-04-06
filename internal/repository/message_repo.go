package repository

import (
	"context"
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageRepo interface {
	Save(ctx context.Context, msg *models.Message) error
	GetByID(ctx context.Context, id int64) (*models.Message, error)
	GetByChatID(ctx context.Context, chatID int64, limit, offset int) ([]models.Message, error)
	Delete(ctx context.Context, id int64) error
}

type messageStorage struct {
	db *pgxpool.Pool
}

func NewMessageStorage(db *pgxpool.Pool) MessageRepo {
	return &messageStorage{db: db}
}

func (s *messageStorage) Save(ctx context.Context, msg *models.Message) error {
	query := `INSERT INTO message (uid, message_text, chat_id, author_id) VALUES ($1, $2, $3, $4) RETURNING id`
	var id int64
	err := s.db.QueryRow(ctx, query, msg.Uid, msg.Text, msg.ChatID, msg.AuthorID).Scan(&id)
	if err != nil {
		return err
	}
	msg.ID = id
	return nil
}

func (s *messageStorage) GetByID(ctx context.Context, id int64) (*models.Message, error) {
	query := `SELECT id, uid, message_text, parent_message_id, chat_id, author_id, sticker_id, is_active, created_at, updated_at
		FROM message
		WHERE id=$1 AND is_active=true`
	var msg models.Message
	err := pgxscan.Get(ctx, s.db, &msg, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("message not found")
		}
		return nil, err
	}
	return &msg, nil
}

func (s *messageStorage) GetByChatID(ctx context.Context, chatID int64, limit, offset int) ([]models.Message, error) {
	query := `SELECT id, uid, message_text, parent_message_id, chat_id, author_id, sticker_id, is_active, created_at, updated_at
		FROM (
			SELECT id, uid, message_text, parent_message_id, chat_id, author_id, sticker_id, is_active, created_at, updated_at
			FROM message
			WHERE chat_id=$1 AND is_active=true
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		) latest_messages
		ORDER BY created_at ASC`
	var messages []models.Message
	err := pgxscan.Select(ctx, s.db, &messages, query, chatID, limit, offset)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return messages, nil
}

func (s *messageStorage) Delete(ctx context.Context, id int64) error {
	query := `UPDATE message SET is_active=false WHERE id=$1`
	_, err := s.db.Exec(ctx, query, id)
	return err
}
