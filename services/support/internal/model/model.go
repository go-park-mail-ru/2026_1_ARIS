package model

import (
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
)

type SessionID string

type TicketCategory int

const (
	CategoryBug TicketCategory = iota
	CategoryFeatureRequest
	CotegoryComplaint
	CategoryQuestion
	CategoryOther
)

type TicketStatus int

const (
	TicketStatusOpen TicketStatus = iota
	TicketStatusInProgress
	TicketStatusWaitingUser
	TicketStatusClosed
)

type TicketPriority int

const (
	TicketPriorityLow TicketPriority = iota
	TicketPriorityMedium
	TicketPriorityHigh
)

type SupportRole string

const (
	SupportRoleUser      SupportRole = "user"
	SupportRoleSupportL1 SupportRole = "support_l1"
	SupportRoleSupportL2 SupportRole = "support_l2"
	SupportRoleAdmin     SupportRole = "admin"
)

type SupportProfileRole struct {
	ProfileID int64       `db:"profile_id"`
	Role      SupportRole `db:"role"`
}

type SupportTicket struct {
	ID              int64          `db:"id"`
	Uid             uuid.UUID      `db:"uid"`
	ProfileID       int64          `db:"profile_id"`
	Login           string         `db:"login"`
	Email           string         `db:"email"`
	Category        TicketCategory `db:"category"`
	Title           string         `db:"title"`
	Description     string         `db:"description"`
	Status          TicketStatus   `db:"status"`
	Priority        TicketPriority `db:"priority"`
	Line            int            `db:"line"`
	AssignedAgentID *int64         `db:"assigned_agent_id"`
	Rating          *int           `db:"rating"`
	CreatedAt       time.Time      `db:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"`
	ClosedAt        *time.Time     `db:"closed_at"`
}

func NewSupportTicket(profileID int64, login, email string, category TicketCategory, title, description string) *SupportTicket {
	now := time.Now()
	return &SupportTicket{
		ID:          rand.Int64(),
		Uid:         uuid.New(),
		ProfileID:   profileID,
		Login:       login,
		Email:       email,
		Category:    category,
		Title:       title,
		Description: description,
		Status:      TicketStatusOpen,
		Priority:    TicketPriorityLow,
		Line:        1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

type TicketWithMedia struct {
	TicketID int64 `db:"ticket_id"`
	MediaID  int64 `db:"media_id"`
	Order    int   `db:"sort_order"`
}

func NewTicketWithMedia(ticketID, mediaID int64, order int) *TicketWithMedia {
	return &TicketWithMedia{TicketID: ticketID, MediaID: mediaID, Order: order}
}

type SupportTicketStats struct {
	TotalCount         int64            `json:"total"`
	OpenCount          int64            `json:"open"`
	InProgressCount    int64            `json:"inProgress"`
	WaitingUserCount   int64            `json:"waitingUser"`
	ClosedCount        int64            `json:"closed"`
	ByCategory         map[string]int64 `json:"byCategory"`
	ByLine             map[string]int64 `json:"byLine"`
	AverageRating      *float64         `json:"avgRating"`
	RatingDistribution map[string]int64 `json:"ratingDistribution"`
}

type SupportTicketMessage struct {
	ID         int64       `db:"id"`
	TicketID   int64       `db:"ticket_id"`
	Text       string      `db:"text"`
	AuthorID   int64       `db:"author_id"`
	AuthorRole SupportRole `db:"author_role"`
	CreatedAt  time.Time   `db:"created_at"`
}

type UserAccount struct {
	ID       int64
	Username string
	Email    *string
	IsActive bool
}

type UserProfile struct {
	ProfileID     int64
	UserAccountID int64
	FirstName     string
	LastName      string
	IsActive      bool
}

type Profile struct {
	ID       int64
	IsActive bool
}

type Media struct {
	ID       int64
	MimeType string
	Link     string
}
