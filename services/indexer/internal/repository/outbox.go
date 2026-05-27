package repository

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type DB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}

type OutboxEvent struct {
	ID         int64
	EntityType string
	EntityID   int64
	Operation  string
}

type OutboxRepo struct {
	db DB
}

func NewOutboxRepo(db DB) *OutboxRepo {
	return &OutboxRepo{db: db}
}

func (r *OutboxRepo) ReadBatch(ctx context.Context, afterID int64, limit int) ([]OutboxEvent, error) {
	start := time.Now()
	rows, err := r.db.Query(ctx, `
		SELECT id, entity_type, entity_id, operation
		FROM search_outbox
		WHERE id > $1
		ORDER BY id ASC
		LIMIT $2
	`, afterID, limit)
	logQuery(ctx, "outbox.ReadBatch", start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []OutboxEvent
	for rows.Next() {
		var e OutboxEvent
		if err := rows.Scan(&e.ID, &e.EntityType, &e.EntityID, &e.Operation); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *OutboxRepo) GetCursor(ctx context.Context) (int64, error) {
	start := time.Now()
	row := r.db.QueryRow(ctx, `SELECT last_processed_id FROM indexer_state WHERE id=1`)
	logQuery(ctx, "outbox.GetCursor", start)

	var cursor int64
	if err := row.Scan(&cursor); err != nil {
		return 0, err
	}
	return cursor, nil
}

func (r *OutboxRepo) Commit(ctx context.Context, cursor int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	start := time.Now()
	if _, err := tx.Exec(ctx, `UPDATE indexer_state SET last_processed_id=$1 WHERE id=1`, cursor); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM search_outbox WHERE id <= $1`, cursor); err != nil {
		return err
	}
	logQuery(ctx, "outbox.Commit", start)
	return tx.Commit(ctx)
}

// SeedIfFirstRun inserts upsert events for all existing active entities when
// the indexer has never run before (last_processed_id == 0 and outbox is empty).
// This covers data that existed before the outbox pattern was introduced.
func (r *OutboxRepo) SeedIfFirstRun(ctx context.Context) (int64, error) {
	cursor, err := r.GetCursor(ctx)
	if err != nil {
		return 0, err
	}
	if cursor != 0 {
		return 0, nil
	}

	var pending int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM search_outbox`).Scan(&pending); err != nil {
		return 0, err
	}
	if pending > 0 {
		return 0, nil
	}

	tag, err := r.db.Exec(ctx, `
		INSERT INTO search_outbox (entity_type, entity_id, operation)
		SELECT 'user', id, 'upsert' FROM user_account WHERE is_active = TRUE
		UNION ALL
		SELECT 'community', id, 'upsert' FROM community WHERE is_active = TRUE
		UNION ALL
		SELECT 'post', id, 'upsert' FROM post WHERE is_active = TRUE
	`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *OutboxRepo) TryAdvisoryLock(ctx context.Context) (bool, error) {
	row := r.db.QueryRow(ctx, `SELECT pg_try_advisory_lock(7654321)`)
	var locked bool
	if err := row.Scan(&locked); err != nil {
		return false, err
	}
	return locked, nil
}

func logQuery(ctx context.Context, name string, start time.Time) {
	logg := logger.FromContext(ctx)
	if logg == nil {
		return
	}
	logg.Debug("db query", zap.String("query", name), zap.Duration("duration", time.Since(start)))
}
