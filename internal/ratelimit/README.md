# Request limits

The API defaults to 20 admitted upload attempts per sliding minute, both per
authenticated user and per client IP. The 21st attempt returns 429 with a
rounded-up `Retry-After` in seconds. Requests admitted to the window count
even if validation or storage subsequently fails. Denied requests do not count.
Each user may hold at most three upload slots through validation, storage,
and transaction completion. Slots are released on every return path.

`UploadLimiter.Begin` is the replacement boundary for a Redis implementation.
The current limiter and concurrency cap apply within one API process, reset on
restart, and are not shared across replicas. History expires, idle token buckets
are removed, and both maps are capped at 100,000 keys; capacity exhaustion
fails closed. Configure `UPLOAD_RATE_WINDOW`, `UPLOAD_RATE_PER_USER`, and
`UPLOAD_RATE_PER_IP`. The concurrency cap is fixed at three.

Other endpoints share a per-IP `golang.org/x/time/rate` token bucket configured
by `RATE_LIMIT_REQUESTS_PER_SECOND` and `RATE_LIMIT_BURST`. Health checks are
exempt. Install `WithRateLimits` before route options. IPs come from RemoteAddr
after the trusted-proxy middleware; arbitrary forwarding headers are ignored.

`DAILY_UPLOAD_BYTES` defaults to 1 GiB, resetting at midnight UTC. A preflight
check reads `upload_usage` before the body and again after validation. Unknown
lengths are still bounded by the upload size limit. The transaction atomically
charges actual bytes and one upload before creating the file and River job.
Quota exhaustion rolls everything back and returns 429 until midnight UTC.
Failed uploads do not consume daily bytes. Storage is removed on a definite
transaction failure. In-flight uploads can finish reading before a concurrent
upload exhausts the quota, but cannot commit excess bytes. This quota is shared
across API replicas through PostgreSQL.

Set `TURNSTILE_SECRET` to require `turnstile_token` in registration JSON.
Optionally set `TURNSTILE_HOSTNAME` to match the widget's hostname. Validation
calls Cloudflare over HTTPS with a five-second timeout before account creation.
Failure returns 403; verification service errors return 503 with `Retry-After`.
Login does not require a challenge. Configure the browser widget separately.
See [Cloudflare server validation](https://developers.cloudflare.com/turnstile/get-started/server-side-validation/).
Tests use a local verification server and never send tokens to Cloudflare.

Run `go test ./...`. With `TEST_DATABASE_URL` pointing to a migrated disposable
database, run `go test -p 1 ./... -count=1`. Tests cover the 21st HTTP upload,
sliding boundaries, independent users/IPs, concurrent slots, bucket refill,
daily quota races, transaction rollback, and Turnstile verification failures.
