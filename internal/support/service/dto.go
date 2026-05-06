package service

import "github.com/go-park-mail-ru/2026_1_ARIS/internal/models"

type TicketUpdate struct {
	Title       *string
	Description *string
	Category    *models.TicketCategory
}

func (u TicketUpdate) IsEmpty() bool {
	return u.Title == nil && u.Description == nil && u.Category == nil
}

type TicketFilter struct {
	Status          *models.TicketStatus
	Category        *models.TicketCategory
	Line            *int
	AssignedAgentID *int64
}

type MediaRef struct {
	MediaID  int64
	MediaURL string
}

type AttachmentError struct {
	Err error
	Pos int
}

type MediaErrors struct {
	Errs []AttachmentError
}
