package worker

import (
	"context"
	"errors"
	"strconv"
	"time"

	esindex "github.com/go-park-mail-ru/2026_1_ARIS/pkg/elasticsearch"
	"github.com/go-park-mail-ru/2026_1_ARIS/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_ARIS/services/indexer/internal/repository"
	"go.uber.org/zap"
)

const (
	batchSize    = 200
	tickInterval = 5 * time.Second
)

type Worker struct {
	outbox  *repository.OutboxRepo
	fetcher *repository.FetcherRepo
	writer  *repository.ESWriter
}

func New(
	outbox *repository.OutboxRepo,
	fetcher *repository.FetcherRepo,
	writer *repository.ESWriter,
) *Worker {
	return &Worker{outbox: outbox, fetcher: fetcher, writer: writer}
}

func (w *Worker) Run(ctx context.Context) {
	logg := logFromCtx(ctx)
	logg.Info("indexer worker started")

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logg.Info("indexer worker stopped")
			return
		case <-ticker.C:
			if err := w.tick(ctx); err != nil {
				logg.Error("indexer tick error", zap.Error(err))
			}
		}
	}
}

func (w *Worker) tick(ctx context.Context) error {
	logg := logFromCtx(ctx)

	cursor, err := w.outbox.GetCursor(ctx)
	if err != nil {
		return err
	}

	events, err := w.outbox.ReadBatch(ctx, cursor, batchSize)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	maxID := events[len(events)-1].ID
	events = dedup(events)

	var bulkItems []repository.BulkItem
	for _, ev := range events {
		item, err := w.resolve(ctx, ev)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				idx := indexForType(ev.EntityType)
				if idx != "" {
					bulkItems = append(bulkItems, repository.DeleteBulkItem(idx, strconv.FormatInt(ev.EntityID, 10)))
				}
				continue
			}
			logg.Warn("resolve entity failed, skipping",
				zap.String("type", ev.EntityType),
				zap.Int64("id", ev.EntityID),
				zap.Error(err))
			continue
		}
		if item != nil {
			bulkItems = append(bulkItems, *item)
		}
	}

	if err := w.writer.Bulk(ctx, bulkItems); err != nil {
		return err
	}

	return w.outbox.Commit(ctx, maxID)
}

func (w *Worker) resolve(ctx context.Context, ev repository.OutboxEvent) (*repository.BulkItem, error) {
	if ev.Operation == "delete" {
		idx := indexForType(ev.EntityType)
		if idx == "" {
			return nil, nil
		}
		item := repository.DeleteBulkItem(idx, strconv.FormatInt(ev.EntityID, 10))
		return &item, nil
	}

	switch ev.EntityType {
	case "user":
		doc, err := w.fetcher.FetchUser(ctx, ev.EntityID)
		if err != nil {
			return nil, err
		}
		item := repository.UserDocToBulkItem(doc)
		return &item, nil

	case "community":
		doc, err := w.fetcher.FetchCommunity(ctx, ev.EntityID)
		if err != nil {
			return nil, err
		}
		item := repository.CommunityDocToBulkItem(doc)
		return &item, nil

	case "post":
		doc, err := w.fetcher.FetchPost(ctx, ev.EntityID)
		if err != nil {
			return nil, err
		}
		item := repository.PostDocToBulkItem(doc)
		return &item, nil
	}

	return nil, nil
}

func indexForType(entityType string) string {
	switch entityType {
	case "user":
		return esindex.IndexUsers
	case "community":
		return esindex.IndexCommunities
	case "post":
		return esindex.IndexPosts
	}
	return ""
}

func logFromCtx(ctx context.Context) *zap.Logger {
	if l := logger.FromContext(ctx); l != nil {
		return l
	}
	l, _ := zap.NewProduction()
	return l
}
