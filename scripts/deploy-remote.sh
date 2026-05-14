#!/usr/bin/env sh
set -eu

set -a
. ./.env.server
set +a

: "${APP_ENDPOINT:?APP_ENDPOINT is required}"

sh scripts/render-service-envs.sh
sh scripts/render-nginx-server-conf.sh

docker compose --env-file ./.env.server \
  -f ./docker-compose.yml \
  -f ./docker-compose.server.yml \
  --profile microservices \
  up --build --force-recreate -d \
  auth media user post chat support community search nginx

sleep 15
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-auth-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-user-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-post-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-chat-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-media-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-support-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-community-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-search-1 .*healthy'
docker ps --format '{{.Names}} {{.Status}}' | grep 'arisback-nginx-1 .*healthy'

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

require_2xx '/'
require_2xx '/health'
curl --fail --silent --show-error "$BASE_URL/health" | grep '"status":"ok"'
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
require_not_5xx '/api/communities'
require_not_5xx '/api/communities/'
require_not_5xx '/api/search?q=test'
require_not_5xx '/api/unknown-smoke'
