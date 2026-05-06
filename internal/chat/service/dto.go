package service

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type Chat struct {
	ID        int64
	UID       string
	Title     string
	AvatarID  *int64
	Type      models.ChatType
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Message struct {
	ID              int64
	UID             string
	Text            *string
	AuthorName      string
	ParentMessageID *int64
	ChatID          int64
	AuthorID        int64
	StickerID       *int64
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
