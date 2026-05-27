package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	models "github.com/go-park-mail-ru/2026_1_ARIS/services/support/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

// mockTicketDB is a minimal ticketDB / DB implementation that always returns errors.
type mockTicketDB struct {
	row pgx.Row
	err error
}

func (m *mockTicketDB) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return nil, m.err
}
func (m *mockTicketDB) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return m.row
}
func (m *mockTicketDB) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, m.err
}

// mockErrRow is a pgx.Row that always returns an error on Scan.
type mockErrRow struct{ err error }

func (r *mockErrRow) Scan(_ ...any) error { return r.err }

func TestTicketRepositoryReturnsDBErrors(t *testing.T) {
	t.Parallel()

	dbErr := errors.New("db down")
	db := &mockTicketDB{
		row: &mockErrRow{err: dbErr},
		err: dbErr,
	}

	ctx := context.Background()
	store := NewTicketStorage(db)

	now := time.Now()
	ticket := models.NewSupportTicket(1, "login", "email@example.com", models.CategoryBug, "title", "desc")

	// Save uses QueryRow (scan for RETURNING id)
	_, err := store.Save(ctx, ticket)
	require.Error(t, err)

	// GetByID uses QueryRow
	_, err = store.GetByID(ctx, 1)
	require.Error(t, err)

	// GetByIDAndProfileID calls GetByID internally
	_, err = store.GetByIDAndProfileID(ctx, 1, 1)
	require.Error(t, err)

	// GetAll uses Query
	_, err = store.GetAll(ctx, TicketFilter{})
	require.Error(t, err)

	// GetByProfileID uses Query
	_, err = store.GetByProfileID(ctx, 1)
	require.Error(t, err)

	// Update uses QueryRow
	_, err = store.Update(ctx, ticket)
	require.Error(t, err)

	// UpdateStatusByID uses QueryRow
	_, err = store.UpdateStatusByID(ctx, 1, models.TicketStatusClosed, nil, now)
	require.Error(t, err)

	// UpdateStatus uses QueryRow
	_, err = store.UpdateStatus(ctx, 1, 1, models.TicketStatusClosed, nil, now)
	require.Error(t, err)

	// Assign uses QueryRow
	_, err = store.Assign(ctx, 1, 2, now)
	require.Error(t, err)

	// Escalate uses QueryRow
	_, err = store.Escalate(ctx, 1, now)
	require.Error(t, err)

	// Rate uses QueryRow
	_, err = store.Rate(ctx, 1, 1, 5, now)
	require.Error(t, err)

	// GetStats uses QueryRow for initial scan
	_, err = store.GetStats(ctx)
	require.Error(t, err)

	// SetProfileRole uses Exec
	err = store.SetProfileRole(ctx, 1, models.SupportRoleAdmin)
	require.Error(t, err)

	// GetProfileRole uses QueryRow
	_, err = store.GetProfileRole(ctx, 1)
	require.Error(t, err)

	// GetMessages uses Query
	_, err = store.GetMessages(ctx, 1)
	require.Error(t, err)

	// SaveMessage uses QueryRow
	message := &models.SupportTicketMessage{
		TicketID:   1,
		Text:       "hello",
		AuthorID:   1,
		AuthorRole: models.SupportRoleUser,
		CreatedAt:  now,
	}
	_, err = store.SaveMessage(ctx, message)
	require.Error(t, err)

	// SaveMedia uses Exec
	err = store.SaveMedia(ctx, models.TicketWithMedia{TicketID: 1, MediaID: 2, Order: 0})
	require.Error(t, err)

	// GetMediaByTicketID returns nil on Query error (no error return)
	ids := store.GetMediaByTicketID(ctx, 1)
	require.Nil(t, ids)
}

func TestProfileRoleRepositoryReturnsDBErrors(t *testing.T) {
	t.Parallel()

	dbErr := errors.New("db down")
	db := &mockTicketDB{
		row: &mockErrRow{err: dbErr},
		err: dbErr,
	}

	ctx := context.Background()
	store := NewProfileRoleStorage(db)

	// SetProfileRole uses Exec
	err := store.SetProfileRole(ctx, 1, models.SupportRoleAdmin)
	require.Error(t, err)

	// GetProfileRole uses QueryRow
	_, err = store.GetProfileRole(ctx, 1)
	require.Error(t, err)
}
