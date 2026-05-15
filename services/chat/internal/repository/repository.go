package repository

import (
	"context"
	"errors"
	"html"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/model"
	"github.com/google/uuid"
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
	Chats        ChatRepo
	ChatMembers  ChatMemberRepo
	Messages     MessageRepo
	MessageMedia MessageMediaRepo
	Stickers     StickerRepo
	Reactions    ReactionRepo
}

func NewStore(db DB) Store {
	return Store{
		Chats:        NewChatStorage(db),
		ChatMembers:  NewChatMemberStorage(db),
		Messages:     NewMessageStorage(db),
		MessageMedia: NewMessageMediaStorage(db),
		Stickers:     NewStickerStorage(db),
		Reactions:    NewReactionStorage(db),
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
	query := `
		INSERT INTO message (uid, message_text, parent_message_id, chat_id, sticker_id, author_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	var id int64
	err := s.db.QueryRow(ctx, query, msg.Uid, msg.Text, msg.ParentMessageID, msg.ChatID, msg.StickerID, msg.AuthorID).Scan(&id)
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

type MessageMediaRepo interface {
	Save(ctx context.Context, item model.MessageMedia) error
	GetByMessageIDs(ctx context.Context, messageIDs []int64) (map[int64][]model.MessageMedia, error)
	GetMediaAuthorID(ctx context.Context, mediaID int64) (int64, error)
	DeleteByMessageID(ctx context.Context, messageID int64) error
}

type messageMediaStorage struct {
	db DB
}

func NewMessageMediaStorage(db DB) MessageMediaRepo {
	return &messageMediaStorage{db: db}
}

func (s *messageMediaStorage) Save(ctx context.Context, item model.MessageMedia) error {
	start := time.Now()
	tag, err := s.db.Exec(ctx, `
		INSERT INTO message_with_media (message_id, media_id, sort_order)
		VALUES ($1, $2, $3)
	`, item.MessageID, item.MediaID, item.Order)
	logQuery(ctx, "messageMediaStorage.Save", start)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("message media not saved")
	}
	return nil
}

func (s *messageMediaStorage) GetByMessageIDs(ctx context.Context, messageIDs []int64) (map[int64][]model.MessageMedia, error) {
	result := make(map[int64][]model.MessageMedia, len(messageIDs))
	if len(messageIDs) == 0 {
		return result, nil
	}
	start := time.Now()
	rows, err := s.db.Query(ctx, `
		SELECT mwm.message_id, mwm.media_id, mwm.sort_order, m.uid AS media_uid, m.mime_type, m.link
		FROM message_with_media mwm
		JOIN media m ON m.id=mwm.media_id AND m.is_active=TRUE
		WHERE mwm.message_id=ANY($1)
		ORDER BY mwm.message_id, mwm.sort_order
	`, messageIDs)
	logQuery(ctx, "messageMediaStorage.GetByMessageIDs", start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item model.MessageMedia
		if err := rows.Scan(&item.MessageID, &item.MediaID, &item.Order, &item.MediaUID, &item.MimeType, &item.Link); err != nil {
			return nil, err
		}
		result[item.MessageID] = append(result[item.MessageID], item)
	}
	return result, rows.Err()
}

func (s *messageMediaStorage) GetMediaAuthorID(ctx context.Context, mediaID int64) (int64, error) {
	start := time.Now()
	row := s.db.QueryRow(ctx, `SELECT author_id FROM media WHERE id=$1 AND is_active=TRUE`, mediaID)
	logQuery(ctx, "messageMediaStorage.GetMediaAuthorID", start)

	var authorID int64
	if err := row.Scan(&authorID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errors.New("media not found")
		}
		return 0, err
	}
	return authorID, nil
}

func (s *messageMediaStorage) DeleteByMessageID(ctx context.Context, messageID int64) error {
	start := time.Now()
	_, err := s.db.Exec(ctx, `DELETE FROM message_with_media WHERE message_id=$1`, messageID)
	logQuery(ctx, "messageMediaStorage.DeleteByMessageID", start)
	return err
}

type StickerRepo interface {
	Get(ctx context.Context, id int64) (*model.Sticker, error)
	ListPacks(ctx context.Context, limit, offset int) ([]model.StickerPack, error)
	ListByPackID(ctx context.Context, packID int64, limit, offset int) ([]model.Sticker, error)
}

type stickerStorage struct {
	db DB
}

func NewStickerStorage(db DB) StickerRepo {
	return &stickerStorage{db: db}
}

func (s *stickerStorage) Get(ctx context.Context, id int64) (*model.Sticker, error) {
	start := time.Now()
	var sticker model.Sticker
	err := pgxscan.Get(ctx, s.db, &sticker, `
		SELECT s.id, s.uid, s.size, s.sort_order, s.pack_id, s.media_id,
		       m.uid AS media_uid, m.mime_type, m.link,
		       s.is_active, s.created_at, s.updated_at
		FROM sticker s
		LEFT JOIN media m ON m.id=s.media_id AND m.is_active=TRUE
		WHERE s.id=$1 AND s.is_active=TRUE
	`, id)
	logQuery(ctx, "stickerStorage.Get", start)
	if err != nil {
		if pgxscan.NotFound(err) {
			return nil, errors.New("sticker not found")
		}
		return nil, err
	}
	return &sticker, nil
}

func (s *stickerStorage) ListPacks(ctx context.Context, limit, offset int) ([]model.StickerPack, error) {
	start := time.Now()
	var packs []model.StickerPack
	err := pgxscan.Select(ctx, s.db, &packs, `
		SELECT *
		FROM sticker_pack
		WHERE is_active=TRUE
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	logQuery(ctx, "stickerStorage.ListPacks", start)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return packs, nil
}

func (s *stickerStorage) ListByPackID(ctx context.Context, packID int64, limit, offset int) ([]model.Sticker, error) {
	start := time.Now()
	var stickers []model.Sticker
	err := pgxscan.Select(ctx, s.db, &stickers, `
		SELECT s.id, s.uid, s.size, s.sort_order, s.pack_id, s.media_id,
		       m.uid AS media_uid, m.mime_type, m.link,
		       s.is_active, s.created_at, s.updated_at
		FROM sticker s
		LEFT JOIN media m ON m.id=s.media_id AND m.is_active=TRUE
		WHERE s.pack_id=$1 AND s.is_active=TRUE
		ORDER BY s.sort_order ASC, s.id ASC
		LIMIT $2 OFFSET $3
	`, packID, limit, offset)
	logQuery(ctx, "stickerStorage.ListByPackID", start)
	if err != nil && !pgxscan.NotFound(err) {
		return nil, err
	}
	return stickers, nil
}

type ReactionRepo interface {
	Upsert(ctx context.Context, messageID, authorID int64, reactionType string) error
	Delete(ctx context.Context, messageID, authorID int64) error
	GetSummaryByMessageIDs(ctx context.Context, messageIDs []int64) (map[int64][]model.ReactionSummary, error)
	GetUserReactionsByMessageIDs(ctx context.Context, messageIDs []int64, authorID int64) (map[int64]string, error)
}

type reactionStorage struct {
	db DB
}

func NewReactionStorage(db DB) ReactionRepo {
	return &reactionStorage{db: db}
}

func (s *reactionStorage) Upsert(ctx context.Context, messageID, authorID int64, reactionType string) error {
	start := time.Now()
	_, err := s.db.Exec(ctx, `
		INSERT INTO reaction (uid, message_id, author_id, reaction_type, is_active)
		VALUES ($4, $1, $2, $3::reaction_type, TRUE)
		ON CONFLICT (message_id, author_id)
		DO UPDATE SET reaction_type=$3::reaction_type, is_active=TRUE, updated_at=NOW()
	`, messageID, authorID, reactionType, uuid.New())
	logQuery(ctx, "reactionStorage.Upsert", start)
	return err
}

func (s *reactionStorage) Delete(ctx context.Context, messageID, authorID int64) error {
	start := time.Now()
	_, err := s.db.Exec(ctx, `UPDATE reaction SET is_active=FALSE, updated_at=NOW() WHERE message_id=$1 AND author_id=$2`, messageID, authorID)
	logQuery(ctx, "reactionStorage.Delete", start)
	return err
}

func (s *reactionStorage) GetSummaryByMessageIDs(ctx context.Context, messageIDs []int64) (map[int64][]model.ReactionSummary, error) {
	result := make(map[int64][]model.ReactionSummary, len(messageIDs))
	if len(messageIDs) == 0 {
		return result, nil
	}
	start := time.Now()
	rows, err := s.db.Query(ctx, `
		SELECT message_id, reaction_type::text, COUNT(*)::int
		FROM reaction
		WHERE message_id=ANY($1) AND is_active=TRUE
		GROUP BY message_id, reaction_type
		ORDER BY message_id, reaction_type
	`, messageIDs)
	logQuery(ctx, "reactionStorage.GetSummaryByMessageIDs", start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var messageID int64
		var summary model.ReactionSummary
		if err := rows.Scan(&messageID, &summary.Type, &summary.Count); err != nil {
			return nil, err
		}
		result[messageID] = append(result[messageID], summary)
	}
	return result, rows.Err()
}

func (s *reactionStorage) GetUserReactionsByMessageIDs(ctx context.Context, messageIDs []int64, authorID int64) (map[int64]string, error) {
	result := make(map[int64]string, len(messageIDs))
	if len(messageIDs) == 0 || authorID <= 0 {
		return result, nil
	}
	start := time.Now()
	rows, err := s.db.Query(ctx, `
		SELECT message_id, reaction_type::text
		FROM reaction
		WHERE message_id=ANY($1) AND author_id=$2 AND is_active=TRUE
	`, messageIDs, authorID)
	logQuery(ctx, "reactionStorage.GetUserReactionsByMessageIDs", start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var messageID int64
		var reactionType string
		if err := rows.Scan(&messageID, &reactionType); err != nil {
			return nil, err
		}
		result[messageID] = reactionType
	}
	return result, rows.Err()
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
