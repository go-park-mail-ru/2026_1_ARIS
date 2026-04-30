.PHONY: test coverage clean dev down reset-db logs migrate backend-shell frontend-shell mocks microservices microservices-up microservices-stop microservices-down microservices-reset server-up server-stop server-down server-reset server-logs server-nginx-up server-nginx-stop server-nginx-test server-nginx-reload server-nginx-update server-nginx-install server-host-nginx-test server-host-nginx-reload auth-up auth-stop media-up media-stop user-up user-stop post-up post-stop chat-up chat-stop support-up support-stop nginx-up nginx-stop nginx-test nginx-reload nginx-update logs-auth logs-media logs-user logs-post logs-chat logs-support logs-nginx

COMPOSE_FILE=./docker-compose.dev.yml
COMPOSE_ENV_FILE=./.env.compose
COMPOSE=docker compose --env-file $(COMPOSE_ENV_FILE) -f $(COMPOSE_FILE)
COMPOSE_SERVER_FILE=./docker-compose.server.yml
COMPOSE_SERVER_ENV_FILE=./.env.server
COMPOSE_SERVER=docker compose --env-file $(COMPOSE_SERVER_ENV_FILE) -f $(COMPOSE_FILE) -f $(COMPOSE_SERVER_FILE)
COMPOSE_LOCAL=docker compose -f ./docker/docker-compose.yml --env-file ./.env
MICROSERVICE_SERVICES=auth media user post chat support
MICROSERVICE_INFRA=db redis minio
MICROSERVICE_EDGE=nginx
MICROSERVICE_INIT=migrate
MICROSERVICE_ALL=$(MICROSERVICE_SERVICES) $(MICROSERVICE_EDGE) $(MICROSERVICE_INFRA)
MICROSERVICE_RUNTIME=$(MICROSERVICE_SERVICES) $(MICROSERVICE_EDGE)
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

# обновить конфигурацию сваггера
swagger:
	swag init -g main.go --dir ./cmd/server,./internal/handler/auth,./internal/handler/feed,./internal/handler/profile,./internal/handler/friend,./internal/handler/user,./internal/handler/proxy,./internal/handler/dto,./internal/service/dto,./internal/models --output docs

# будет подтянут postgres:16
db-up:
	$(COMPOSE_LOCAL) up -d ARISNET-DB

db-stop:
	$(COMPOSE_LOCAL) stop ARISNET-DB

s3-up:
	$(COMPOSE_LOCAL) up -d ARISNET-MINIO

s3-stop:
	$(COMPOSE_LOCAL) stop ARISNET-MINIO

services-up:
	$(COMPOSE_LOCAL) up -d

services-stop:
	$(COMPOSE_LOCAL) stop

microservices: microservices-up

local-up: microservices-up
local-stop: microservices-stop
local-down: microservices-down
local-reset: microservices-reset

microservices-up:
	$(COMPOSE) --profile microservices up --build -d $(MICROSERVICE_SERVICES)
	$(COMPOSE) --profile microservices up --build -d $(MICROSERVICE_EDGE)

microservices-stop:
	$(COMPOSE) stop $(MICROSERVICE_ALL)

microservices-down:
	$(COMPOSE) stop $(MICROSERVICE_ALL)
	$(COMPOSE) rm -f $(MICROSERVICE_ALL) $(MICROSERVICE_INIT)

microservices-reset:
	$(COMPOSE) --profile microservices down -v

server-up:
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

server-nginx-up:
	$(COMPOSE_SERVER) --profile microservices up -d nginx

server-nginx-stop:
	$(COMPOSE_SERVER) stop nginx

server-nginx-test:
	$(COMPOSE_SERVER) exec -T nginx nginx -t

server-nginx-reload:
	$(COMPOSE_SERVER) exec -T nginx nginx -t
	$(COMPOSE_SERVER) exec -T nginx nginx -s reload

server-nginx-update:
	$(COMPOSE_SERVER) --profile microservices up -d nginx
	$(COMPOSE_SERVER) exec -T nginx nginx -t
	$(COMPOSE_SERVER) exec -T nginx nginx -s reload

server-nginx-install:
	sudo mkdir -p $(NGINX_SITES_AVAILABLE) $(NGINX_SITES_ENABLED)
	sudo install -m 0644 ./config/nginx.server.conf $(NGINX_SITES_AVAILABLE)/$(NGINX_SITE_NAME)
	sudo ln -sfn $(NGINX_SITES_AVAILABLE)/$(NGINX_SITE_NAME) $(NGINX_SITES_ENABLED)/$(NGINX_SITE_NAME)
	sudo nginx -t

server-host-nginx-test:
	sudo nginx -t

server-host-nginx-reload:
	sudo nginx -t
	sudo systemctl reload nginx

auth-up:
	$(COMPOSE) up --build -d auth

auth-stop:
	$(COMPOSE) stop auth

media-up:
	$(COMPOSE) up --build -d media

media-stop:
	$(COMPOSE) stop media

user-up:
	$(COMPOSE) up --build -d user

user-stop:
	$(COMPOSE) stop user

post-up:
	$(COMPOSE) up --build -d post

post-stop:
	$(COMPOSE) stop post

chat-up:
	$(COMPOSE) up --build -d chat

chat-stop:
	$(COMPOSE) stop chat

support-up:
	$(COMPOSE) up --build -d support

support-stop:
	$(COMPOSE) stop support

nginx-up:
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

logs-nginx:
	$(COMPOSE) logs -f nginx

dev:
	$(COMPOSE) up --build -d
	sh ./scripts/dev-ready.sh $(COMPOSE_ENV_FILE)

down:
	$(COMPOSE) down

stop:
	$(COMPOSE) stop

reset-db:
	$(COMPOSE) down -v
	$(COMPOSE) up --build -d
	sh ./scripts/dev-ready.sh $(COMPOSE_ENV_FILE)

logs:
	$(COMPOSE) logs -f

logs-backend:
	$(COMPOSE) logs -f backend

logs-redis:
	$(COMPOSE) logs -f redis

logs-db:
	$(COMPOSE) logs -f db

logs-migrate:
	$(COMPOSE) logs -f migrate

backend-shell:
	$(COMPOSE) exec backend sh

frontend-shell:
	$(COMPOSE) exec frontend sh
	
coverage-excluding-mocks: clean
	go test ./... -coverprofile=coverage.out.tmp -coverpkg=./internal/...
	cat coverage.out.tmp | grep -v -E "(mock|generated|pb\.go|mocks|_test\.go)" > coverage.out
	go tool cover -func=coverage.out | grep total
