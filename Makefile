.PHONY: run test lint migrate migrate-river

GO ?= go
GOOSE ?= goose
MIGRATE_COMMAND ?= up

# Export DATABASE_URL in your shell; .env.example is a configuration reference.
run:
	$(GO) run ./cmd/api

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
