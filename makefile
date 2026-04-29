.PHONY: test coverage clean dev down reset-db logs migrate backend-shell frontend-shell mocks microservices microservices-up microservices-stop microservices-down microservices-reset auth-up auth-stop media-up media-stop user-up user-stop post-up post-stop chat-up chat-stop logs-auth logs-media logs-user logs-post logs-chat

COMPOSE_FILE=./docker-compose.dev.yml
COMPOSE_ENV_FILE=./.env.compose
COMPOSE=docker compose --env-file $(COMPOSE_ENV_FILE) -f $(COMPOSE_FILE)
COMPOSE_LOCAL=docker compose -f ./docker/docker-compose.yml --env-file ./.env
MICROSERVICE_SERVICES=auth media user post chat
MICROSERVICE_INFRA=db redis minio
MICROSERVICE_INIT=migrate
MICROSERVICE_ALL=$(MICROSERVICE_SERVICES) $(MICROSERVICE_INFRA)

# include .env

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

microservices-up:
	$(COMPOSE) --profile microservices up --build -d $(MICROSERVICE_SERVICES)

microservices-stop:
	$(COMPOSE) stop $(MICROSERVICE_ALL)

microservices-down:
	$(COMPOSE) stop $(MICROSERVICE_ALL)
	$(COMPOSE) rm -f $(MICROSERVICE_ALL) $(MICROSERVICE_INIT)

microservices-reset:
	$(COMPOSE) --profile microservices down -v

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
