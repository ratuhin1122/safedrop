# Logs, metrics, and traces

Every HTTP request receives a server-generated 128-bit `X-Request-ID` response
header. The ID is stored in context and included by the JSON slog handler,
alongside `trace_id` and `span_id` when a span context exists. Incoming request
IDs are not trusted. Request completion logs include method, redacted path,
client address, status, and duration. Share tokens, credentials, and query
strings are excluded from telemetry attributes.

GET `/metrics` on the API uses `prometheus/client_golang` and custom registries.
Go runtime and process collectors are registered; process metrics depend on OS
support. The worker exposes
GET `/metrics` on `WORKER_METRICS_ADDR` (default `:9091`). The worker listener is
reachable inside the Compose network; no extra host port is published. Scrape
each worker and API instance separately. Metrics are unauthenticated; use your
deployment network controls to restrict scraping.

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

Queue gauges query PostgreSQL on each scrape. API and worker expose the same
counts, so aggregate with `max`, not `sum`, across replicas. Queries have a
two-second timeout; a query error fails the scrape with 500. With no database,
all queue states are zero. HTTP route labels use chi patterns, with `unmatched`
for unknown routes and 405 responses, including requests rejected before routing.
Unknown HTTP methods use OTHER. Metrics never label user/file IDs, tokens,
filenames, raw paths, or request IDs. Unauthorized/malformed upload requests and
infrastructure failures appear in HTTP metrics; they do not invent upload results
outside the listed vocabulary. Quota hits do not invent a rate limiter label.

OpenTelemetry installs a tracer provider with a service resource, W3C Trace
Context propagation, and parent-based 10% sampling. HTTP server spans propagate
`traceparent` into River job arguments; worker consumer spans continue the trace.
Older jobs without trace context start a new trace. Only route templates,
HTTP methods/statuses, job IDs, and attempt numbers are span attributes.

`OTEL_TRACES_EXPORTER=none` (default) exports no spans. Set it to `slog` for
sampled spans in structured logs through a bounded batch exporter. Shutdown
flushes the exporter with a five-second deadline. This is basic SDK wiring;
an OTLP exporter or collector is not included.

Implementation references: [Prometheus custom registries](https://prometheus.io/docs/guides/go-application/)
and [OpenTelemetry Go instrumentation](https://opentelemetry.io/docs/languages/go/instrumentation/).

Run `go test ./...`; tests cover request/log correlation, trace propagation,
token redaction, histogram buckets, query failure reporting, and HTTP scrapes.
