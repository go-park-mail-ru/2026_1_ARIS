package message

//go:generate mockgen -destination=./../mocks/message_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/message MessageRepo

import (
	"context"
	"errors"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type MessageRepo interface {
	Save(ctx context.Context, msg *models.Message) error
	GetByID(ctx context.Context, id int64) (*models.Message, error)
	GetByChatID(ctx context.Context, chatID int64, limit, offset int) ([]models.Message, error)
	Update(ctx context.Context, msg *models.Message) error // новый
	Delete(ctx context.Context, id int64) error
}

type messageStorage struct {
	db messageDB
}

type messageDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func (s *messageStorage) Update(ctx context.Context, msg *models.Message) error {
	logger := logger.FromContext(ctx)
	query := `UPDATE message SET message_text=$1, updated_at=$2 WHERE id=$3 AND is_active=true`
	start := time.Now()
	_, err := s.db.Exec(ctx, query, msg.Text, time.Now(), msg.ID)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "updateMessageByID"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	return err
}
func NewMessageStorage(db messageDB) MessageRepo {
	return &messageStorage{db: db}
}

func (s *messageStorage) Save(ctx context.Context, msg *models.Message) error {
	logger := logger.FromContext(ctx)
	query := `INSERT INTO message (uid, message_text, chat_id, author_id) VALUES ($1, $2, $3, $4) RETURNING id`
	var id int64
	start := time.Now()
	err := s.db.QueryRow(ctx, query, msg.Uid, msg.Text, msg.ChatID, msg.AuthorID).Scan(&id)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "saveMessage"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil {
		return err
	}
	msg.ID = id
	return nil
}

func (s *messageStorage) GetByID(ctx context.Context, id int64) (*models.Message, error) {
	logger := logger.FromContext(ctx)
	query := `SELECT id, uid, message_text, parent_message_id, chat_id, author_id, sticker_id, is_active, created_at, updated_at
		FROM message
		WHERE id=$1 AND is_active=true`
	var msg models.Message
	start := time.Now()
	err := pgxscan.Get(ctx, s.db, &msg, query, id)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "getMessageByID"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("message not found")
		}
		return nil, err
	}
	return &msg, nil
}

func (s *messageStorage) GetByChatID(ctx context.Context, chatID int64, limit, offset int) ([]models.Message, error) {
	logger := logger.FromContext(ctx)
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
	start := time.Now()
	err := pgxscan.Select(ctx, s.db, &messages, query, chatID, limit, offset)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "getMessagesByChatID"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	return messages, nil
}

func (s *messageStorage) Delete(ctx context.Context, id int64) error {
	logger := logger.FromContext(ctx)
	query := `UPDATE message SET is_active=false WHERE id=$1`
	start := time.Now()

	_, err := s.db.Exec(ctx, query, id)
	if logger != nil {
		logger.Debug("db query",
			zap.String("query", "deleteMessageByID"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	return err
}
