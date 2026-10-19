# Logs, metrics, and traces

Every HTTP request receives a server-generated 128-bit `X-Request-ID` response
header. The ID is stored in context and included by the JSON slog handler,
alongside `trace_id` and `span_id` when a span context exists. Incoming request
IDs are not trusted. Request completion logs include method, redacted path,
client address, status, and duration. Share tokens, credentials, and query
strings are excluded from telemetry attributes.

GET `/metrics` on the API exposes Prometheus text format. The worker exposes
GET `/metrics` on `WORKER_METRICS_ADDR` (default `:9091`). The worker listener is
reachable inside the Compose network; no extra host port is published. Scrape
each worker and API instance separately. Metrics are unauthenticated; use your
deployment network controls to restrict scraping.

| Metric | Meaning |
| --- | --- |
| `safedrop_uploads_total` | API upload responses with HTTP 201 |
| `safedrop_upload_rejections_total` | Upload responses with HTTP 4xx |
| `safedrop_rejected_total` | Worker commits transitioning files to rejected, including reconciliation |
| `safedrop_queue_depth` | PostgreSQL count of available, scheduled, or retryable River jobs across queues |
| `safedrop_queue_depth_up` | 1 when the queue query succeeded, otherwise 0 |
| `safedrop_processing_duration_seconds` | Histogram of ProcessFile attempt duration, including retries and skipped attempts |

Counters and histograms are process-local and reset on restart. A crash between
database commit and counter update can omit a rejection. Queue depth is a live
database gauge; API and worker expose the same backlog, so aggregate it with
`max`, not `sum`, across replicas. A failed or unconfigured database query
exposes queue depth 0 with `queue_depth_up` 0. Scrape queries have a two-second
timeout. Metrics contain no per-user, file, token, or request-ID labels.

OpenTelemetry installs a tracer provider with a service resource, W3C Trace
Context propagation, and parent-based 10% sampling. HTTP server spans propagate
`traceparent` into River job arguments; worker consumer spans continue the trace.
Older jobs without trace context start a new trace. Only route templates,
HTTP methods/statuses, job IDs, and attempt numbers are span attributes.

`OTEL_TRACES_EXPORTER=none` (default) exports no spans. Set it to `slog` for
sampled spans in structured logs through a bounded batch exporter. Shutdown
flushes the exporter with a five-second deadline. This is basic SDK wiring;
an OTLP exporter or collector is not included.

Implementation references: [Prometheus exposition format](https://prometheus.io/docs/instrumenting/exposition_formats/)
and [OpenTelemetry Go instrumentation](https://opentelemetry.io/docs/languages/go/instrumentation/).

Run `go test ./...`; tests cover request/log correlation, trace propagation,
token redaction, histogram buckets, query failure reporting, and HTTP scrapes.
