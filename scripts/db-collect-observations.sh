#!/usr/bin/env sh
set -eu

ENV_FILE="${COMPOSE_ENV_FILE:-./.env.compose}"
COMPOSE_FILE="${COMPOSE_FILE:-./docker-compose.dev.yml}"
OUT_DIR="${DB_OBSERVABILITY_OUT_DIR:-.tmp/db-observability}"

if [ ! -f "$ENV_FILE" ]; then
	echo "env file not found: $ENV_FILE" >&2
	exit 1
fi

set -a
. "$ENV_FILE"
set +a

: "${DB_NAME:?DB_NAME is required in $ENV_FILE}"
: "${DB_USER:?DB_USER is required in $ENV_FILE}"

DB_ADMIN="${DB_ADMIN_USER:-$DB_USER}"

mkdir -p "$OUT_DIR"

run_check() {
	name="$1"
	sql_file="$2"
	docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T db \
		psql -v ON_ERROR_STOP=1 -U "$DB_ADMIN" -d "$DB_NAME" \
		-f "/workspace/$sql_file" >"$OUT_DIR/$name.txt"
}

docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T db \
	mkdir -p /workspace/db/checks
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" cp \
	./db/checks/. db:/workspace/db/checks

run_check "access" "db/checks/access.sql"
run_check "observability" "db/checks/observability.sql"
run_check "symptoms" "db/checks/symptoms.sql"

docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" logs --tail=200 db >"$OUT_DIR/postgres.log"
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T postgres-exporter \
	wget -qO- http://127.0.0.1:9187/metrics >"$OUT_DIR/postgres-exporter.prom"

cat >"$OUT_DIR/README.txt" <<EOF
ARIS DB observability artifacts

Generated files:
- access.txt: roles, grants and effective privileges.
- observability.txt: PostgreSQL logging/statistics settings and pg_stat_statements sample.
- symptoms.txt: active queries, blocking locks, deadlocks, temp files and table/index stats.
- postgres.log: recent PostgreSQL logs, including slow statements and lock waits when present.
- postgres-exporter.prom: raw Prometheus metrics exported by postgres-exporter.
EOF

echo "DB observability artifacts written to $OUT_DIR"
