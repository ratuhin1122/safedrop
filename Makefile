.PHONY: up down logs ps run worker test lint migrate migrate-river

GO ?= go
GOOSE ?= goose
MIGRATE_COMMAND ?= up
COMPOSE ?= docker compose

up:
	$(COMPOSE) up --build -d --wait --wait-timeout 300

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f api worker web migrate

ps:
	$(COMPOSE) ps -a

# Export DATABASE_URL in your shell; .env.example is a configuration reference.
run:
	$(GO) run ./cmd/api

worker:
	$(GO) run ./cmd/worker

test:
	$(GO) test ./...

lint:
	$(GO) vet ./...

migrate:
	$(GOOSE) -dir migrations postgres "$(DATABASE_URL)" $(MIGRATE_COMMAND)
ifeq ($(MIGRATE_COMMAND),up)
	$(GO) run ./cmd/migrate
endif

migrate-river:
	$(GO) run ./cmd/migrate
