package support

//go:generate mockgen -destination=./../mocks/support_ticket_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/support TicketRepository

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	pgerrors "github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/pg_errors"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type TicketRepository interface {
	Save(ctx context.Context, ticket *models.SupportTicket) (int64, error)
}

type ticketStorage struct {
	db ticketDB
}

type ticketDB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewTicketStorage(db ticketDB) TicketRepository {
	return &ticketStorage{db: db}
}

func (s *ticketStorage) Save(ctx context.Context, ticket *models.SupportTicket) (int64, error) {
	log := logger.FromContext(ctx)
	query := `
		INSERT INTO support_ticket (
			profile_id,
			login,
			email,
			category,
			title,
			description,
			status,
			priority,
			created_at,
			updated_at,
			closed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	start := time.Now()
	row := s.db.QueryRow(
		ctx,
		query,
		ticket.ProfileID,
		ticket.Login,
		ticket.Email,
		int(ticket.Category),
		ticket.Title,
		ticket.Description,
		int(ticket.Status),
		int(ticket.Priority),
		ticket.CreatedAt,
		ticket.UpdatedAt,
		ticket.ClosedAt,
	)
	if log != nil {
		log.Debug("db query",
			zap.String("query", "ticketStorage.Save"),
			zap.Duration("duration_ms", time.Since(start)))
	}

	var ticketID int64
	if err := row.Scan(&ticketID); err != nil {
		return 0, pgerrors.MapPgError(err)
	}

	ticket.ID = ticketID
	return ticketID, nil
}
