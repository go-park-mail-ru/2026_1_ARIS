#!/usr/bin/env sh
set -eu

# Preserve IMAGE_TAG passed from CI before .env.server potentially overwrites it
_IMAGE_TAG="${IMAGE_TAG:-}"

set -a
. ./.env.server
set +a

export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
export DBUS_SESSION_BUS_ADDRESS="${DBUS_SESSION_BUS_ADDRESS:-unix:path=${XDG_RUNTIME_DIR}/bus}"

if [ -n "$_IMAGE_TAG" ]; then
  export IMAGE_TAG="$_IMAGE_TAG"
fi

: "${APP_ENDPOINT:?APP_ENDPOINT is required}"
DEPLOY_LOCK_FILE="${DEPLOY_LOCK_FILE:-/tmp/arisback-deploy.lock}"
if command -v flock >/dev/null 2>&1; then
  exec 9>"$DEPLOY_LOCK_FILE"
  if ! flock -n 9; then
    echo "Another deploy is already running; exiting." >&2
    exit 1
  fi
else
  DEPLOY_LOCK_DIR="${DEPLOY_LOCK_DIR:-/tmp/arisback-deploy.lock.d}"
  if ! mkdir "$DEPLOY_LOCK_DIR" 2>/dev/null; then
    echo "Another deploy is already running; exiting." >&2
    exit 1
  fi
  trap 'rmdir "$DEPLOY_LOCK_DIR"' EXIT INT TERM
fi

export COMPOSE_PARALLEL_LIMIT="${COMPOSE_PARALLEL_LIMIT:-1}"
deploy_nginx_container="${DEPLOY_NGINX_CONTAINER:-1}"

APP_SERVICES="auth media user post chat support community search game indexer"
INFRA_SERVICES="db redis minio tarantool elasticsearch clickhouse"
MONITORING_SERVICES="${MONITORING_SERVICES:-prometheus grafana node-exporter nginx-exporter nginxlog-exporter}"
RUN_SEED_ON_DEPLOY="${RUN_SEED_ON_DEPLOY:-0}"
AUTO_SEED_ON_EMPTY="${AUTO_SEED_ON_EMPTY:-1}"

sh scripts/render-service-envs.sh
sh scripts/render-nginx-server-conf.sh

compose() {
  docker compose --env-file ./.env.server \
    -f ./docker-compose.yml \
    -f ./docker-compose.server.yml \
    --profile microservices \
    "$@"
}

# GHCR_ACTOR приходит из CI; GHCR_USER — старый формат из .env.server (staging)
_ghcr_user="${GHCR_ACTOR:-${GHCR_USER:-}}"
if [ -n "${GHCR_TOKEN:-}" ] && [ -n "$_ghcr_user" ]; then
  echo "Logging in to ghcr.io as ${_ghcr_user}..."
  echo "$GHCR_TOKEN" | docker login ghcr.io -u "$_ghcr_user" --password-stdin
fi

echo "Starting persistent infrastructure..."
compose up --no-build --no-recreate -d $INFRA_SERVICES

echo "Applying database migrations..."
compose up --no-build --force-recreate --no-deps migrate

seed_marker_count() {
  compose exec -T db sh -c "psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -Atqc \"SELECT COUNT(*) FROM user_account WHERE username IN ('demoowner', 'komandaaris');\""
}

refresh_search_outbox() {
  compose exec -T db sh -c "psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -c \"
INSERT INTO search_outbox (entity_type, entity_id, operation)
SELECT 'user', id, 'upsert'
FROM user_account
WHERE is_active = TRUE;

INSERT INTO search_outbox (entity_type, entity_id, operation)
SELECT 'community', id, 'upsert'
FROM community
WHERE is_active = TRUE;

INSERT INTO search_outbox (entity_type, entity_id, operation)
SELECT 'post', id, 'upsert'
FROM post
WHERE is_active = TRUE;
\""
}

wait_search_outbox_drained() {
  attempts="${SEARCH_OUTBOX_WAIT_ATTEMPTS:-24}"
  while [ "$attempts" -gt 0 ]; do
    pending="$(compose exec -T db sh -c "psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -Atqc \"SELECT COUNT(*) FROM search_outbox;\"")"
    if [ "$pending" = "0" ]; then
      echo "Search outbox processed."
      return 0
    fi
    echo "Waiting for search outbox to be processed (${pending} pending)..."
    attempts=$((attempts - 1))
    sleep 5
  done

  echo "Search outbox was not fully processed in time." >&2
  return 1
}

should_run_seed="$RUN_SEED_ON_DEPLOY"
if [ "$should_run_seed" != "1" ] && [ "$AUTO_SEED_ON_EMPTY" = "1" ]; then
  marker_count="$(seed_marker_count || printf '0')"
  case "$marker_count" in
    ''|*[!0-9]*)
      echo "Could not determine seed marker count (${marker_count}); skipping automatic seed."
      ;;
    0)
      echo "Seed marker users are absent; seed will run automatically."
      should_run_seed="1"
      ;;
    *)
      echo "Seed marker users found (${marker_count}); skipping seed."
      ;;
  esac
fi

if [ "$should_run_seed" = "1" ]; then
  echo "Running seed data refresh..."
  compose up --no-build --force-recreate --no-deps seed
fi

echo "Pulling backend images (tag: ${IMAGE_TAG:-latest})..."
compose pull $APP_SERVICES

echo "Updating backend runtime services..."
compose up --no-build --remove-orphans -d $APP_SERVICES

echo "Starting monitoring and edge services..."
compose up --no-build --force-recreate -d $MONITORING_SERVICES
if [ "$deploy_nginx_container" = "1" ]; then
  compose up --no-build --force-recreate -d nginx
else
  echo "Skipping Docker nginx service because DEPLOY_NGINX_CONTAINER=$deploy_nginx_container"
fi

if systemctl --user is-active --quiet arisfront 2>/dev/null || systemctl --user is-enabled --quiet arisfront 2>/dev/null; then
  echo "Restarting frontend service..."
  systemctl --user restart arisfront
fi

sleep 15
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisnet-tarantool .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-auth-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-user-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-post-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-chat-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-media-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-support-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-community-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-search-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-game-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-prometheus-1 .*Up'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-grafana-1 .*Up'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-node-exporter-1 .*Up'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-nginx-exporter-1 .*Up'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-nginxlog-exporter-1 .*Up'
if [ "$deploy_nginx_container" = "1" ]; then
  docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-nginx-1 .*healthy'
fi
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-elasticsearch-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-indexer-1 .*Up'

echo "Refreshing Elasticsearch index queue..."
refresh_search_outbox
wait_search_outbox_drained

BASE_URL="$APP_ENDPOINT"

require_2xx() {
  path="$1"
  code="$(curl --output /dev/null --silent --show-error --max-time 15 --write-out '%{http_code}' "$BASE_URL$path")"
  case "$code" in
    2*) echo "[PASS] $path -> $code" ;;
    *) echo "[FAIL] $path -> $code, expected 2xx" >&2; exit 1 ;;
  esac
}

require_not_5xx() {
  path="$1"
  code="$(curl --output /dev/null --silent --show-error --max-time 15 --write-out '%{http_code}' "$BASE_URL$path")"
  case "$code" in
    000|5*) echo "[FAIL] $path -> $code" >&2; exit 1 ;;
    *) echo "[PASS] $path -> $code" ;;
  esac
}

require_status() {
  path="$1"
  expected="$2"
  code="$(curl --output /dev/null --silent --show-error --max-time 15 --write-out '%{http_code}' "$BASE_URL$path")"
  if [ "$code" = "$expected" ]; then
    echo "[PASS] $path -> $code"
  else
    echo "[FAIL] $path -> $code, expected $expected" >&2
    exit 1
  fi
}

require_2xx '/'
require_2xx '/health'
curl --fail --silent --show-error "$BASE_URL/health" | grep '"status":"ok"'
require_2xx '/metrics/api/health'
require_2xx '/prometheus/-/healthy'
require_2xx '/api/public/feed?limit=1'
require_not_5xx '/api/auth'
require_not_5xx '/api/auth/'
require_not_5xx '/api/public/popular-users'
require_not_5xx '/api/users/'
require_not_5xx '/api/profile/'
require_not_5xx '/api/friends/'
require_not_5xx '/api/settings/'
require_not_5xx '/api/public/popular-posts'
require_not_5xx '/api/feed'
require_not_5xx '/api/posts/popular'
require_not_5xx '/api/post'
require_not_5xx '/api/post/'
require_not_5xx '/api/media'
require_not_5xx '/api/media/'
require_not_5xx '/api/chats'
require_not_5xx '/api/chats/'
require_not_5xx '/api/support'
require_not_5xx '/api/support/'
require_status '/api/support/tickets/my' 401
require_not_5xx '/api/communities'
require_not_5xx '/api/communities/'
require_not_5xx '/api/search?q=test'
require_not_5xx '/api/games'
require_not_5xx '/api/games/'
require_not_5xx '/api/unknown-smoke'
