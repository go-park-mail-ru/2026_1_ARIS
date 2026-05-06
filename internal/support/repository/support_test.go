package repository

import (
	"context"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models"
	"github.com/go-park-mail-ru/2026_1_ARIS/internal/models/xerrors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func ticketColumns() []string {
	return []string{
		"id", "uid", "profile_id", "login", "email", "category", "title", "description", "status", "priority",
		"line", "assigned_agent_id", "rating", "created_at", "updated_at", "closed_at",
	}
}

func addTicketRow(rows *pgxmock.Rows, id, profileID int64) *pgxmock.Rows {
	now := time.Now()
	agentID := int64(33)
	rating := 5
	return rows.AddRow(
		id, uuid.New(), profileID, "neo", "neo@example.test", models.CategoryBug, "Bug", "Description",
		models.TicketStatusOpen, models.TicketPriorityLow, 1, &agentID, &rating, now, now, nil,
	)
}

func newSupportRepo(t *testing.T) (pgxmock.PgxPoolIface, TicketRepository) {
	t.Helper()
	mockPool, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mockPool.Close)
	return mockPool, NewTicketStorage(mockPool)
}

func TestTicketStorageSaveAndGet(t *testing.T) {
	t.Parallel()

	t.Run("Save", func(t *testing.T) {
		t.Parallel()
		mockPool, repo := newSupportRepo(t)
		ticket := models.NewSupportTicket(10, "neo", "neo@example.test", models.CategoryBug, "Bug", "Description")
		mockPool.ExpectQuery("INSERT INTO support_ticket").
			WithArgs(ticket.Uid, ticket.ProfileID, ticket.Login, ticket.Email, int(ticket.Category), ticket.Title, ticket.Description, int(ticket.Status), int(ticket.Priority), ticket.Line, ticket.AssignedAgentID, ticket.Rating, ticket.CreatedAt, ticket.UpdatedAt, ticket.ClosedAt).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(99)))

		id, err := repo.Save(context.Background(), ticket)

		require.NoError(t, err)
		require.Equal(t, int64(99), id)
		require.Equal(t, int64(99), ticket.ID)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("GetByID", func(t *testing.T) {
		t.Parallel()
		mockPool, repo := newSupportRepo(t)
		mockPool.ExpectQuery("FROM support_ticket WHERE id = \\$1").WithArgs(int64(99)).
			WillReturnRows(addTicketRow(pgxmock.NewRows(ticketColumns()), 99, 10))

		ticket, err := repo.GetByID(context.Background(), 99)

		require.NoError(t, err)
		require.Equal(t, int64(99), ticket.ID)
		require.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("GetByID not found", func(t *testing.T) {
		t.Parallel()
		mockPool, repo := newSupportRepo(t)
		mockPool.ExpectQuery("FROM support_ticket WHERE id = \\$1").WithArgs(int64(404)).
			WillReturnError(pgx.ErrNoRows)

		_, err := repo.GetByID(context.Background(), 404)

		require.ErrorIs(t, err, xerrors.SupportTicketNotFound)
	})

	t.Run("GetByIDAndProfileID mismatch", func(t *testing.T) {
		t.Parallel()
		mockPool, repo := newSupportRepo(t)
		mockPool.ExpectQuery("FROM support_ticket WHERE id = \\$1").WithArgs(int64(99)).
			WillReturnRows(addTicketRow(pgxmock.NewRows(ticketColumns()), 99, 10))

		_, err := repo.GetByIDAndProfileID(context.Background(), 99, 11)

		require.ErrorIs(t, err, xerrors.SupportTicketNotFound)
	})
}

func TestTicketStorageQueries(t *testing.T) {
	t.Parallel()

	t.Run("GetAll with filter", func(t *testing.T) {
		t.Parallel()
		mockPool, repo := newSupportRepo(t)
		status := models.TicketStatusOpen
		category := models.CategoryBug
		line := 1
		agentID := int64(33)
		mockPool.ExpectQuery("FROM support_ticket").
			WithArgs(int(status), int(category), line, agentID).
			WillReturnRows(addTicketRow(pgxmock.NewRows(ticketColumns()), 99, 10))

		got, err := repo.GetAll(context.Background(), TicketFilter{Status: &status, Category: &category, Line: &line, AssignedAgentID: &agentID})

		require.NoError(t, err)
		require.Len(t, got, 1)
	})

	t.Run("GetByProfileID", func(t *testing.T) {
		t.Parallel()
		mockPool, repo := newSupportRepo(t)
		mockPool.ExpectQuery("WHERE profile_id = \\$1").WithArgs(int64(10)).
			WillReturnRows(addTicketRow(pgxmock.NewRows(ticketColumns()), 99, 10))

		got, err := repo.GetByProfileID(context.Background(), 10)

		require.NoError(t, err)
		require.Len(t, got, 1)
	})
}

func TestTicketStorageUpdateMethods(t *testing.T) {
	t.Parallel()

	now := time.Now()
	for _, tc := range []struct {
		name string
		call func(TicketRepository) (*models.SupportTicket, error)
		args []any
	}{
		{
			name: "Update",
			call: func(repo TicketRepository) (*models.SupportTicket, error) {
				return repo.Update(context.Background(), &models.SupportTicket{ID: 99, ProfileID: 10, Title: "T", Description: "D", Category: models.CategoryQuestion, UpdatedAt: now})
			},
			args: []any{"T", "D", int(models.CategoryQuestion), now, int64(99), int64(10)},
		},
		{
			name: "UpdateStatus",
			call: func(repo TicketRepository) (*models.SupportTicket, error) {
				return repo.UpdateStatus(context.Background(), 99, 10, models.TicketStatusClosed, &now, now)
			},
			args: []any{int(models.TicketStatusClosed), &now, now, int64(99), int64(10)},
		},
		{
			name: "UpdateStatusByID",
			call: func(repo TicketRepository) (*models.SupportTicket, error) {
				return repo.UpdateStatusByID(context.Background(), 99, models.TicketStatusInProgress, nil, now)
			},
			args: []any{int(models.TicketStatusInProgress), (*time.Time)(nil), now, int64(99)},
		},
		{
			name: "Assign",
			call: func(repo TicketRepository) (*models.SupportTicket, error) {
				return repo.Assign(context.Background(), 99, 33, now)
			},
			args: []any{int64(33), int(models.TicketStatusInProgress), now, int64(99)},
		},
		{
			name: "Escalate",
			call: func(repo TicketRepository) (*models.SupportTicket, error) {
				return repo.Escalate(context.Background(), 99, now)
			},
			args: []any{int(models.TicketStatusOpen), now, int64(99)},
		},
		{
			name: "Rate",
			call: func(repo TicketRepository) (*models.SupportTicket, error) {
				return repo.Rate(context.Background(), 99, 10, 5, now)
			},
			args: []any{5, now, int64(99), int64(10), int(models.TicketStatusClosed)},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mockPool, repo := newSupportRepo(t)
			mockPool.ExpectQuery("UPDATE support_ticket").
				WithArgs(tc.args...).
				WillReturnRows(addTicketRow(pgxmock.NewRows(ticketColumns()), 99, 10))

			got, err := tc.call(repo)

			require.NoError(t, err)
			require.Equal(t, int64(99), got.ID)
			require.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestTicketStorageStatsRolesMediaAndMessages(t *testing.T) {
	t.Parallel()

	t.Run("GetStats", func(t *testing.T) {
		t.Parallel()
		mockPool, repo := newSupportRepo(t)
		avg := 4.5
		mockPool.ExpectQuery("COUNT\\(\\*\\)::bigint").WillReturnRows(
			pgxmock.NewRows([]string{"total", "open", "in_progress", "waiting", "closed", "avg"}).
				AddRow(int64(5), int64(2), int64(1), int64(1), int64(1), &avg),
		)
		mockPool.ExpectQuery("SELECT category").WillReturnRows(pgxmock.NewRows([]string{"category", "count"}).AddRow(0, int64(2)))
		mockPool.ExpectQuery("SELECT line").WillReturnRows(pgxmock.NewRows([]string{"line", "count"}).AddRow(2, int64(3)))
		mockPool.ExpectQuery("SELECT rating").WillReturnRows(pgxmock.NewRows([]string{"rating", "count"}).AddRow(5, int64(1)))

		stats, err := repo.GetStats(context.Background())

		require.NoError(t, err)
		require.Equal(t, int64(5), stats.TotalCount)
		require.Equal(t, int64(2), stats.ByCategory["bug"])
		require.Equal(t, int64(3), stats.ByLine["l2"])
		require.Equal(t, int64(1), stats.RatingDistribution["5"])
	})

	t.Run("roles", func(t *testing.T) {
		t.Parallel()
		mockPool, repo := newSupportRepo(t)
		mockPool.ExpectExec("INSERT INTO support_profile_role").WithArgs(int64(10), string(models.SupportRoleAdmin)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		require.NoError(t, repo.SetProfileRole(context.Background(), 10, models.SupportRoleAdmin))

		mockPool.ExpectQuery("SELECT profile_id, role").WithArgs(int64(10)).
			WillReturnRows(pgxmock.NewRows([]string{"profile_id", "role"}).AddRow(int64(10), models.SupportRoleAdmin))
		role, err := repo.GetProfileRole(context.Background(), 10)
		require.NoError(t, err)
		require.Equal(t, models.SupportRoleAdmin, role.Role)
	})

	t.Run("media", func(t *testing.T) {
		t.Parallel()
		mockPool, repo := newSupportRepo(t)
		mockPool.ExpectExec("INSERT INTO ticket_with_media").WithArgs(int64(99), int64(5), 1).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		require.NoError(t, repo.SaveMedia(context.Background(), models.TicketWithMedia{TicketID: 99, MediaID: 5, Order: 1}))

		mockPool.ExpectQuery("SELECT media_id").WithArgs(int64(99)).
			WillReturnRows(pgxmock.NewRows([]string{"media_id"}).AddRow(int64(5)).AddRow(int64(6)))
		require.Equal(t, []int64{5, 6}, repo.GetMediaByTicketID(context.Background(), 99))
	})

	t.Run("messages", func(t *testing.T) {
		t.Parallel()
		mockPool, repo := newSupportRepo(t)
		now := time.Now()
		mockPool.ExpectQuery("SELECT id, ticket_id, text").WithArgs(int64(99)).
			WillReturnRows(pgxmock.NewRows([]string{"id", "ticket_id", "text", "author_id", "author_role", "created_at"}).
				AddRow(int64(1), int64(99), "hello", int64(10), models.SupportRoleUser, now))
		messages, err := repo.GetMessages(context.Background(), 99)
		require.NoError(t, err)
		require.Len(t, messages, 1)

		message := &models.SupportTicketMessage{TicketID: 99, Text: "hello", AuthorID: 10, AuthorRole: models.SupportRoleUser, CreatedAt: now}
		mockPool.ExpectQuery("INSERT INTO support_ticket_message").WithArgs(message.TicketID, message.Text, message.AuthorID, string(message.AuthorRole), message.CreatedAt).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(2)))
		id, err := repo.SaveMessage(context.Background(), message)
		require.NoError(t, err)
		require.Equal(t, int64(2), id)
		require.Equal(t, int64(2), message.ID)
	})
}
