THRESHOLD = 10
SRC = $(shell find . -type f -name "*.go" -not -path "./vendor/*" -not -path "./testdata/*")
BINARY = myapp

include .env
export $(shell grep -Eo '^[A-Z_]+=' .env | sed 's/=//')

MIGRATION_FOLDER = $(CURDIR)/internal/migrations
DB_SETUP = 'user=$(DB_USER) password=$(DB_PASSWORD) dbname=$(DB_NAME) host=$(DB_HOST) port=$(DB_PORT) sslmode=$(DB_SSLMODE)'

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

db-logs:
	docker logs -f orders-pg

goose-install:
	@go install github.com/pressly/goose/v3/cmd/goose@latest
	@goose --version

migration-create:
	@mkdir -p "$(MIGRATION_FOLDER)"
	goose -dir "$(MIGRATION_FOLDER)" create "$(name)" sql

migration-up:
	goose -dir "$(MIGRATION_FOLDER)" postgres $(DB_SETUP) up

migration-down:
	goose -dir "$(MIGRATION_FOLDER)" postgres $(DB_SETUP) down

migration-reset:
	goose -dir "$(MIGRATION_FOLDER)" postgres $(DB_SETUP) reset

migration-status:
	goose -dir "$(MIGRATION_FOLDER)" postgres $(DB_SETUP) status

db-init: db-up goose-install migration-up migration-status
db-reseed: migration-reset migration-up migration-status

test-all:
	@go test ./... -v -cover
