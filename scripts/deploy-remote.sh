#!/usr/bin/env sh
set -eu

set -a
. ./.env.server
set +a

: "${APP_ENDPOINT:?APP_ENDPOINT is required}"
: "${IMAGE_REGISTRY:?IMAGE_REGISTRY is required}"
: "${IMAGE_TAG:?IMAGE_TAG is required}"

sh scripts/render-service-envs.sh
sh scripts/render-nginx-server-conf.sh

compose() {
  docker compose --env-file ./.env.server \
    -f ./docker-compose.yml \
    -f ./docker-compose.server.yml \
    --profile microservices \
    "$@"
}

if [ -n "${GHCR_TOKEN:-}" ]; then
  echo "$GHCR_TOKEN" | docker login ghcr.io -u "${GHCR_USER:?GHCR_USER is required when GHCR_TOKEN is set}" --password-stdin
fi

echo "Pulling backend images from ${IMAGE_REGISTRY}..."
compose pull auth media user post chat support community search game indexer

compose up --no-build --force-recreate -d \
  tarantool auth media user post chat support community search game prometheus grafana node-exporter nginx \
  indexer

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
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-nginx-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-elasticsearch-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-indexer-1 .*Up'

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
