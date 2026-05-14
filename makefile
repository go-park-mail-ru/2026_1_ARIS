.PHONY: test coverage clean dev down reset-db logs migrate mocks microservices local-prepare microservices-up microservices-rebuild microservices-monitoring-up microservices-full-up microservices-stop microservices-down microservices-reset local-up local-rebuild local-monitoring-up local-full-up local-stop local-down local-reset server-prepare server-up server-stop server-down server-reset server-logs server-nginx-up server-nginx-stop server-nginx-test server-nginx-reload server-nginx-update server-nginx-install server-host-nginx-test server-host-nginx-reload auth-up auth-stop media-up media-stop user-up user-stop post-up post-stop chat-up chat-stop support-up support-stop community-up community-stop search-up search-stop nginx-up nginx-stop nginx-test nginx-reload nginx-update logs-auth logs-media logs-user logs-post logs-chat logs-support logs-community logs-search logs-nginx

COMPOSE_FILE=./docker-compose.yml
COMPOSE_ENV_FILE=./.env
COMPOSE_PARALLEL_LIMIT ?= 2
COMPOSE=COMPOSE_PARALLEL_LIMIT=$(COMPOSE_PARALLEL_LIMIT) docker compose --env-file $(COMPOSE_ENV_FILE) -f $(COMPOSE_FILE)
COMPOSE_SERVER_FILE=./docker-compose.server.yml
COMPOSE_SERVER_ENV_FILE=./.env.server
COMPOSE_SERVER=docker compose --env-file $(COMPOSE_SERVER_ENV_FILE) -f $(COMPOSE_FILE) -f $(COMPOSE_SERVER_FILE)
MICROSERVICE_SERVICES=auth media user post chat support community search
MICROSERVICE_INFRA=db redis minio
MICROSERVICE_EDGE=nginx
MICROSERVICE_MONITORING=prometheus grafana node-exporter
MICROSERVICE_INIT=migrate
MICROSERVICE_ALL=$(MICROSERVICE_SERVICES) $(MICROSERVICE_EDGE) $(MICROSERVICE_MONITORING) $(MICROSERVICE_INFRA)
MICROSERVICE_RUNTIME=$(MICROSERVICE_SERVICES) $(MICROSERVICE_EDGE)
MICROSERVICE_RUNTIME_FULL=$(MICROSERVICE_RUNTIME) $(MICROSERVICE_MONITORING)
NGINX_SITE_NAME ?= arisnet.ru
NGINX_SITES_AVAILABLE ?= /etc/nginx/sites-available
NGINX_SITES_ENABLED ?= /etc/nginx/sites-enabled

DB_HOST ?= 127.0.0.1
DB_PORT ?= 5431
DB_USER ?= kokinside
DB_PASSWORD ?= password1
DB_NAME ?= ARIS-DB
SSL_MODE ?= disable
MIGRATIONS_PATH ?= file://./db/migrations
DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(SSL_MODE)
MIGRATE=migrate -source "$(MIGRATIONS_PATH)" -database "$(DATABASE_URL)"

test:
	go test -v ./...

mocks:
	go generate ./...

clean:
	go clean -testcache
	rm -f ./coverage.out ./coverage.out.tmp
	touch ./coverage.out
	touch ./coverage.out.tmp

coverage: clean
	go test -coverprofile=coverage.out -coverpkg=./internal/... ./...
	go tool cover -html=coverage.out

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

microservices-up: local-prepare
	$(COMPOSE) --profile microservices up -d $(MICROSERVICE_RUNTIME)

microservices-rebuild: local-prepare
	$(COMPOSE) --profile microservices up --build -d $(MICROSERVICE_RUNTIME)

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
	$(COMPOSE) --profile microservices up -d nginx
	$(COMPOSE) exec -T nginx nginx -t
	$(COMPOSE) exec -T nginx nginx -s reload

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
	go test -count=1 ./... -coverprofile=coverage.out.tmp -coverpkg=./internal/...
	cat coverage.out.tmp | grep -v -E "(/mock/|/mocks/|_mock\.go)" > coverage.out
	go tool cover -func=coverage.out | grep total
