package support

//go:generate mockgen -destination=./../mocks/support_ticket_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/repository/support TicketRepository

import (
	"context"
	"errors"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	pgerrors "github.com/go-park-mail-ru/2026_1_ARIS/internal/utils/pg_errors"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type TicketRepository interface {
	Save(ctx context.Context, ticket *models.SupportTicket) (int64, error)
	GetByIDAndProfileID(ctx context.Context, ticketID, profileID int64) (*models.SupportTicket, error)
	GetByProfileID(ctx context.Context, profileID int64) ([]models.SupportTicket, error)
	Update(ctx context.Context, ticket *models.SupportTicket) (*models.SupportTicket, error)
}

type ticketStorage struct {
	db ticketDB
}

type ticketDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
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

func (s *ticketStorage) GetByIDAndProfileID(ctx context.Context, ticketID, profileID int64) (*models.SupportTicket, error) {
	log := logger.FromContext(ctx)
	query := `
		SELECT id, profile_id, login, email, category, title, description, status, priority, created_at, updated_at, closed_at
		FROM support_ticket
		WHERE id = $1 AND profile_id = $2`

	start := time.Now()
	row := s.db.QueryRow(ctx, query, ticketID, profileID)
	if log != nil {
		log.Debug("db query",
			zap.String("query", "ticketStorage.GetByIDAndProfileID"),
			zap.Duration("duration_ms", time.Since(start)))
	}

	ticket, err := scanSupportTicket(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, xerrors.SupportTicketNotFound
	}
	if err != nil {
		return nil, err
	}

	return ticket, nil
}

func (s *ticketStorage) GetByProfileID(ctx context.Context, profileID int64) ([]models.SupportTicket, error) {
	log := logger.FromContext(ctx)
	query := `
		SELECT id, profile_id, login, email, category, title, description, status, priority, created_at, updated_at, closed_at
		FROM support_ticket
		WHERE profile_id = $1
		ORDER BY created_at DESC`

	start := time.Now()
	rows, err := s.db.Query(ctx, query, profileID)
	if log != nil {
		log.Debug("db query",
			zap.String("query", "ticketStorage.GetByProfileID"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil {
		return nil, err
	}

	tickets, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.SupportTicket, error) {
		ticket, err := scanSupportTicket(row)
		if err != nil {
			return models.SupportTicket{}, err
		}
		return *ticket, nil
	})
	if err != nil {
		return nil, err
	}

	return tickets, nil
}

func (s *ticketStorage) Update(ctx context.Context, ticket *models.SupportTicket) (*models.SupportTicket, error) {
	log := logger.FromContext(ctx)
	query := `
		UPDATE support_ticket
		SET title = $1,
			description = $2,
			category = $3,
			updated_at = $4
		WHERE id = $5 AND profile_id = $6
		RETURNING id, profile_id, login, email, category, title, description, status, priority, created_at, updated_at, closed_at`

	start := time.Now()
	row := s.db.QueryRow(
		ctx,
		query,
		ticket.Title,
		ticket.Description,
		int(ticket.Category),
		ticket.UpdatedAt,
		ticket.ID,
		ticket.ProfileID,
	)
	if log != nil {
		log.Debug("db query",
			zap.String("query", "ticketStorage.Update"),
			zap.Duration("duration_ms", time.Since(start)))
	}

	updatedTicket, err := scanSupportTicket(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, xerrors.SupportTicketNotFound
	}
	if err != nil {
		return nil, pgerrors.MapPgError(err)
	}

	return updatedTicket, nil
}

type supportTicketRow interface {
	Scan(dest ...any) error
}

func scanSupportTicket(row supportTicketRow) (*models.SupportTicket, error) {
	var ticket models.SupportTicket
	err := row.Scan(
		&ticket.ID,
		&ticket.ProfileID,
		&ticket.Login,
		&ticket.Email,
		&ticket.Category,
		&ticket.Title,
		&ticket.Description,
		&ticket.Status,
		&ticket.Priority,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
		&ticket.ClosedAt,
	)
	if err != nil {
		return nil, err
	}

	return &ticket, nil
}
