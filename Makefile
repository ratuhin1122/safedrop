.PHONY: run test lint migrate

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
