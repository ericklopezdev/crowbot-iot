# Crowbot — dev workflow
DB_URL ?= postgres://crowbot:crowbot@localhost:5433/crowbot?sslmode=disable
MIGRATIONS_DIR := migrations

.PHONY: db-up db-down migrate-up migrate-down migrate-status sqlc tidy

## Start Postgres + Mosquitto
db-up:
	docker compose up -d

## Stop the stack (keeps data volume)
db-down:
	docker compose down

## Apply all migrations
migrate-up:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" up

## Roll back the last migration
migrate-down:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" down

## Show migration status
migrate-status:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" status

## Regenerate type-safe Go from SQL
sqlc:
	sqlc generate

tidy:
	go mod tidy
