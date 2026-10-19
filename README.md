# safedrop

A Go service for streamed uploads, background scanning and processing, and
sharing files through expiring links. Authenticated image upload and transactional
job enqueueing, consumption, clamd scanning, image variants, and expiring sharing
are wired. Native image processing requires libvips.

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

See [accounts and ownership](internal/auth/README.md) for registration, login,
JWT configuration, and owner-scoped file endpoints.
See [request limits](internal/ratelimit/README.md) for upload windows, daily
quotas, concurrency caps, and optional Turnstile registration checks.
See [observability](internal/observability/README.md) for request IDs, metrics,
worker scraping, and tracing configuration.
See [upload setup and protocol](internal/httpx/UPLOADS.md) for raw-body
POST /v1/files and PostgreSQL integration tests.
See [worker setup](internal/jobs/README.md) for retries and the processor interface.
See [share links](internal/share/README.md) for creation, passwords, download limits,
and periodic expiry cleanup.

## Compose observability

Copy `.env.example` to `.env` and set JWT_SECRET to a random secret of at least
32 bytes. Apply application and River migrations before starting the worker
(see `make migrate` above; use COMPOSE_DATABASE_URL inside the Compose network).
Start the optional monitoring services with:

```sh
docker compose --profile observability up
```

Use `--build` on the first run or after code changes. The default profile starts
neither monitoring service and publishes no API or worker metrics ports.
Compose fixes both internal metrics listeners at :9090; METRICS_ADDR applies to
host-run binaries. Prometheus scrapes api:9090 and worker:9090 every 15 seconds.
Open [Prometheus Targets](http://localhost:9090/targets) and confirm both jobs
show UP after startup and a scrape interval. Query `up{job=~"api|worker"}`;
both values should be 1. Open [Grafana](http://localhost:3000), sign in with
admin/admin on a fresh volume, and change the password when prompted.
The default Prometheus datasource is provisioned automatically. Both monitoring
UIs bind to localhost, and their data persists in named volumes.

Run `pwsh -File deploy/check-observability.ps1` to check Compose isolation and
mounts; add `-Live` after startup to verify both Prometheus targets are UP.
Compose and configuration checks passed for this step. Docker's daemon did not
respond, so live target status could not be verified here.

Configuration references: [Prometheus](https://prometheus.io/docs/prometheus/latest/configuration/configuration/)
and [Grafana provisioning](https://grafana.com/docs/grafana/latest/administration/provisioning/).

## File lifecycle

`uploaded -> scanning -> processing -> ready | rejected`

Uploads stream into private `incoming/` storage. Only passing files reach
private `final/` storage. Files must never be buffered whole in memory.
Keep each source file under 150 lines and add tests with each implementation.
