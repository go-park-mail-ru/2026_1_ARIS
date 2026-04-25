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
