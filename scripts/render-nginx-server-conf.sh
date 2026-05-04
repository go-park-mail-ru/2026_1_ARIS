#!/usr/bin/env sh
set -eu

template="${1:-config/nginx.container.server.conf.template}"
output="${2:-config/nginx.container.server.conf}"

endpoint_host="$(
  printf '%s' "${APP_ENDPOINT:-}" \
    | sed -E 's#^[a-zA-Z][a-zA-Z0-9+.-]*://##; s#/.*$##; s#:.*$##'
)"
default_cert_name="${endpoint_host:-${NGINX_SITE_NAME:-arisnet.ru}}"
case "$default_cert_name" in
  arisnet.ru) default_server_name="arisnet.ru www.arisnet.ru" ;;
  *) default_server_name="$default_cert_name" ;;
esac
server_name="${NGINX_SERVER_NAME:-$default_server_name}"
cert_name="${NGINX_CERT_NAME:-$default_cert_name}"
health_path="${NGINX_HEALTH_PATH:-/nginx-health}"
frontend_upstream="${FRONTEND_UPSTREAM:-host.docker.internal:3001}"

if [ ! -f "$template" ]; then
  echo "nginx template not found: $template" >&2
  exit 1
fi

case "$health_path" in
  /*) ;;
  *)
    echo "NGINX_HEALTH_PATH must start with /" >&2
    exit 1
    ;;
esac

sed \
  -e "s|\${NGINX_SERVER_NAME}|$server_name|g" \
  -e "s|\${NGINX_CERT_NAME}|$cert_name|g" \
  -e "s|\${NGINX_HEALTH_PATH}|$health_path|g" \
  -e "s|\${FRONTEND_UPSTREAM}|$frontend_upstream|g" \
  "$template" > "$output"
