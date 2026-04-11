.PHONY: test coverage clean dev down reset-db logs migrate backend-shell frontend-shell

COMPOSE_FILE=./docker-compose.dev.yml
COMPOSE_ENV_FILE=./.env.compose
COMPOSE=docker compose --env-file $(COMPOSE_ENV_FILE) -f $(COMPOSE_FILE)

MIGRATE=migrate -source "file://./db/migrations" -database "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${SSL_MODE}"

test:
	go test -v ./...

clean:
	if exist coverage.out del /f coverage.out

coverage: clean
	go test -coverprofile=coverage.out -coverpkg=./internal/... ./...
	go tool cover -html=coverage.out

migrate-up: migrate
	$(MIGRATE) up 5

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
	docker compose -f ./docker/docker-compose.yml --env-file ./.env up -d ARISNET-DB

db-stop:
	docker compose -f ./docker/docker-compose.yml --env-file ./.env stop ARISNET-DB

s3-up:
	docker compose -f ./docker/docker-compose.yml --env-file ./.env up -d ARISNET-MINIO

s3-stop:
	docker compose -f ./docker/docker-compose.yml --env-file ./.env stop ARISNET-MINIO

services-up:
	docker compose -f ./docker/docker-compose.yml --env-file ./.env up -d

services-stop:
	docker compose -f ./docker/docker-compose.yml --env-file ./.env stop

dev:
	$(COMPOSE) up --build -d
	sh ./scripts/dev-ready.sh $(COMPOSE_ENV_FILE)

down:
	$(COMPOSE) down

reset-db:
	$(COMPOSE) down -v
	$(COMPOSE) up --build -d
	sh ./scripts/dev-ready.sh $(COMPOSE_ENV_FILE)

logs:
	$(COMPOSE) logs -f

logs-backend:
	$(COMPOSE) logs -f backend

backend-shell:
	$(COMPOSE) exec backend sh

frontend-shell:
	$(COMPOSE) exec frontend sh
