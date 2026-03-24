.PHONY: test coverage clean

include .env

MIGRATE=migrate -source "file://./db/migrations" -database "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${SSL_MODE}"

test:
	go test -v ./...

clean:
	if exist coverage.out del /f coverage.out

coverage: clean
	go test -coverprofile=coverage.out -coverpkg=./internal/... ./...
	go tool cover -html=coverage.out

migrage-up:
	$(MIGRATE) up

migrage-down:
	$(MIGRATE) down

# будет подтянут postgres:16
db-up:
	docker compose -f ./docker/docker-compose.yml --env-file ./.env up -d

db-stop:
	docker compose -f ./docker/docker-compose.yml --env-file ./.env stop
