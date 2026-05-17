package model

import (
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
)

type Post struct {
	ID            int64     `db:"id"`
	Uid           uuid.UUID `db:"uid"`
	Text          *string   `db:"post_text"`
	AuthorID      int64     `db:"author_id"`
	CommunityID   *int64    `db:"community_id"`
	IsPublicDemo  bool      `db:"is_public_demo"`
	AllowComments bool      `db:"allow_comments"`
	IsActive      bool      `db:"is_active"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

func NewPost(text *string, authorID int64, isPublicDemo, allowComments bool) *Post {
	now := time.Now()
	return &Post{
		ID:            rand.Int64(),
		Uid:           uuid.New(),
		Text:          text,
		AuthorID:      authorID,
		IsPublicDemo:  isPublicDemo,
		AllowComments: allowComments,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

type PostWithMedia struct {
	PostID  int64 `db:"post_id"`
	MediaID int64 `db:"media_id"`
	Order   int   `db:"sort_order"`
}

type AttachedMedia struct {
	MediaID  int64     `db:"media_id"`
	UID      uuid.UUID `db:"uid"`
	MimeType string    `db:"mime_type"`
	Link     string    `db:"link"`
	AuthorID int64     `db:"author_id"`
	Order    int       `db:"sort_order"`
}

type Comment struct {
	ID              int64     `db:"id"`
	Uid             uuid.UUID `db:"uid"`
	Text            *string   `db:"comment_text"`
	PostID          int64     `db:"post_id"`
	ParentCommentID *int64    `db:"parent_comment_id"`
	StickerID       *int64    `db:"sticker_id"`
	AuthorID        int64     `db:"author_id"`
	IsActive        bool      `db:"is_active"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
	RepliesCount    int       `db:"replies_count"`
}

type Like struct {
	ID        int64     `db:"id"`
	Uid       uuid.UUID `db:"uid"`
	PostID    *int64    `db:"post_id"`
	CommentID *int64    `db:"comment_id"`
	AuthorID  int64     `db:"author_id"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func NewLikeToPost(postID int64, authorID int64) *Like {
	now := time.Now()
	return &Like{
		ID:        rand.Int64(),
		Uid:       uuid.New(),
		PostID:    &postID,
		AuthorID:  authorID,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func NewPostWithMedia(postID, mediaID int64, order int) *PostWithMedia {
	return &PostWithMedia{PostID: postID, MediaID: mediaID, Order: order}
}

func NewComment(text *string, postID int64, parentCommentID *int64, authorID int64) *Comment {
	now := time.Now()
	return &Comment{
		ID:              rand.Int64(),
		Uid:             uuid.New(),
		Text:            text,
		PostID:          postID,
		ParentCommentID: parentCommentID,
		AuthorID:        authorID,
		IsActive:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}
