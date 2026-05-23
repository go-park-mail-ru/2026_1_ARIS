#!/usr/bin/env sh
set -eu

COMPOSE_ENV_FILE="${COMPOSE_ENV_FILE:-./.env}"
COMPOSE_FILE="${COMPOSE_FILE:-./docker-compose.yml}"

eval_tarantool() {
  code="$1"
  docker compose --env-file "$COMPOSE_ENV_FILE" -f "$COMPOSE_FILE" --profile microservices exec -T tarantool tarantool -e "
local net_box = require('net.box')
local conn = assert(net_box.connect('127.0.0.1:3301', {
  user = os.getenv('TARANTOOL_USER') or 'cache',
  password = os.getenv('TARANTOOL_PASSWORD') or 'local-tarantool-password',
}))
local result = conn:eval([[$code]])
print(tostring(result))
conn:close()
os.exit(0)
"
}

print_stat() {
  name="$1"
  code="$2"
  printf '%s=' "$name"
  eval_tarantool "$code"
}

print_stat profile_details 'return box.space.profile_details:count()'
print_stat profile_summaries 'return box.space.profile_summaries:count()'
print_stat auth_users 'return box.space.auth_users:count()'
print_stat profile_id_by_account 'return box.space.profile_id_by_account:count()'
print_stat post_like_counts 'return box.space.post_like_counts:count()'
print_stat presence 'return box.space.presence:count()'
print_stat presence_online 'local total = 0; for _, row in box.space.presence:pairs() do if row.is_online == true or row[2] == true then total = total + 1 end end; return total'
print_stat presence_connections 'local total = 0; for _, row in box.space.presence:pairs() do total = total + (tonumber(row.connections) or tonumber(row[5]) or 0) end; return total'
