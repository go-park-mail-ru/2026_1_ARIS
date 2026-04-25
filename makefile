.PHONY: test coverage clean dev down reset-db logs migrate backend-shell frontend-shell mocks

COMPOSE_FILE=./docker-compose.dev.yml
COMPOSE_ENV_FILE=./.env.compose
COMPOSE=docker-compose --env-file $(COMPOSE_ENV_FILE) -f $(COMPOSE_FILE)
COMPOSE_LOCAL=docker-compose -f ./docker/docker-compose.yml --env-file ./.env

-include .env

MIGRATE=migrate -source "file://./db/migrations" -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(SSL_MODE)"

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

migrate-up: migrate
	$(MIGRATE) up

migrate-down: migrate
	$(MIGRATE) down

migrate-version: migrate
	${MIGRATE} version

migrate-force-down: migrate
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

services-down:
	$(COMPOSE_LOCAL) down -v

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
	cat coverage.out.tmp | grep -v -E "(/mocks/|generated|pb\.go|_test\.go|internal/utils/mock_data\.go)" > coverage.out
	go tool cover -func=coverage.out | grep total
