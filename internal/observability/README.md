# Logs, metrics, and traces

LOG_LEVEL selects the JSON slog level (INFO by default). Each request preserves
one valid X-Request-ID (1–64 ASCII letters, digits, hyphens or underscores), or
generates a random 128-bit ID. It is stored in context and the response header.
Exactly one completion record includes request_id, method, chi route template
(unmatched if absent), status, duration_ms, and user_id when authentication has
identified the caller. LoggerFrom(ctx) also binds trace_id/span_id when present,
including calls without an explicit Context argument. Raw URLs, query strings,
Authorization headers, passwords, share tokens, and file contents are not logged.
Panic values and scanner signatures are omitted. Rejections produce an additional
WARN domain event with file_id and a fixed reason (virus or rejected_type, etc.).
Pre-persistence rejections use a generated attempt UUID; there is no database row.

Both binaries serve GET `/metrics` on a separate listener configured by
`METRICS_ADDR` (default `:9090`), using `promhttp.HandlerFor` with the custom
registry. The public API listener does not serve `/metrics`. Go runtime and
process collectors are registered; process metrics depend on OS support.
Both metrics listeners drain requests during shutdown, with a five-second limit.
Each binary constructs one Metrics instance and injects it into its server,
handlers, limiter middleware, and worker constructors. No global registry is used.
Stage observers use the registry carried by their ProcessFile attempt context.
When running both binaries on one host, set a different METRICS_ADDR for one.
Compose keeps API and worker metrics private on port 9090. The optional
observability profile scrapes both every 15 seconds and publishes Prometheus at
localhost:9090 and Grafana at localhost:3000. See the root README for startup.
Scrape each instance separately. Metrics are unauthenticated; use deployment
network controls to restrict scraping.

With the API running, `curl localhost:8080/healthz` followed by
`curl localhost:9090/metrics` shows the health request counter.
`curl -i localhost:8080/metrics` returns 404.

| Metric | Meaning |
| --- | --- |
| `safedrop_uploads_total{result}` | accepted, rejected_size, rejected_type, rejected_pixels, rate_limited, quota_exceeded |
| `safedrop_upload_bytes_total` | Actual bytes of committed uploads |
| `safedrop_files_processed_total{result}` | ready, rejected_virus, rejected_error, retried |
| `safedrop_processing_duration_seconds{stage}` | scan, convert, total |
| `safedrop_queue_jobs{state}` | available, running, retryable, discarded |
| `safedrop_ratelimit_hits_total{limiter}` | upload_window, api_bucket, concurrency |
| `safedrop_share_requests_total{result}` | ok, expired, limit_reached, wrong_password, not_found |
| `http_requests_total{method,route,status}` | HTTP outcomes by chi route template |
| `http_request_duration_seconds{method,route}` | HTTP duration by chi route template |

Counters and histograms are process-local and reset on restart. A crash between
database commit and counter update can omit an outcome. Terminal replays do not
increment processing outcomes again. `retried` counts nonterminal failed attempts
with attempts remaining; planned snoozes and cancellations are excluded. Terminal
outcomes are counted after commit. Reconciliation counts repaired terminal failures
as rejected_error. Scan/convert timings measure those stages; total includes the
whole worker attempt, even replays. Share `ok` counts authorization, including
presigned redirects, before streaming completes. Missing/deleted links or objects
count as not_found; an already-removed expired link cannot be distinguished.

Accepted means committed before scanning: a good image plus an EICAR-bearing
image yields accepted=2, ready=1, rejected_virus=1. Raw EICAR text fails image
type validation and never reaches the worker. API uploads and worker outcomes
are scraped from their respective processes and combined in Prometheus.

Queue gauges take one grouped PostgreSQL snapshot at startup and every 15
seconds using the existing pool. Missing states reset to zero after successful
polls. Queries have a five-second timeout; failures are logged and the previous
snapshot is retained until a later poll succeeds. Scrapes never query the database
and remain available during database failures. Pollers stop on context cancellation
and are joined before the pool closes. API and worker expose the same counts, so
aggregate with `max`, not `sum`, across replicas. With no database, all queue states
are zero. HTTP route labels use chi patterns, with `unmatched`
when chi has no final pattern, including requests rejected before routing.
Unknown HTTP methods use OTHER. Metrics never label user/file IDs, tokens,
filenames, raw paths, or request IDs. Unauthorized/malformed upload requests and
infrastructure failures appear in HTTP metrics; they do not invent upload results
outside the listed vocabulary. Quota hits do not invent a rate limiter label.

OpenTelemetry uses a no-op provider when `OTEL_EXPORTER_OTLP_ENDPOINT` is empty.
Set it to an OTLP HTTP base URL; `/v1/traces` is appended. `OTEL_SERVICE_NAME`
overrides the binary's service name (safedrop-api or safedrop-worker). Enabled
tracing samples new roots and respects the parent's sampling decision. Shutdown
flushes queued spans with a five-second deadline. Export failures log a fixed
message without exposing endpoint credentials or telemetry payloads.

The otelhttp server span uses the chi route template. Upload spans are
`validate.upload`, `storage.put`, and `jobs.enqueue`. River arguments persist the
W3C traceparent of jobs.enqueue; the worker restores it as the remote parent of
`process_file`. Worker stages are `scan.clamav`, `imageproc.convert`, and
`storage.move`. Converted variants are staged in private incoming/ storage before
moving to final/. Older jobs without traceparent start a new trace. LoggerFrom
binds trace_id and span_id from the active span. Exported attributes are limited
to route template, method, status, job ID, and attempt; raw paths are removed.

For the optional Compose Jaeger profile, set `OTEL_EXPORTER_OTLP_ENDPOINT` to
`http://jaeger:4318`, then run `docker compose --profile tracing up --build`.
Open [Jaeger](http://localhost:16686), upload an image, and search safedrop-api.
Its trace includes safedrop-worker spans after processing. When running Go
binaries on the host, use `http://localhost:4318` instead. Compose service names
can be overridden with API_OTEL_SERVICE_NAME and WORKER_OTEL_SERVICE_NAME.

Validation passed: go build ./..., go vet ./..., the full test suite with an
isolated PostgreSQL database, and docker compose config. Tests cover no-op
startup, an HTTP OTLP receiver, shutdown flushing, private
attribute removal, and persisted River trace continuity through a database-backed
upload pipeline. The pipeline uses a test scanner and encoder. Docker's daemon
was unavailable during this step, so the Jaeger container check was not run.

Implementation references: [Prometheus custom registries](https://prometheus.io/docs/guides/go-application/)
and [OpenTelemetry Go instrumentation](https://opentelemetry.io/docs/languages/go/instrumentation/).

Run `go test ./...`; tests cover request/log correlation, trace propagation,
token redaction, histogram buckets, query failure reporting, and HTTP scrapes.
