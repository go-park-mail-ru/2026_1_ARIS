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
	GetByID(ctx context.Context, ticketID int64) (*models.SupportTicket, error)
	GetByIDAndProfileID(ctx context.Context, ticketID, profileID int64) (*models.SupportTicket, error)
	GetAll(ctx context.Context) ([]models.SupportTicket, error)
	GetByProfileID(ctx context.Context, profileID int64) ([]models.SupportTicket, error)
	Update(ctx context.Context, ticket *models.SupportTicket) (*models.SupportTicket, error)
	UpdateStatusByID(ctx context.Context, ticketID int64, status models.TicketStatus, closedAt *time.Time, updatedAt time.Time) (*models.SupportTicket, error)
	UpdateStatus(ctx context.Context, ticketID, profileID int64, status models.TicketStatus, closedAt *time.Time, updatedAt time.Time) (*models.SupportTicket, error)
	GetStats(ctx context.Context) (*models.SupportTicketStats, error)
	SetProfileRole(ctx context.Context, profileID int64, role models.SupportRole) error
	GetProfileRole(ctx context.Context, profileID int64) (*models.SupportProfileRole, error)
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
		WHERE id = $1`

	start := time.Now()
	row := s.db.QueryRow(ctx, query, ticketID)
	if log != nil {
		log.Debug("db query",
			zap.String("query", "ticketStorage.GetByID"),
			zap.Duration("duration_ms", time.Since(start)))
	}

	ticket, err := scanSupportTicket(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, xerrors.SupportTicketNotFound
	}
	if err != nil {
		return nil, err
	}

	if ticket.ProfileID != profileID {
		return nil, xerrors.SupportTicketNotFound
	}

	return ticket, nil
}

func (s *ticketStorage) GetByID(ctx context.Context, ticketID int64) (*models.SupportTicket, error) {
	log := logger.FromContext(ctx)
	query := `
		SELECT id, profile_id, login, email, category, title, description, status, priority, created_at, updated_at, closed_at
		FROM support_ticket
		WHERE id = $1`

	start := time.Now()
	row := s.db.QueryRow(ctx, query, ticketID)
	if log != nil {
		log.Debug("db query",
			zap.String("query", "ticketStorage.GetByID"),
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

func (s *ticketStorage) GetAll(ctx context.Context) ([]models.SupportTicket, error) {
	log := logger.FromContext(ctx)
	query := `
		SELECT id, profile_id, login, email, category, title, description, status, priority, created_at, updated_at, closed_at
		FROM support_ticket
		ORDER BY created_at DESC`

	start := time.Now()
	rows, err := s.db.Query(ctx, query)
	if log != nil {
		log.Debug("db query",
			zap.String("query", "ticketStorage.GetAll"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil {
		return nil, err
	}

	return collectSupportTickets(rows)
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

	return collectSupportTickets(rows)
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

func (s *ticketStorage) UpdateStatus(
	ctx context.Context,
	ticketID,
	profileID int64,
	status models.TicketStatus,
	closedAt *time.Time,
	updatedAt time.Time,
) (*models.SupportTicket, error) {
	log := logger.FromContext(ctx)
	query := `
		UPDATE support_ticket
		SET status = $1,
			closed_at = $2,
			updated_at = $3
		WHERE id = $4 AND profile_id = $5
		RETURNING id, profile_id, login, email, category, title, description, status, priority, created_at, updated_at, closed_at`

	start := time.Now()
	row := s.db.QueryRow(ctx, query, int(status), closedAt, updatedAt, ticketID, profileID)
	if log != nil {
		log.Debug("db query",
			zap.String("query", "ticketStorage.UpdateStatus"),
			zap.Duration("duration_ms", time.Since(start)))
	}

	ticket, err := scanSupportTicket(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, xerrors.SupportTicketNotFound
	}
	if err != nil {
		return nil, pgerrors.MapPgError(err)
	}

	return ticket, nil
}

func (s *ticketStorage) UpdateStatusByID(
	ctx context.Context,
	ticketID int64,
	status models.TicketStatus,
	closedAt *time.Time,
	updatedAt time.Time,
) (*models.SupportTicket, error) {
	log := logger.FromContext(ctx)
	query := `
		UPDATE support_ticket
		SET status = $1,
			closed_at = $2,
			updated_at = $3
		WHERE id = $4
		RETURNING id, profile_id, login, email, category, title, description, status, priority, created_at, updated_at, closed_at`

	start := time.Now()
	row := s.db.QueryRow(ctx, query, int(status), closedAt, updatedAt, ticketID)
	if log != nil {
		log.Debug("db query",
			zap.String("query", "ticketStorage.UpdateStatusByID"),
			zap.Duration("duration_ms", time.Since(start)))
	}

	ticket, err := scanSupportTicket(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, xerrors.SupportTicketNotFound
	}
	if err != nil {
		return nil, pgerrors.MapPgError(err)
	}

	return ticket, nil
}

func (s *ticketStorage) GetStats(ctx context.Context) (*models.SupportTicketStats, error) {
	log := logger.FromContext(ctx)
	query := `
		SELECT
			COUNT(*)::bigint,
			COUNT(*) FILTER (WHERE status = 0)::bigint,
			COUNT(*) FILTER (WHERE status = 1)::bigint,
			COUNT(*) FILTER (WHERE status = 2)::bigint,
			COUNT(*) FILTER (WHERE status = 3)::bigint,
			AVG(EXTRACT(EPOCH FROM (closed_at - created_at))) FILTER (WHERE closed_at IS NOT NULL)
		FROM support_ticket`

	start := time.Now()
	row := s.db.QueryRow(ctx, query)
	if log != nil {
		log.Debug("db query",
			zap.String("query", "ticketStorage.GetStatsTotals"),
			zap.Duration("duration_ms", time.Since(start)))
	}

	stats := &models.SupportTicketStats{}
	if err := row.Scan(
		&stats.TotalCount,
		&stats.OpenCount,
		&stats.InProgressCount,
		&stats.WaitingUserCount,
		&stats.ClosedCount,
		&stats.AverageCloseTimeSeconds,
	); err != nil {
		return nil, err
	}

	byCategory, err := s.getStatsByCategory(ctx)
	if err != nil {
		return nil, err
	}
	stats.ByCategory = byCategory

	byStatus, err := s.getStatsByStatus(ctx)
	if err != nil {
		return nil, err
	}
	stats.ByStatus = byStatus

	return stats, nil
}

func (s *ticketStorage) getStatsByCategory(ctx context.Context) ([]models.SupportTicketCategoryStats, error) {
	log := logger.FromContext(ctx)
	query := `
		SELECT category, COUNT(*)::bigint
		FROM support_ticket
		GROUP BY category
		ORDER BY category`

	start := time.Now()
	rows, err := s.db.Query(ctx, query)
	if log != nil {
		log.Debug("db query",
			zap.String("query", "ticketStorage.GetStatsByCategory"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.SupportTicketCategoryStats, error) {
		var stat models.SupportTicketCategoryStats
		err := row.Scan(&stat.Category, &stat.Count)
		return stat, err
	})
}

func (s *ticketStorage) getStatsByStatus(ctx context.Context) ([]models.SupportTicketStatusStats, error) {
	log := logger.FromContext(ctx)
	query := `
		SELECT status, COUNT(*)::bigint
		FROM support_ticket
		GROUP BY status
		ORDER BY status`

	start := time.Now()
	rows, err := s.db.Query(ctx, query)
	if log != nil {
		log.Debug("db query",
			zap.String("query", "ticketStorage.GetStatsByStatus"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.SupportTicketStatusStats, error) {
		var stat models.SupportTicketStatusStats
		err := row.Scan(&stat.Status, &stat.Count)
		return stat, err
	})
}

func (s *ticketStorage) SetProfileRole(ctx context.Context, profileID int64, role models.SupportRole) error {
	log := logger.FromContext(ctx)
	query := `
		INSERT INTO support_profile_role (profile_id, role)
		VALUES ($1, $2)
		ON CONFLICT (profile_id) DO UPDATE SET role = EXCLUDED.role`

	start := time.Now()
	_, err := s.db.Exec(ctx, query, profileID, string(role))
	if log != nil {
		log.Debug("db query",
			zap.String("query", "ticketStorage.SetProfileRole"),
			zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil {
		return pgerrors.MapPgError(err)
	}

	return nil
}

func (s *ticketStorage) GetProfileRole(ctx context.Context, profileID int64) (*models.SupportProfileRole, error) {
	log := logger.FromContext(ctx)
	query := `
		SELECT profile_id, role
		FROM support_profile_role
		WHERE profile_id = $1`

	start := time.Now()
	row := s.db.QueryRow(ctx, query, profileID)
	if log != nil {
		log.Debug("db query",
			zap.String("query", "ticketStorage.GetProfileRole"),
			zap.Duration("duration_ms", time.Since(start)))
	}

	var role models.SupportProfileRole
	if err := row.Scan(&role.ProfileID, &role.Role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, xerrors.SupportForbidden
		}
		return nil, err
	}

	return &role, nil
}

type supportTicketRow interface {
	Scan(dest ...any) error
}

func collectSupportTickets(rows pgx.Rows) ([]models.SupportTicket, error) {
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.SupportTicket, error) {
		ticket, err := scanSupportTicket(row)
		if err != nil {
			return models.SupportTicket{}, err
		}
		return *ticket, nil
	})
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
