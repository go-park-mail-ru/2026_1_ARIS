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
