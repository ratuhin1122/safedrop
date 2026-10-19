# safedrop

A Go service for streamed uploads, background scanning and processing, and
sharing files through expiring links. This repository currently contains only
the project scaffold; application code, tests, and migrations come later.

## Stack

Go 1.25, chi, pgx v5, sqlc, goose, River, govips/libvips, ClamAV via clamd,
aws-sdk-go-v2 with an S3-compatible endpoint (or local storage), Docker Compose,
and slog. Dependencies will be added as their features are implemented.

## Layout

- `cmd/api`, `cmd/worker`: future service entry points.
- `internal/config`, `internal/db`, `internal/storage`: configuration and data.
- `internal/validate`, `internal/scan`, `internal/imageproc`: file checks and processing.
- `internal/jobs`, `internal/share`, `internal/auth`, `internal/ratelimit`: service features.
- `internal/httpx`, `internal/observability`: HTTP helpers and diagnostics.
- `migrations/`, `queries/`: goose migrations and sqlc query sources.

## Development

Install Go 1.25 or newer and Make. Copy `.env.example` to `.env` and export the
needed values in your shell; the scaffold does not load `.env` automatically.

- `go build ./...`: check the scaffold. With no Go files, Go reports no packages.
- `make test`: run Go tests once packages exist.
- `make lint`: run the standard Go vet checks once packages exist.
- `make run`: start the API once `cmd/api` is implemented.
- `make migrate`: apply migrations using an installed goose CLI and `DATABASE_URL`.
  Use `make migrate MIGRATE_COMMAND=status` to inspect migration status.

## File lifecycle

`uploaded -> scanning -> processing -> ready | rejected`

Uploads stream into private `incoming/` storage. Only passing files reach
private `final/` storage. Files must never be buffered whole in memory.
Keep each source file under 150 lines and add tests with each implementation.
