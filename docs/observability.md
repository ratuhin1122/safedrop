# Observability queries

Paste these expressions into Prometheus or Grafana's Prometheus query editor.
They assume one deployment, with API and worker scraped every 15 seconds.
Counters are summed across instances after increase/rate handles resets.
Queue gauges repeat the same database snapshot on each instance, so use max.
Upload acceptance precedes scanning; virus rejection is a processing outcome.

## Uploads accepted versus rejected in the last hour

```promql
sum(increase(safedrop_uploads_total{result="accepted"}[1h]))
```

Counts accepted uploads during the last hour, including files later rejected by scanning.

```promql
sum(increase(safedrop_uploads_total{result!="accepted"}[1h]))
```

Counts recorded upload denials during the last hour, including rate and quota denials.

## Rejection reasons breakdown

```promql
sum by (result) (increase(safedrop_uploads_total{result!="accepted"}[1h]))
```

Splits hourly upload denials into rejected_size, rejected_type, rejected_pixels, rate_limited, and quota_exceeded.

## Files processed per minute

```promql
60 * sum by (result) (rate(safedrop_files_processed_total{result=~"ready|rejected_virus|rejected_error"}[5m]))
```

Shows terminal processing outcomes per minute, averaged over five minutes, excluding retried attempts.

## Virus detections in the last 24 hours

```promql
sum(increase(safedrop_files_processed_total{result="rejected_virus"}[24h]))
```

Counts committed virus rejections during the last day; raw EICAR rejected by upload validation is excluded.

## Queue size by state

```promql
max by (state) (safedrop_queue_jobs{state=~"available|running|retryable|discarded"})
```

Shows the shared River queue size per state without double-counting API and worker snapshots.

```promql
max(safedrop_queue_jobs{state="available"}) > 100
```

Selects a backlog above 100 available jobs; use an alert rule with `for: 5m` for persistence, as below.

## p95 processing time per stage

```promql
histogram_quantile(0.95, sum by (stage, le) (rate(safedrop_processing_duration_seconds_bucket[5m])))
```

Estimates the five-minute p95 duration in seconds for scan, convert, and total, including retry attempts.

`_bucket` is the series exported by the defined processing histogram; `le` is its bucket boundary label.
Stages with no observations produce NaN; increase/rate need at least two samples.

## Rate-limit hits per limiter

```promql
sum by (limiter) (increase(safedrop_ratelimit_hits_total[1h]))
```

Counts hourly denials by upload_window, api_bucket, and concurrency limiter.

## HTTP 5xx ratio

```promql
(
  (sum(rate(http_requests_total{status=~"5.."}[5m])) or vector(0))
  / sum(rate(http_requests_total[5m]))
)
and (sum(rate(http_requests_total[5m])) > 0)
```

Shows the five-minute fraction of requests returning 5xx (0–1), with zero for no errors and no result for no traffic.

## Suggested alerts

These are example rule definitions, not installed alerting rules; tune thresholds to expected traffic and job duration.
`for` belongs to the alert rule, not PromQL, and requires its expression to remain true across evaluations.

```yaml
groups:
  - name: safedrop
    rules:
      - alert: SafedropQueueBacklog
        expr: max(safedrop_queue_jobs{state="available"}) > 100
        for: 5m
      - alert: SafedropHTTP5xxRatio
        expr: |
          (
            (sum(rate(http_requests_total{status=~"5.."}[5m])) or vector(0))
            / sum(rate(http_requests_total[5m]))
          ) > 0.05
        for: 5m
      - alert: SafedropProcessingStalled
        expr: |
          (max(safedrop_queue_jobs{state="available"}) > 0)
          and
          (sum(increase(safedrop_files_processed_total{result=~"ready|rejected_virus|rejected_error"}[10m])) == 0)
        for: 5m
```

- Queue backlog: more than 100 available jobs continuously for five minutes.
- HTTP failures: more than 5% of requests return 5xx continuously for five minutes; no traffic does not fire.
- Processing stalled: available work persists for five minutes while the rolling ten-minute window has no terminal completions.

Missing scrapes can leave queue snapshots stale or completion series absent; these alerts do not detect scrape failure.
Scope all selectors consistently if multiple deployments share Prometheus; max assumes a shared database.

References: [PromQL functions](https://prometheus.io/docs/prometheus/latest/querying/functions/)
and [alerting rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/).
