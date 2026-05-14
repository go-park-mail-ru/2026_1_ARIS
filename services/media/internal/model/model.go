package model

import (
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
)

type Media struct {
	ID          int64     `db:"id"`
	Uid         uuid.UUID `db:"uid"`
	Name        string    `db:"media_name"`
	AuthorID    int64     `db:"author_id"`
	Extension   string    `db:"extension"`
	Description *string   `db:"description"`
	MimeType    string    `db:"mime_type"`
	Link        string    `db:"link"`
	Size        int       `db:"size"`
	IsActive    bool      `db:"is_active"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func NewMedia(name, extension string, uid uuid.UUID, description *string, mimeType, link string, authorID int64) *Media {
	now := time.Now()
	return &Media{
		ID:          rand.Int64(),
		Uid:         uid,
		Name:        name,
		Extension:   extension,
		Description: description,
		MimeType:    mimeType,
		Link:        link,
		Size:        0,
		AuthorID:    authorID,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
