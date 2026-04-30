package http

import (
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
)

type MediaRequestData struct {
	MediaID  int64  `json:"mediaID"`
	MediaURL string `json:"mediaURL"`
}

type SupportRequest struct {
	Category    TicketCategoryValue `json:"category"`
	Title       string              `json:"title"`
	Login       string              `json:"login"`
	Email       string              `json:"email"`
	Description string              `json:"description"`
	Medias      *[]MediaRequestData `json:"media"`
}

type SupportResponse struct {
	ID     string             `json:"id"`
	Login  string             `json:"login"`
	Status string             `json:"status"`
	Media  []MediaRequestData `json:"media,omitempty"`
}

type SupportTicketResponse struct {
	ID              string                `json:"id"`
	UID             string                `json:"uid"`
	ProfileID       string                `json:"profileID"`
	Login           string                `json:"login"`
	Email           string                `json:"email"`
	Category        string                `json:"category"`
	Title           string                `json:"title"`
	Description     string                `json:"description"`
	Status          string                `json:"status"`
	Priority        models.TicketPriority `json:"priority"`
	Line            int                   `json:"line"`
	AssignedAgentID *string               `json:"assignedAgentId"`
	Rating          *int                  `json:"rating"`
	Media           []MediaRequestData    `json:"media"`
	CreatedAt       time.Time             `json:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt"`
	ClosedAt        *time.Time            `json:"closedAt,omitempty"`
}

type SupportTicketListResponse struct {
	Tickets []SupportTicketResponse `json:"tickets"`
}

type SupportUpdateRequest struct {
	Category    *TicketCategoryValue `json:"category"`
	Title       *string              `json:"title"`
	Description *string              `json:"description"`
}

type SupportStatusUpdateRequest struct {
	Status *TicketStatusValue `json:"status"`
}

type SupportAssignRequest struct {
	AgentID string `json:"agentId"`
}

type SupportEscalateRequest struct {
	Reason string `json:"reason,omitempty"`
}

type SupportRatingRequest struct {
	Rating int `json:"rating"`
}

type SupportMessageRequest struct {
	Text string `json:"text"`
}

type TicketCategoryValue struct {
	Value models.TicketCategory
}

func (v *TicketCategoryValue) UnmarshalJSON(data []byte) error {
	category, err := parseTicketCategoryJSON(data)
	if err != nil {
		return err
	}
	v.Value = category
	return nil
}

type TicketStatusValue struct {
	Value models.TicketStatus
}

func (v *TicketStatusValue) UnmarshalJSON(data []byte) error {
	status, err := parseTicketStatusJSON(data)
	if err != nil {
		return err
	}
	v.Value = status
	return nil
}

type SupportMessageResponse struct {
	ID         string             `json:"id"`
	TicketID   string             `json:"ticketId"`
	Text       string             `json:"text"`
	AuthorID   string             `json:"authorId"`
	AuthorName string             `json:"authorName"`
	AuthorRole models.SupportRole `json:"authorRole"`
	CreatedAt  time.Time          `json:"createdAt"`
}
