package model

import (
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
)

type ChatType string

const (
	PrivateChat ChatType = "personal"
	GroupChat   ChatType = "community"
)

type Chat struct {
	ID        int64     `db:"id"`
	Uid       uuid.UUID `db:"uid"`
	Type      ChatType  `db:"chat_type"`
	Title     string    `db:"title"`
	AvatarID  *int64    `db:"avatar_id,omitempty"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func NewChat(chatType ChatType, title string, avatarID *int64) *Chat {
	now := time.Now()
	return &Chat{
		ID:        rand.Int64(),
		Uid:       uuid.New(),
		Type:      chatType,
		Title:     title,
		AvatarID:  avatarID,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

type ChatMember struct {
	ID        int64      `db:"id"`
	Uid       uuid.UUID  `db:"uid"`
	ChatID    int64      `db:"chat_id"`
	MemberID  int64      `db:"profile_id"`
	JoinedAt  time.Time  `db:"joined_at"`
	IsActive  bool       `db:"is_active"`
	LeaveAt   *time.Time `db:"leave_at"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	Role      string     `db:"chat_role"`
}

type Message struct {
	ID              int64     `db:"id"`
	Uid             uuid.UUID `db:"uid"`
	Text            *string   `db:"message_text"`
	ParentMessageID *int64    `db:"parent_message_id"`
	ChatID          int64     `db:"chat_id"`
	AuthorID        int64     `db:"author_id"`
	StickerID       *int64    `db:"sticker_id"`
	IsActive        bool      `db:"is_active"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

type MessageMedia struct {
	MessageID int64     `db:"message_id"`
	MediaID   int64     `db:"media_id"`
	Order     int       `db:"sort_order"`
	MediaUID  uuid.UUID `db:"media_uid"`
	Name      string    `db:"media_name"`
	MimeType  string    `db:"mime_type"`
	Link      string    `db:"link"`
}

type StickerPack struct {
	ID        int64     `db:"id"`
	Uid       uuid.UUID `db:"uid"`
	Title     string    `db:"title"`
	AuthorID  *int64    `db:"author_id"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Sticker struct {
	ID        int64      `db:"id"`
	Uid       uuid.UUID  `db:"uid"`
	Size      int64      `db:"size"`
	Order     int        `db:"sort_order"`
	PackID    *int64     `db:"pack_id"`
	MediaID   *int64     `db:"media_id"`
	MediaUID  *uuid.UUID `db:"media_uid"`
	MimeType  *string    `db:"mime_type"`
	Link      *string    `db:"link"`
	IsActive  bool       `db:"is_active"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
}

type MediaInfo struct {
	ID       int64  `db:"id"`
	AuthorID int64  `db:"author_id"`
	MimeType string `db:"mime_type"`
	Link     string `db:"link"`
	Size     int64  `db:"size"`
}

type Reaction struct {
	ID        int64     `db:"id"`
	Uid       uuid.UUID `db:"uid"`
	MessageID int64     `db:"message_id"`
	Type      string    `db:"reaction_type"`
	AuthorID  int64     `db:"author_id"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type ReactionSummary struct {
	Type  string `db:"reaction_type"`
	Count int    `db:"count"`
}
