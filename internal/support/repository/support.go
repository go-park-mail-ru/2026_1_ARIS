package repository

//go:generate mockgen -destination=../../repository/mocks/support_ticket_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_ARIS/internal/support/repository TicketRepository

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

type TicketFilter struct {
	Status          *models.TicketStatus
	Category        *models.TicketCategory
	Line            *int
	AssignedAgentID *int64
}

type TicketRepository interface {
	Save(ctx context.Context, ticket *models.SupportTicket) (int64, error)
	GetByID(ctx context.Context, ticketID int64) (*models.SupportTicket, error)
	GetByIDAndProfileID(ctx context.Context, ticketID, profileID int64) (*models.SupportTicket, error)
	GetAll(ctx context.Context, filter TicketFilter) ([]models.SupportTicket, error)
	GetByProfileID(ctx context.Context, profileID int64) ([]models.SupportTicket, error)
	Update(ctx context.Context, ticket *models.SupportTicket) (*models.SupportTicket, error)
	UpdateStatusByID(ctx context.Context, ticketID int64, status models.TicketStatus, closedAt *time.Time, updatedAt time.Time) (*models.SupportTicket, error)
	UpdateStatus(ctx context.Context, ticketID, profileID int64, status models.TicketStatus, closedAt *time.Time, updatedAt time.Time) (*models.SupportTicket, error)
	Assign(ctx context.Context, ticketID, agentID int64, updatedAt time.Time) (*models.SupportTicket, error)
	Escalate(ctx context.Context, ticketID int64, updatedAt time.Time) (*models.SupportTicket, error)
	Rate(ctx context.Context, ticketID, profileID int64, rating int, updatedAt time.Time) (*models.SupportTicket, error)
	GetStats(ctx context.Context) (*models.SupportTicketStats, error)
	SetProfileRole(ctx context.Context, profileID int64, role models.SupportRole) error
	GetProfileRole(ctx context.Context, profileID int64) (*models.SupportProfileRole, error)
	SaveMedia(ctx context.Context, ticketWithMedia models.TicketWithMedia) error
	GetMediaByTicketID(ctx context.Context, ticketID int64) []int64
	GetMessages(ctx context.Context, ticketID int64) ([]models.SupportTicketMessage, error)
	SaveMessage(ctx context.Context, message *models.SupportTicketMessage) (int64, error)
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

const supportTicketColumns = `
	id, uid, profile_id, login, email, category, title, description, status, priority,
	line, assigned_agent_id, rating, created_at, updated_at, closed_at`

func (s *ticketStorage) Save(ctx context.Context, ticket *models.SupportTicket) (int64, error) {
	log := logger.FromContext(ctx)
	query := `
		INSERT INTO support_ticket (
			uid, profile_id, login, email, category, title, description, status, priority,
			line, assigned_agent_id, rating, created_at, updated_at, closed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id`

	start := time.Now()
	row := s.db.QueryRow(
		ctx,
		query,
		ticket.Uid,
		ticket.ProfileID,
		ticket.Login,
		ticket.Email,
		int(ticket.Category),
		ticket.Title,
		ticket.Description,
		int(ticket.Status),
		int(ticket.Priority),
		ticket.Line,
		ticket.AssignedAgentID,
		ticket.Rating,
		ticket.CreatedAt,
		ticket.UpdatedAt,
		ticket.ClosedAt,
	)
	if log != nil {
		log.Debug("db query", zap.String("query", "ticketStorage.Save"), zap.Duration("duration_ms", time.Since(start)))
	}

	var ticketID int64
	if err := row.Scan(&ticketID); err != nil {
		return 0, pgerrors.MapPgError(err)
	}

	ticket.ID = ticketID
	return ticketID, nil
}

func (s *ticketStorage) GetByIDAndProfileID(ctx context.Context, ticketID, profileID int64) (*models.SupportTicket, error) {
	ticket, err := s.GetByID(ctx, ticketID)
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
	query := `SELECT ` + supportTicketColumns + ` FROM support_ticket WHERE id = $1`

	start := time.Now()
	row := s.db.QueryRow(ctx, query, ticketID)
	if log != nil {
		log.Debug("db query", zap.String("query", "ticketStorage.GetByID"), zap.Duration("duration_ms", time.Since(start)))
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

func (s *ticketStorage) GetAll(ctx context.Context, filter TicketFilter) ([]models.SupportTicket, error) {
	log := logger.FromContext(ctx)
	query := `
		SELECT ` + supportTicketColumns + `
		FROM support_ticket
		WHERE ($1::int IS NULL OR status = $1)
			AND ($2::int IS NULL OR category = $2)
			AND ($3::int IS NULL OR line = $3)
			AND ($4::bigint IS NULL OR assigned_agent_id = $4)
		ORDER BY created_at DESC`

	var status, category, line, assignedAgentID any
	if filter.Status != nil {
		status = int(*filter.Status)
	}
	if filter.Category != nil {
		category = int(*filter.Category)
	}
	if filter.Line != nil {
		line = *filter.Line
	}
	if filter.AssignedAgentID != nil {
		assignedAgentID = *filter.AssignedAgentID
	}

	start := time.Now()
	rows, err := s.db.Query(ctx, query, status, category, line, assignedAgentID)
	if log != nil {
		log.Debug("db query", zap.String("query", "ticketStorage.GetAll"), zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil {
		return nil, err
	}

	return collectSupportTickets(rows)
}

func (s *ticketStorage) GetByProfileID(ctx context.Context, profileID int64) ([]models.SupportTicket, error) {
	log := logger.FromContext(ctx)
	query := `
		SELECT ` + supportTicketColumns + `
		FROM support_ticket
		WHERE profile_id = $1
		ORDER BY created_at DESC`

	start := time.Now()
	rows, err := s.db.Query(ctx, query, profileID)
	if log != nil {
		log.Debug("db query", zap.String("query", "ticketStorage.GetByProfileID"), zap.Duration("duration_ms", time.Since(start)))
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
		SET title = $1, description = $2, category = $3, updated_at = $4
		WHERE id = $5 AND profile_id = $6
		RETURNING ` + supportTicketColumns

	start := time.Now()
	row := s.db.QueryRow(ctx, query, ticket.Title, ticket.Description, int(ticket.Category), ticket.UpdatedAt, ticket.ID, ticket.ProfileID)
	if log != nil {
		log.Debug("db query", zap.String("query", "ticketStorage.Update"), zap.Duration("duration_ms", time.Since(start)))
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

func (s *ticketStorage) UpdateStatus(ctx context.Context, ticketID, profileID int64, status models.TicketStatus, closedAt *time.Time, updatedAt time.Time) (*models.SupportTicket, error) {
	log := logger.FromContext(ctx)
	query := `
		UPDATE support_ticket
		SET status = $1, closed_at = $2, updated_at = $3
		WHERE id = $4 AND profile_id = $5
		RETURNING ` + supportTicketColumns

	start := time.Now()
	row := s.db.QueryRow(ctx, query, int(status), closedAt, updatedAt, ticketID, profileID)
	if log != nil {
		log.Debug("db query", zap.String("query", "ticketStorage.UpdateStatus"), zap.Duration("duration_ms", time.Since(start)))
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

func (s *ticketStorage) UpdateStatusByID(ctx context.Context, ticketID int64, status models.TicketStatus, closedAt *time.Time, updatedAt time.Time) (*models.SupportTicket, error) {
	log := logger.FromContext(ctx)
	query := `
		UPDATE support_ticket
		SET status = $1, closed_at = $2, updated_at = $3
		WHERE id = $4
		RETURNING ` + supportTicketColumns

	start := time.Now()
	row := s.db.QueryRow(ctx, query, int(status), closedAt, updatedAt, ticketID)
	if log != nil {
		log.Debug("db query", zap.String("query", "ticketStorage.UpdateStatusByID"), zap.Duration("duration_ms", time.Since(start)))
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

func (s *ticketStorage) Assign(ctx context.Context, ticketID, agentID int64, updatedAt time.Time) (*models.SupportTicket, error) {
	log := logger.FromContext(ctx)
	query := `
		UPDATE support_ticket
		SET assigned_agent_id = $1, status = $2, updated_at = $3
		WHERE id = $4
		RETURNING ` + supportTicketColumns

	start := time.Now()
	row := s.db.QueryRow(ctx, query, agentID, int(models.TicketStatusInProgress), updatedAt, ticketID)
	if log != nil {
		log.Debug("db query", zap.String("query", "ticketStorage.Assign"), zap.Duration("duration_ms", time.Since(start)))
	}

	ticket, err := scanSupportTicket(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, xerrors.SupportTicketNotFound
	}
	return ticket, pgerrors.MapPgError(err)
}

func (s *ticketStorage) Escalate(ctx context.Context, ticketID int64, updatedAt time.Time) (*models.SupportTicket, error) {
	log := logger.FromContext(ctx)
	query := `
		UPDATE support_ticket
		SET line = 2, status = $1, assigned_agent_id = NULL, updated_at = $2
		WHERE id = $3
		RETURNING ` + supportTicketColumns

	start := time.Now()
	row := s.db.QueryRow(ctx, query, int(models.TicketStatusOpen), updatedAt, ticketID)
	if log != nil {
		log.Debug("db query", zap.String("query", "ticketStorage.Escalate"), zap.Duration("duration_ms", time.Since(start)))
	}

	ticket, err := scanSupportTicket(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, xerrors.SupportTicketNotFound
	}
	return ticket, pgerrors.MapPgError(err)
}

func (s *ticketStorage) Rate(ctx context.Context, ticketID, profileID int64, rating int, updatedAt time.Time) (*models.SupportTicket, error) {
	log := logger.FromContext(ctx)
	query := `
		UPDATE support_ticket
		SET rating = $1, updated_at = $2
		WHERE id = $3 AND profile_id = $4 AND status = $5 AND rating IS NULL
		RETURNING ` + supportTicketColumns

	start := time.Now()
	row := s.db.QueryRow(ctx, query, rating, updatedAt, ticketID, profileID, int(models.TicketStatusClosed))
	if log != nil {
		log.Debug("db query", zap.String("query", "ticketStorage.Rate"), zap.Duration("duration_ms", time.Since(start)))
	}

	ticket, err := scanSupportTicket(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, xerrors.SupportTicketNotFound
	}
	return ticket, pgerrors.MapPgError(err)
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
			(AVG(rating) FILTER (WHERE rating IS NOT NULL))::float8
		FROM support_ticket`

	start := time.Now()
	row := s.db.QueryRow(ctx, query)
	if log != nil {
		log.Debug("db query", zap.String("query", "ticketStorage.GetStatsTotals"), zap.Duration("duration_ms", time.Since(start)))
	}

	stats := &models.SupportTicketStats{
		ByCategory:         map[string]int64{"bug": 0, "feature_request": 0, "complaint": 0, "question": 0, "other": 0},
		ByLine:             map[string]int64{"l1": 0, "l2": 0},
		RatingDistribution: map[string]int64{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0},
	}
	if err := row.Scan(&stats.TotalCount, &stats.OpenCount, &stats.InProgressCount, &stats.WaitingUserCount, &stats.ClosedCount, &stats.AverageRating); err != nil {
		return nil, err
	}

	if err := s.fillStatsMap(ctx, `SELECT category, COUNT(*)::bigint FROM support_ticket GROUP BY category`, func(key int, count int64) {
		names := []string{"bug", "feature_request", "complaint", "question", "other"}
		if key >= 0 && key < len(names) {
			stats.ByCategory[names[key]] = count
		}
	}); err != nil {
		return nil, err
	}

	if err := s.fillStatsMap(ctx, `SELECT line, COUNT(*)::bigint FROM support_ticket GROUP BY line`, func(key int, count int64) {
		if key == 1 {
			stats.ByLine["l1"] = count
		}
		if key == 2 {
			stats.ByLine["l2"] = count
		}
	}); err != nil {
		return nil, err
	}

	if err := s.fillStatsMap(ctx, `SELECT rating, COUNT(*)::bigint FROM support_ticket WHERE rating IS NOT NULL GROUP BY rating`, func(key int, count int64) {
		if key >= 1 && key <= 5 {
			stats.RatingDistribution[string(rune('0'+key))] = count
		}
	}); err != nil {
		return nil, err
	}

	return stats, nil
}

func (s *ticketStorage) fillStatsMap(ctx context.Context, query string, apply func(key int, count int64)) error {
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var key int
		var count int64
		if err := rows.Scan(&key, &count); err != nil {
			return err
		}
		apply(key, count)
	}
	return rows.Err()
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
		log.Debug("db query", zap.String("query", "ticketStorage.SetProfileRole"), zap.Duration("duration_ms", time.Since(start)))
	}
	return pgerrors.MapPgError(err)
}

func (s *ticketStorage) GetProfileRole(ctx context.Context, profileID int64) (*models.SupportProfileRole, error) {
	log := logger.FromContext(ctx)
	query := `SELECT profile_id, role FROM support_profile_role WHERE profile_id = $1`

	start := time.Now()
	row := s.db.QueryRow(ctx, query, profileID)
	if log != nil {
		log.Debug("db query", zap.String("query", "ticketStorage.GetProfileRole"), zap.Duration("duration_ms", time.Since(start)))
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

func (s *ticketStorage) SaveMedia(ctx context.Context, ticketWithMedia models.TicketWithMedia) error {
	log := logger.FromContext(ctx)
	query := `INSERT INTO ticket_with_media (ticket_id, media_id, sort_order) VALUES ($1, $2, $3)`

	start := time.Now()
	res, err := s.db.Exec(ctx, query, ticketWithMedia.TicketID, ticketWithMedia.MediaID, ticketWithMedia.Order)
	if log != nil {
		log.Debug("db query", zap.String("query", "ticketStorage.SaveMedia"), zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil {
		return pgerrors.MapPgError(err)
	}
	if res.RowsAffected() != 1 {
		return errors.New("affected not on 1 row")
	}
	return nil
}

func (s *ticketStorage) GetMediaByTicketID(ctx context.Context, ticketID int64) []int64 {
	rows, err := s.db.Query(ctx, `SELECT media_id FROM ticket_with_media WHERE ticket_id = $1 ORDER BY sort_order`, ticketID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var mediaIDs []int64
	for rows.Next() {
		var mediaID int64
		if err := rows.Scan(&mediaID); err == nil {
			mediaIDs = append(mediaIDs, mediaID)
		}
	}
	return mediaIDs
}

func (s *ticketStorage) GetMessages(ctx context.Context, ticketID int64) ([]models.SupportTicketMessage, error) {
	log := logger.FromContext(ctx)
	query := `
		SELECT id, ticket_id, text, author_id, author_role, created_at
		FROM support_ticket_message
		WHERE ticket_id = $1
		ORDER BY created_at ASC, id ASC`

	start := time.Now()
	rows, err := s.db.Query(ctx, query, ticketID)
	if log != nil {
		log.Debug("db query", zap.String("query", "ticketStorage.GetMessages"), zap.Duration("duration_ms", time.Since(start)))
	}
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.SupportTicketMessage, error) {
		var message models.SupportTicketMessage
		err := row.Scan(&message.ID, &message.TicketID, &message.Text, &message.AuthorID, &message.AuthorRole, &message.CreatedAt)
		return message, err
	})
}

func (s *ticketStorage) SaveMessage(ctx context.Context, message *models.SupportTicketMessage) (int64, error) {
	log := logger.FromContext(ctx)
	query := `
		INSERT INTO support_ticket_message (ticket_id, text, author_id, author_role, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	start := time.Now()
	row := s.db.QueryRow(ctx, query, message.TicketID, message.Text, message.AuthorID, string(message.AuthorRole), message.CreatedAt)
	if log != nil {
		log.Debug("db query", zap.String("query", "ticketStorage.SaveMessage"), zap.Duration("duration_ms", time.Since(start)))
	}

	var messageID int64
	if err := row.Scan(&messageID); err != nil {
		return 0, pgerrors.MapPgError(err)
	}
	message.ID = messageID
	return messageID, nil
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
		&ticket.Uid,
		&ticket.ProfileID,
		&ticket.Login,
		&ticket.Email,
		&ticket.Category,
		&ticket.Title,
		&ticket.Description,
		&ticket.Status,
		&ticket.Priority,
		&ticket.Line,
		&ticket.AssignedAgentID,
		&ticket.Rating,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
		&ticket.ClosedAt,
	)
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}
