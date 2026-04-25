package support

//go:generate mockgen -destination=../mocks/support_service_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/service/support TicketService

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	supportrepo "github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/support"
)

var ErrNilTicket = errors.New("support ticket is nil")

type TicketService interface {
	Save(ctx context.Context, ticket *models.SupportTicket) (int64, error)
	GetByID(ctx context.Context, ticketID, profileID int64) (*models.SupportTicket, error)
	GetByProfileID(ctx context.Context, profileID int64) ([]models.SupportTicket, error)
	Update(ctx context.Context, ticketID, profileID int64, upd TicketUpdate) (*models.SupportTicket, error)
}

type TicketUpdate struct {
	Title       *string
	Description *string
	Category    *models.TicketCategory
}

func (u TicketUpdate) IsEmpty() bool {
	return u.Title == nil && u.Description == nil && u.Category == nil
}

type ticketService struct {
	ticketRepo supportrepo.TicketRepository
}

func NewTicketService(ticketRepo supportrepo.TicketRepository) TicketService {
	return &ticketService{ticketRepo: ticketRepo}
}

func (s *ticketService) Save(ctx context.Context, ticket *models.SupportTicket) (int64, error) {
	if ticket == nil {
		return 0, ErrNilTicket
	}

	now := time.Now()
	if ticket.CreatedAt.IsZero() {
		ticket.CreatedAt = now
	}
	ticket.UpdatedAt = now

	ticketID, err := s.ticketRepo.Save(ctx, ticket)
	if err != nil {
		return 0, fmt.Errorf("ticketService.Save: %w", err)
	}

	return ticketID, nil
}

func (s *ticketService) GetByID(ctx context.Context, ticketID, profileID int64) (*models.SupportTicket, error) {
	ticket, err := s.ticketRepo.GetByIDAndProfileID(ctx, ticketID, profileID)
	if err != nil {
		return nil, fmt.Errorf("ticketService.GetByID: %w", err)
	}

	return ticket, nil
}

func (s *ticketService) GetByProfileID(ctx context.Context, profileID int64) ([]models.SupportTicket, error) {
	tickets, err := s.ticketRepo.GetByProfileID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("ticketService.GetByProfileID: %w", err)
	}

	return tickets, nil
}

func (s *ticketService) Update(ctx context.Context, ticketID, profileID int64, upd TicketUpdate) (*models.SupportTicket, error) {
	if upd.IsEmpty() {
		return nil, errors.New("no ticket fields provided for update")
	}

	ticket, err := s.ticketRepo.GetByIDAndProfileID(ctx, ticketID, profileID)
	if err != nil {
		return nil, fmt.Errorf("ticketService.Update get ticket: %w", err)
	}

	if upd.Title != nil {
		ticket.Title = *upd.Title
	}
	if upd.Description != nil {
		ticket.Description = *upd.Description
	}
	if upd.Category != nil {
		ticket.Category = *upd.Category
	}
	ticket.UpdatedAt = time.Now()

	updatedTicket, err := s.ticketRepo.Update(ctx, ticket)
	if err != nil {
		return nil, fmt.Errorf("ticketService.Update save ticket: %w", err)
	}

	return updatedTicket, nil
}
