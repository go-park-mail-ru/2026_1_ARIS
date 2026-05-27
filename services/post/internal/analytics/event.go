package analytics

import "time"

type EventType string

const (
	EventPostCreated EventType = "post_created"
	EventPostUpdated EventType = "post_updated"
	EventPostDeleted EventType = "post_deleted"
	EventPostView    EventType = "post_view"
	EventPostLike    EventType = "post_like"
	EventPostUnlike  EventType = "post_unlike"
	EventPostComment EventType = "post_comment"
	EventPostRepost  EventType = "post_repost"
	EventPostHide    EventType = "post_hide"
	EventPostReport  EventType = "post_report"
)

type PostEvent struct {
	EventTime       time.Time
	ProfileID       int64
	PostID          int64
	AuthorProfileID int64
	CommunityID     *int64
	Type            EventType
	Source          string
	DwellMs         uint32
	Position        uint16
}

type PostSnapshot struct {
	PostID          int64
	AuthorProfileID int64
	CommunityID     *int64
	IsPublicDemo    bool
	IsActive        bool
	AllowComments   bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
	TextLength      uint16
	HasMedia        bool
}
