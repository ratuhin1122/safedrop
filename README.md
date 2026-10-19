# safedrop

A Go service for streamed uploads, background scanning and processing, and
sharing files through expiring links. Authenticated image upload and transactional
job enqueueing and consumption are implemented; the scan/processing pipeline and
sharing come later.

## Stack

Go 1.25, chi, pgx v5, sqlc, goose, River, govips/libvips, ClamAV via clamd,
aws-sdk-go-v2 with an S3-compatible endpoint (or local storage), Docker Compose,
and slog.

## Layout

- `cmd/api`, `cmd/worker`: API and River consumer entry points.
- `cmd/migrate`: River schema migrations, after application goose migrations.
- `internal/config`, `internal/db`, `internal/storage`: configuration and data.
- `internal/validate`, `internal/scan`, `internal/imageproc`: file checks and processing.
- `internal/jobs`, `internal/share`, `internal/auth`, `internal/ratelimit`: service features.
- `internal/httpx`, `internal/observability`: HTTP helpers and diagnostics.
- `migrations/`, `queries/`: goose migrations and sqlc query sources.

## Development

Install Go 1.25 or newer and Make. Copy `.env.example` to `.env` and export the
needed values in your shell; the scaffold does not load `.env` automatically.

- `go build ./...`: build all implemented packages.
- `make test`: run Go tests.
- `make lint`: run standard Go vet checks.
- `make run`: start the API.
- `make worker`: start the River consumer after migrations.
- `make migrate`: apply goose and River migrations using `DATABASE_URL`.
  Use `make migrate MIGRATE_COMMAND=status` to inspect migration status.

See [upload setup and protocol](internal/httpx/UPLOADS.md) for the development
auth stub, raw-body POST /v1/files, and PostgreSQL integration tests.
See [worker setup](internal/jobs/README.md) for retries and the processor interface.

## File lifecycle

`uploaded -> scanning -> processing -> ready | rejected`

Uploads stream into private `incoming/` storage. Only passing files reach
private `final/` storage. Files must never be buffered whole in memory.
Keep each source file under 150 lines and add tests with each implementation.
