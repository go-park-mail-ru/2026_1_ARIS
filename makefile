.PHONY: test lint ci coverage coverage-excluding-mocks clean dev down reset-db logs migrate generate mocks seed tarantool-stats ws-open microservices local-prepare microservices-up microservices-rebuild microservices-monitoring-up microservices-full-up microservices-stop microservices-down microservices-reset local-up local-rebuild local-monitoring-up local-full-up local-stop local-down local-reset server-prepare server-up server-stop server-down server-reset server-logs server-nginx-up server-nginx-stop server-nginx-test server-nginx-reload server-nginx-update server-nginx-install server-host-nginx-test server-host-nginx-reload auth-up auth-stop media-up media-stop user-up user-stop post-up post-stop chat-up chat-stop support-up support-stop community-up community-stop search-up search-stop elasticsearch-up elasticsearch-stop indexer-up indexer-stop nginx-up nginx-stop nginx-test nginx-reload nginx-update nginx-recreate logs-auth logs-media logs-user logs-post logs-chat logs-support logs-community logs-search logs-elasticsearch logs-indexer logs-nginx seed-elasticsearch

COMPOSE_FILE=./docker-compose.yml
COMPOSE_ENV_FILE=./.env
COMPOSE_PARALLEL_LIMIT ?= 2
COMPOSE=COMPOSE_PARALLEL_LIMIT=$(COMPOSE_PARALLEL_LIMIT) docker compose --env-file $(COMPOSE_ENV_FILE) -f $(COMPOSE_FILE)
COMPOSE_SERVER_FILE=./docker-compose.server.yml
COMPOSE_SERVER_ENV_FILE=./.env.server
COMPOSE_SERVER=docker compose --env-file $(COMPOSE_SERVER_ENV_FILE) -f $(COMPOSE_FILE) -f $(COMPOSE_SERVER_FILE)
MICROSERVICE_SERVICES=auth media user post chat support community search game indexer
MICROSERVICE_INFRA=db redis minio tarantool elasticsearch clickhouse
MICROSERVICE_EDGE=nginx
MICROSERVICE_MONITORING=prometheus grafana node-exporter nginx-exporter nginxlog-exporter
MICROSERVICE_INIT=migrate
MICROSERVICE_ALL=$(MICROSERVICE_SERVICES) $(MICROSERVICE_EDGE) $(MICROSERVICE_MONITORING) $(MICROSERVICE_INFRA)
MICROSERVICE_RUNTIME=$(MICROSERVICE_SERVICES) $(MICROSERVICE_EDGE)
MICROSERVICE_RUNTIME_FULL=$(MICROSERVICE_RUNTIME) $(MICROSERVICE_MONITORING)
NGINX_SITE_NAME ?= arisnet.ru
NGINX_SITES_AVAILABLE ?= /etc/nginx/sites-available
NGINX_SITES_ENABLED ?= /etc/nginx/sites-enabled
COVER_PACKAGE_EXCLUDE_PATTERN ?= "(/mock($$|/)|/mocks($$|/)|/cmd($$|/)|/proto($$|/)|/tools($$|/)|/internal/dto$$|/internal/xerrors$$|^github.com/go-park-mail-ru/2026_1_ARIS/common$$|/pkg/(clickhouse|minio|redis|tarantool)$$|/services/indexer($$|/)|/internal/websocket$$|/services/post/internal/analytics$$)"
COVER_TEST_PACKAGES ?= $(shell go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./... | grep -v -E $(COVER_PACKAGE_EXCLUDE_PATTERN))
COVER_PACKAGES ?= $(shell go list -f '{{if .GoFiles}}{{.ImportPath}}{{end}}' ./... | grep -v -E $(COVER_PACKAGE_EXCLUDE_PATTERN) | paste -sd, -)
COVER_EXCLUDE_PATTERN ?= "(/mock/|/mocks/|_mock\.go|_easyjson\.go|\.pb\.go|_grpc\.pb\.go|(^|/)dto\.go|(^|/)main\.go)"
COVER_MERGE_AWK = 'NR==1 { print; next } { key=$$1 " " $$2; if (!(key in seen) || $$3 > count[key]) { line[key]=$$0; count[key]=$$3; seen[key]=1 } } END { for (key in line) print line[key] }'

DB_HOST ?= 127.0.0.1
DB_PORT ?= 5431
DB_USER ?= kokinside
DB_PASSWORD ?= password1
DB_NAME ?= ARIS-DB
SSL_MODE ?= disable
MIGRATIONS_PATH ?= file://./db/migrations
DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(SSL_MODE)
MIGRATE=migrate -source "$(MIGRATIONS_PATH)" -database "$(DATABASE_URL)"
GOCACHE ?= $(CURDIR)/.cache/go-build
STATICCHECK_CACHE ?= $(CURDIR)/.cache/staticcheck
STATICCHECK ?= go tool staticcheck

test:
	GOCACHE="$(GOCACHE)" go test -v ./...

lint:
	@fmt_files="$$(gofmt -l .)"; \
	if [ -n "$$fmt_files" ]; then \
		echo "gofmt is required for these files:"; \
		printf '%s\n' "$$fmt_files"; \
		exit 1; \
	fi
	GOCACHE="$(GOCACHE)" go vet ./...
	GOCACHE="$(GOCACHE)" STATICCHECK_CACHE="$(STATICCHECK_CACHE)" $(STATICCHECK) ./...

ci: lint test

generate:
	go generate ./...

mocks:
	go generate ./...

clean:
	go clean -testcache
	rm -f ./coverage.out ./coverage.out.tmp
	touch ./coverage.out
	touch ./coverage.out.tmp

coverage: clean
	go test -count=1 $(COVER_TEST_PACKAGES) -coverprofile=coverage.out.tmp -coverpkg=$(COVER_PACKAGES)
	grep -v -E $(COVER_EXCLUDE_PATTERN) coverage.out.tmp | awk $(COVER_MERGE_AWK) > coverage.out
	go tool cover -func=coverage.out | grep total

migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down

migrate-version:
	${MIGRATE} version

migrate-force-down:
	${MIGRATE} force 1

migrate:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

microservices: microservices-up

local-prepare:
	set -a; . $(COMPOSE_ENV_FILE); set +a; sh scripts/render-service-envs.sh

local-up: local-prepare microservices-up
local-rebuild: local-prepare microservices-rebuild
local-monitoring-up: microservices-monitoring-up
local-full-up: local-prepare microservices-full-up
local-stop: microservices-stop
local-down: microservices-down
local-reset: microservices-reset

seed: local-prepare
	$(COMPOSE) up seed

seed-elasticsearch:
	set -a; . $(COMPOSE_ENV_FILE); set +a; sh scripts/seed-elasticsearch.sh

tarantool-stats:
	sh scripts/tarantool-stats.sh

ws-open:
	GOCACHE="$(CURDIR)/.cache/go-build" go run ./tools/ws-open/cmd --base-url "$${WS_BASE_URL:-ws://localhost:18080}" --chat-id "$${CHAT_ID:-1}" --cookie-file "$${COOKIE_FILE:-/tmp/aris-cookies.txt}" --duration "$${DURATION:-0}"

microservices-up: local-prepare
	$(COMPOSE) --profile microservices up --pull never -d $(MICROSERVICE_RUNTIME)
	$(COMPOSE) --profile microservices up --pull never --force-recreate -d nginx

microservices-rebuild: local-prepare
	$(COMPOSE) --profile microservices up --build --pull never -d $(MICROSERVICE_RUNTIME)
	$(COMPOSE) --profile microservices up -d $(MICROSERVICE_MONITORING)
	$(COMPOSE) --profile microservices up --pull never --force-recreate -d nginx

microservices-monitoring-up:
	$(COMPOSE) --profile microservices up -d $(MICROSERVICE_MONITORING)

microservices-full-up: local-prepare
	$(COMPOSE) --profile microservices up -d $(MICROSERVICE_RUNTIME_FULL)

microservices-stop:
	$(COMPOSE) stop $(MICROSERVICE_ALL)

microservices-down:
	$(COMPOSE) stop $(MICROSERVICE_ALL)
	$(COMPOSE) rm -f $(MICROSERVICE_ALL) $(MICROSERVICE_INIT)

microservices-reset:
	$(COMPOSE) --profile microservices down -v

server-prepare:
	set -a; . $(COMPOSE_SERVER_ENV_FILE); set +a; sh scripts/render-service-envs.sh
	set -a; . $(COMPOSE_SERVER_ENV_FILE); set +a; sh scripts/render-nginx-server-conf.sh

server-up: server-prepare
	$(COMPOSE_SERVER) --profile microservices up --build -d $(MICROSERVICE_RUNTIME)

server-stop:
	$(COMPOSE_SERVER) stop $(MICROSERVICE_ALL)

server-down:
	$(COMPOSE_SERVER) stop $(MICROSERVICE_ALL)
	$(COMPOSE_SERVER) rm -f $(MICROSERVICE_ALL) $(MICROSERVICE_INIT)

server-reset:
	$(COMPOSE_SERVER) --profile microservices down -v

server-logs:
	$(COMPOSE_SERVER) logs -f $(MICROSERVICE_RUNTIME)

server-nginx-up: server-prepare
	$(COMPOSE_SERVER) --profile microservices up -d nginx

server-nginx-stop:
	$(COMPOSE_SERVER) stop nginx

server-nginx-test:
	$(COMPOSE_SERVER) exec -T nginx nginx -t

server-nginx-reload:
	$(COMPOSE_SERVER) exec -T nginx nginx -t
	$(COMPOSE_SERVER) exec -T nginx nginx -s reload

server-nginx-update: server-prepare
	$(COMPOSE_SERVER) --profile microservices up -d nginx
	$(COMPOSE_SERVER) exec -T nginx nginx -t
	$(COMPOSE_SERVER) exec -T nginx nginx -s reload

server-nginx-install:
	sudo mkdir -p $(NGINX_SITES_AVAILABLE) $(NGINX_SITES_ENABLED)
	sudo install -m 0644 ./nginx/config/nginx.server.conf $(NGINX_SITES_AVAILABLE)/$(NGINX_SITE_NAME)
	sudo ln -sfn $(NGINX_SITES_AVAILABLE)/$(NGINX_SITE_NAME) $(NGINX_SITES_ENABLED)/$(NGINX_SITE_NAME)
	sudo nginx -t

server-host-nginx-test:
	sudo nginx -t

server-host-nginx-reload:
	sudo nginx -t
	sudo systemctl reload nginx

auth-up: local-prepare
	$(COMPOSE) up --build -d auth

auth-stop:
	$(COMPOSE) stop auth

media-up: local-prepare
	$(COMPOSE) up --build -d media

media-stop:
	$(COMPOSE) stop media

user-up: local-prepare
	$(COMPOSE) up --build -d user

user-stop:
	$(COMPOSE) stop user

post-up: local-prepare
	$(COMPOSE) up --build -d post

post-stop:
	$(COMPOSE) stop post

chat-up: local-prepare
	$(COMPOSE) up --build -d chat

chat-stop:
	$(COMPOSE) stop chat

support-up: local-prepare
	$(COMPOSE) up --build -d support

support-stop:
	$(COMPOSE) stop support

community-up: local-prepare
	$(COMPOSE) up --build -d community

community-stop:
	$(COMPOSE) stop community

search-up: local-prepare
	$(COMPOSE) up --build -d search

search-stop:
	$(COMPOSE) stop search

elasticsearch-up:
	$(COMPOSE) --profile microservices up -d elasticsearch

elasticsearch-stop:
	$(COMPOSE) stop elasticsearch

indexer-up: local-prepare
	$(COMPOSE) up --build -d indexer

indexer-stop:
	$(COMPOSE) stop indexer

nginx-up: local-prepare
	$(COMPOSE) --profile microservices up -d nginx

nginx-stop:
	$(COMPOSE) stop nginx

nginx-test:
	$(COMPOSE) exec -T nginx nginx -t

nginx-reload:
	$(COMPOSE) exec -T nginx nginx -t
	$(COMPOSE) exec -T nginx nginx -s reload

nginx-update:
	$(COMPOSE) --profile microservices up -d --force-recreate nginx

nginx-recreate: local-prepare
	$(COMPOSE) --profile microservices up -d --force-recreate nginx

logs-auth:
	$(COMPOSE) logs -f auth

logs-media:
	$(COMPOSE) logs -f media

logs-user:
	$(COMPOSE) logs -f user

logs-post:
	$(COMPOSE) logs -f post

logs-chat:
	$(COMPOSE) logs -f chat

logs-support:
	$(COMPOSE) logs -f support

logs-community:
	$(COMPOSE) logs -f community

logs-search:
	$(COMPOSE) logs -f search

logs-elasticsearch:
	$(COMPOSE) logs -f elasticsearch

logs-indexer:
	$(COMPOSE) logs -f indexer

logs-nginx:
	$(COMPOSE) logs -f nginx

dev: local-up

down: local-down

stop: local-stop

reset-db:
	$(MAKE) microservices-reset
	$(MAKE) microservices-up

logs:
	$(COMPOSE) logs -f $(MICROSERVICE_RUNTIME)

logs-redis:
	$(COMPOSE) logs -f redis

logs-db:
	$(COMPOSE) logs -f db

logs-migrate:
	$(COMPOSE) logs -f migrate
	
coverage-excluding-mocks: clean
	go test -count=1 $(COVER_TEST_PACKAGES) -coverprofile=coverage.out.tmp -coverpkg=$(COVER_PACKAGES)
	grep -v -E $(COVER_EXCLUDE_PATTERN) coverage.out.tmp > coverage.out
	go tool cover -func=coverage.out | grep total
