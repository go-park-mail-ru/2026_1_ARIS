#!/usr/bin/env sh
set -eu

OUT_DIR="${DB_OBSERVABILITY_OUT_DIR:-.tmp/db-observability}"
PROJECT_NAME="${COMPOSE_PROJECT_NAME:-$(basename "$(pwd)")}"
DATA_VOLUME="${PROJECT_NAME}_aris_postgres_data"

mkdir -p "$OUT_DIR"

docker run --rm \
	-v "${DATA_VOLUME}:/var/lib/postgresql/data:ro" \
	-v "$(pwd)/${OUT_DIR}:/reports" \
	dalibo/pgbadger:latest \
	-f stderr -o /reports/pgbadger.html /var/lib/postgresql/data/log/postgresql.log

echo "pgBadger report written to ${OUT_DIR}/pgbadger.html"
