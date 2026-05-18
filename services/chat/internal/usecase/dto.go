package usecase

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/services/chat/internal/model"
)

type Chat struct {
	ID                        int64
	UID                       string
	Title                     string
	AvatarID                  *int64
	AvatarLink                string
	Type                      model.ChatType
	IsActive                  bool
	InterlocutorProfileID     *int64
	InterlocutorUserAccountID *int64
	IsOnline                  bool
	LastSeenAt                *time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
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
	Sticker         *Sticker
	Media           []Attachment
	Files           []Attachment
	Reactions       []ReactionSummary
	MyReaction      *string
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type MessageInput struct {
	Text            string
	ParentMessageID *int64
	StickerID       *int64
	Media           []AttachmentInput
	Files           []AttachmentInput
}

type AttachmentInput struct {
	MediaID int64
}

type Attachment struct {
	ID       int64
	UID      string
	MimeType string
	URL      string
}

type StickerPack struct {
	ID        int64
	UID       string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Sticker struct {
	ID       int64
	UID      string
	PackID   *int64
	MediaID  *int64
	MimeType *string
	URL      *string
}

type ReactionSummary struct {
	Type  string
	Count int
}
