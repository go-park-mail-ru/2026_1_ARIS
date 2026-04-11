#!/bin/sh

set -eu

ENV_FILE="${1:-./.env.compose}"
TIMEOUT_SECONDS="${2:-120}"

if [ ! -f "$ENV_FILE" ]; then
  echo "Env file '$ENV_FILE' not found."
  exit 1
fi

set -a
. "$ENV_FILE"
set +a

FRONTEND_HOST="${FRONTEND_HOST:-127.0.0.1}"
BACKEND_HOST="${BACKEND_HOST:-127.0.0.1}"
MINIO_HOST="${MINIO_HOST:-127.0.0.1}"
DB_HOST_LOCAL="${DB_HOST_LOCAL:-127.0.0.1}"

FRONTEND_PORT="${FRONTEND_PORT:-3001}"
BACKEND_PORT="${BACKEND_PORT:-8080}"
MINIO_CONSOLE_PORT="${MINIO_CONSOLE_PORT:-9001}"
DB_PORT="${DB_PORT:-5431}"

frontend_ready() {
  curl -fsS "http://${FRONTEND_HOST}:${FRONTEND_PORT}/" >/dev/null 2>&1
}

backend_ready() {
  nc -z "${BACKEND_HOST}" "${BACKEND_PORT}" >/dev/null 2>&1
}

elapsed=0
while [ "$elapsed" -lt "$TIMEOUT_SECONDS" ]; do
  if backend_ready && frontend_ready; then
    cat <<EOF

ARIS dev environment is ready:
Frontend: http://${FRONTEND_HOST}:${FRONTEND_PORT}
Backend:  http://${BACKEND_HOST}:${BACKEND_PORT}
MinIO:    http://${MINIO_HOST}:${MINIO_CONSOLE_PORT}
Postgres: ${DB_HOST_LOCAL}:${DB_PORT}

You can start testing now.
To follow logs: make logs
EOF
    exit 0
  fi

  sleep 2
  elapsed=$((elapsed + 2))
done

cat <<EOF

Services are still starting or did not become ready within ${TIMEOUT_SECONDS}s.
Check logs with:
make logs
EOF

exit 1
