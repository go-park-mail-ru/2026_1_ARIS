#!/usr/bin/env sh
# Populates search_outbox with all existing active entities so the indexer
# syncs them into Elasticsearch. Safe to run multiple times (dedup is handled
# by the indexer's per-entity deduplication logic).
set -eu

: "${DB_USER:?DB_USER is required}"
: "${DB_PASSWORD:?DB_PASSWORD is required}"
: "${DB_NAME:?DB_NAME is required}"
: "${DB_HOST:?DB_HOST is required}"
: "${DB_PORT:?DB_PORT is required}"

PGPASSWORD="$DB_PASSWORD" psql \
  -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<'SQL'
INSERT INTO search_outbox (entity_type, entity_id, operation)
SELECT 'user', id, 'upsert' FROM user_account WHERE is_active = TRUE;

INSERT INTO search_outbox (entity_type, entity_id, operation)
SELECT 'community', id, 'upsert' FROM community WHERE is_active = TRUE;

INSERT INTO search_outbox (entity_type, entity_id, operation)
SELECT 'post', id, 'upsert' FROM post WHERE is_active = TRUE;
SQL

echo "Seeded search_outbox. The indexer will sync to Elasticsearch within 5 seconds."
