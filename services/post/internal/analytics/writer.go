package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"go.uber.org/zap"
)

const (
	channelCap    = 10_000
	batchSize     = 1_000
	flushInterval = time.Second
	maxRetries    = 3
)

type Writer struct {
	conn   driver.Conn
	events chan PostEvent
	snaps  chan PostSnapshot
	log    *zap.Logger
}

func NewWriter(conn driver.Conn, log *zap.Logger) *Writer {
	return &Writer{
		conn:   conn,
		events: make(chan PostEvent, channelCap),
		snaps:  make(chan PostSnapshot, channelCap),
		log:    log,
	}
}

// Run запускается как горутина; блокирует до закрытия ctx.
func (w *Writer) Run(ctx context.Context) {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	eventBuf := make([]PostEvent, 0, batchSize)
	snapBuf := make([]PostSnapshot, 0, batchSize)

	flush := func() {
		if len(eventBuf) > 0 {
			w.flushEvents(ctx, eventBuf)
			eventBuf = eventBuf[:0]
		}
		if len(snapBuf) > 0 {
			w.flushSnapshots(ctx, snapBuf)
			snapBuf = snapBuf[:0]
		}
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case e := <-w.events:
			eventBuf = append(eventBuf, e)
			if len(eventBuf) >= batchSize {
				w.flushEvents(ctx, eventBuf)
				eventBuf = eventBuf[:0]
			}
		case s := <-w.snaps:
			snapBuf = append(snapBuf, s)
			if len(snapBuf) >= batchSize {
				w.flushSnapshots(ctx, snapBuf)
				snapBuf = snapBuf[:0]
			}
		case <-ticker.C:
			flush()
		}
	}
}

// WriteEvent — неблокирующий push. Если канал полон: drop + warn.
func (w *Writer) WriteEvent(e PostEvent) {
	select {
	case w.events <- e:
	default:
		w.log.Warn("analytics: event dropped, channel full",
			zap.String("type", string(e.Type)),
			zap.Int64("post_id", e.PostID))
	}
}

// WriteSnapshot — неблокирующий push для снапшотов.
func (w *Writer) WriteSnapshot(s PostSnapshot) {
	select {
	case w.snaps <- s:
	default:
		w.log.Warn("analytics: snapshot dropped, channel full",
			zap.Int64("post_id", s.PostID))
	}
}

// SyncSnapshotsIfEmpty bulk-inserts snapshots into post_snapshot only when the
// table is empty. Safe to call on every startup: becomes a no-op once data
// exists, so restarts never overwrite accumulated analytics data.
// Returns (true, nil) when snapshots were inserted, (false, nil) when skipped.
func (w *Writer) SyncSnapshotsIfEmpty(ctx context.Context, snapshots []PostSnapshot) (bool, error) {
	var count uint64
	if err := w.conn.QueryRow(ctx, "SELECT count() FROM post_snapshot").Scan(&count); err != nil {
		return false, fmt.Errorf("check post_snapshot count: %w", err)
	}
	if count > 0 {
		return false, nil
	}
	if len(snapshots) == 0 {
		return false, nil
	}
	const batchLimit = 1000
	for i := 0; i < len(snapshots); i += batchLimit {
		end := i + batchLimit
		if end > len(snapshots) {
			end = len(snapshots)
		}
		if err := w.insertSnapshots(ctx, snapshots[i:end]); err != nil {
			return false, fmt.Errorf("insert snapshot batch starting at %d: %w", i, err)
		}
	}
	return true, nil
}

func (w *Writer) flushEvents(ctx context.Context, batch []PostEvent) {
	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := w.insertEvents(ctx, batch); err == nil {
			return
		} else if attempt == maxRetries-1 {
			w.log.Error("analytics: failed to insert events after retries",
				zap.Error(err), zap.Int("count", len(batch)))
		}
	}
}

func (w *Writer) flushSnapshots(ctx context.Context, batch []PostSnapshot) {
	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := w.insertSnapshots(ctx, batch); err == nil {
			return
		} else if attempt == maxRetries-1 {
			w.log.Error("analytics: failed to insert snapshots after retries",
				zap.Error(err), zap.Int("count", len(batch)))
		}
	}
}

func (w *Writer) insertEvents(ctx context.Context, batch []PostEvent) error {
	b, err := w.conn.PrepareBatch(ctx, "INSERT INTO post_events")
	if err != nil {
		return err
	}
	for _, e := range batch {
		if err := b.Append(
			e.EventTime,
			e.EventTime,
			e.ProfileID,
			e.PostID,
			e.AuthorProfileID,
			e.CommunityID,
			string(e.Type),
			e.Source,
			e.DwellMs,
			e.Position,
			"",
		); err != nil {
			return err
		}
	}
	return b.Send()
}

func (w *Writer) insertSnapshots(ctx context.Context, batch []PostSnapshot) error {
	b, err := w.conn.PrepareBatch(ctx, "INSERT INTO post_snapshot")
	if err != nil {
		return err
	}
	for _, s := range batch {
		version := uint64(s.UpdatedAt.UnixMilli())
		if err := b.Append(
			s.PostID,
			s.AuthorProfileID,
			s.CommunityID,
			s.IsPublicDemo,
			s.IsActive,
			s.AllowComments,
			s.CreatedAt,
			s.TextLength,
			s.HasMedia,
			version,
		); err != nil {
			return err
		}
	}
	return b.Send()
}
