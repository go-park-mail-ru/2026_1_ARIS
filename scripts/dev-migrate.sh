#!/bin/sh

set -eu

DB_HOST="${DB_HOST:-db}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:?DB_USER is required}"
DB_PASSWORD="${DB_PASSWORD:?DB_PASSWORD is required}"
DB_NAME="${DB_NAME:?DB_NAME is required}"
SSL_MODE="${SSL_MODE:-disable}"
MIGRATIONS_PATH="${MIGRATIONS_PATH:-file://./db/migrations}"
DATABASE_URL="${DATABASE_URL:-postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${SSL_MODE}}"

export PGPASSWORD="$DB_PASSWORD"
export PATH="$(go env GOPATH)/bin:$PATH"

echo "Waiting for PostgreSQL at ${DB_HOST}:${DB_PORT}..."
until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1; do
  sleep 1
done

if ! command -v migrate >/dev/null 2>&1; then
  echo "Installing golang-migrate..."
  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
fi

echo "Applying database migrations..."
set +e
migration_output="$(migrate -source "$MIGRATIONS_PATH" -database "$DATABASE_URL" up 2>&1)"
migration_status=$?
set -e

if [ "$migration_status" -ne 0 ]; then
  case "$migration_output" in
    *"no change"*)
      echo "$migration_output"
      echo "No new database migrations."
      exit 0
      ;;
    *)
      echo "$migration_output" >&2
      exit "$migration_status"
      ;;
  esac
fi

if [ -n "$migration_output" ]; then
  echo "$migration_output"
fi

echo "Database migrations completed."
