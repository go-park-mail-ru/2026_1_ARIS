package worker

import "github.com/go-park-mail-ru/2026_1_ARIS/services/indexer/internal/repository"

// dedup collapses a batch of outbox events so only the last event per
// (entity_type, entity_id) pair survives.  If the last event is a delete
// all prior upserts for that entity are discarded.
func dedup(events []repository.OutboxEvent) []repository.OutboxEvent {
	type key struct {
		entityType string
		entityID   int64
	}

	// Track last-seen index for each key.
	last := make(map[key]int, len(events))
	for i, e := range events {
		last[key{e.EntityType, e.EntityID}] = i
	}

	out := make([]repository.OutboxEvent, 0, len(last))
	for i, e := range events {
		if last[key{e.EntityType, e.EntityID}] == i {
			out = append(out, e)
		}
	}
	return out
}
