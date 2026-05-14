package repository

import (
	"context"
	"errors"
	"html"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type DB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Store struct {
	Chats       ChatRepo
	ChatMembers ChatMemberRepo
	Messages    MessageRepo
}

func NewStore(db DB) Store {
	return Store{
		Chats:       NewChatStorage(db),
		ChatMembers: NewChatMemberStorage(db),
		Messages:    NewMessageStorage(db),
	}
}

type ChatRepo interface {
	Save(ctx context.Context, chat *model.Chat) error
	GetByID(ctx context.Context, id int64) (*model.Chat, error)
	Delete(ctx context.Context, id int64) error
}

type chatStorage struct {
	db DB
}

func NewChatStorage(db DB) ChatRepo {
	return &chatStorage{db: db}
}

func (r *chatStorage) Save(ctx context.Context, chat *model.Chat) error {
	start := time.Now()
	query := `INSERT INTO chat (uid, chat_type, title, avatar_id) VALUES ($1, $2, $3, $4) RETURNING id`
	var id int64
	err := r.db.QueryRow(ctx, query, chat.Uid, chat.Type, chat.Title, chat.AvatarID).Scan(&id)
	logQuery(ctx, "chatStorage.Save", start)
	if err != nil {
		return err
	}
	chat.ID = id
	return nil
}

func (r *chatStorage) GetByID(ctx context.Context, id int64) (*model.Chat, error) {
	start := time.Now()
	var chat model.Chat
	err := pgxscan.Get(ctx, r.db, &chat, `SELECT * FROM chat WHERE id=$1`, id)
	logQuery(ctx, "chatStorage.GetByID", start)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, errors.New("chat not found")
		}
		return nil, err
	}
	return &chat, nil
}

func (r *chatStorage) Delete(ctx context.Context, id int64) error {
	start := time.Now()
	_, err := r.db.Exec(ctx, `UPDATE chat SET is_active=false WHERE id=$1`, id)
	logQuery(ctx, "chatStorage.Delete", start)
	return err
}

type ChatMemberRepo interface {
	Save(ctx context.Context, member model.ChatMember) error
	GetByChatID(ctx context.Context, chatID int64) ([]model.ChatMember, error)
	GetByUserID(ctx context.Context, userID int64) ([]model.ChatMember, error)
	Delete(ctx context.Context, id int64) error
}

type chatMemberStorage struct {
	db DB
}

func NewChatMemberStorage(db DB) ChatMemberRepo {
	return &chatMemberStorage{db: db}
}

func (s *chatMemberStorage) Save(ctx context.Context, member model.ChatMember) error {
	start := time.Now()
	_, err := s.db.Exec(ctx, `
		INSERT INTO chat_member (uid, chat_id, profile_id, joined_at, chat_role)
		VALUES ($1, $2, $3, $4, $5)
	`, member.Uid, member.ChatID, member.MemberID, member.JoinedAt, member.Role)
	logQuery(ctx, "chatMemberStorage.Save", start)
	return err
}

func (s *chatMemberStorage) GetByChatID(ctx context.Context, chatID int64) ([]model.ChatMember, error) {
	start := time.Now()
	var members []model.ChatMember
	err := pgxscan.Select(ctx, s.db, &members, `SELECT * FROM chat_member WHERE chat_id=$1 AND leave_at IS NULL`, chatID)
	logQuery(ctx, "chatMemberStorage.GetByChatID", start)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return members, nil
}

func (s *chatMemberStorage) GetByUserID(ctx context.Context, userID int64) ([]model.ChatMember, error) {
	start := time.Now()
	var members []model.ChatMember
	err := pgxscan.Select(ctx, s.db, &members, `SELECT * FROM chat_member WHERE profile_id=$1 AND leave_at IS NULL`, userID)
	logQuery(ctx, "chatMemberStorage.GetByUserID", start)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return members, nil
}

func (s *chatMemberStorage) Delete(ctx context.Context, id int64) error {
	start := time.Now()
	_, err := s.db.Exec(ctx, `UPDATE chat_member SET leave_at=NOW(), updated_at=NOW() WHERE id=$1`, id)
	logQuery(ctx, "chatMemberStorage.Delete", start)
	return err
}

type MessageRepo interface {
	Save(ctx context.Context, msg *model.Message) error
	GetByID(ctx context.Context, id int64) (*model.Message, error)
	GetByChatID(ctx context.Context, chatID int64, limit, offset int) ([]model.Message, error)
	GetByChatIDAfter(ctx context.Context, chatID, afterID int64, limit int) ([]model.Message, error)
	Update(ctx context.Context, msg *model.Message) error
	Delete(ctx context.Context, id int64) error
}

type messageStorage struct {
	db DB
}

func NewMessageStorage(db DB) MessageRepo {
	return &messageStorage{db: db}
}

func (s *messageStorage) Save(ctx context.Context, msg *model.Message) error {
	start := time.Now()
	query := `INSERT INTO message (uid, message_text, chat_id, author_id) VALUES ($1, $2, $3, $4) RETURNING id`
	var id int64
	err := s.db.QueryRow(ctx, query, msg.Uid, msg.Text, msg.ChatID, msg.AuthorID).Scan(&id)
	logQuery(ctx, "messageStorage.Save", start)
	if err != nil {
		return err
	}
	msg.ID = id
	return nil
}

func (s *messageStorage) GetByID(ctx context.Context, id int64) (*model.Message, error) {
	start := time.Now()
	var msg model.Message
	err := pgxscan.Get(ctx, s.db, &msg, `
		SELECT id, uid, message_text, parent_message_id, chat_id, author_id, sticker_id, is_active, created_at, updated_at
		FROM message
		WHERE id=$1 AND is_active=true
	`, id)
	logQuery(ctx, "messageStorage.GetByID", start)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, errors.New("message not found")
		}
		return nil, err
	}
	return &msg, nil
}

func (s *messageStorage) GetByChatID(ctx context.Context, chatID int64, limit, offset int) ([]model.Message, error) {
	start := time.Now()
	var messages []model.Message
	err := pgxscan.Select(ctx, s.db, &messages, `
		SELECT id, uid, message_text, parent_message_id, chat_id, author_id, sticker_id, is_active, created_at, updated_at
		FROM (
			SELECT id, uid, message_text, parent_message_id, chat_id, author_id, sticker_id, is_active, created_at, updated_at
			FROM message
			WHERE chat_id=$1 AND is_active=true
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		) latest_messages
		ORDER BY created_at ASC
	`, chatID, limit, offset)
	logQuery(ctx, "messageStorage.GetByChatID", start)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	escapeMessages(messages)
	return messages, nil
}

func (s *messageStorage) GetByChatIDAfter(ctx context.Context, chatID, afterID int64, limit int) ([]model.Message, error) {
	start := time.Now()
	var messages []model.Message
	err := pgxscan.Select(ctx, s.db, &messages, `
		SELECT id, uid, message_text, parent_message_id, chat_id, author_id, sticker_id, is_active, created_at, updated_at
		FROM message
		WHERE chat_id=$1 AND id>$2 AND is_active=true
		ORDER BY id ASC
		LIMIT $3
	`, chatID, afterID, limit)
	logQuery(ctx, "messageStorage.GetByChatIDAfter", start)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	escapeMessages(messages)
	return messages, nil
}

func (s *messageStorage) Update(ctx context.Context, msg *model.Message) error {
	start := time.Now()
	_, err := s.db.Exec(ctx, `UPDATE message SET message_text=$1, updated_at=$2 WHERE id=$3 AND is_active=true`, msg.Text, time.Now(), msg.ID)
	logQuery(ctx, "messageStorage.Update", start)
	return err
}

func (s *messageStorage) Delete(ctx context.Context, id int64) error {
	start := time.Now()
	_, err := s.db.Exec(ctx, `UPDATE message SET is_active=false WHERE id=$1`, id)
	logQuery(ctx, "messageStorage.Delete", start)
	return err
}

func escapeMessages(messages []model.Message) {
	for i := range messages {
		if messages[i].Text != nil {
			text := html.EscapeString(*messages[i].Text)
			messages[i].Text = &text
		}
	}
}

func logQuery(ctx context.Context, query string, start time.Time) {
	logg := logger.FromContext(ctx)
	if logg == nil {
		return
	}
	logg.Debug("db query", zap.String("query", query), zap.Duration("duration_ms", time.Since(start)))
}
