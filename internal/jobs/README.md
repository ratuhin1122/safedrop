# River workers

Export DATABASE_URL and run `make migrate` to apply application goose migrations
followed by River's embedded migrations. `make migrate-river` applies only River
migrations and is safe to rerun. API and worker startup do not apply migrations.

Run `make worker` or `go run ./cmd/worker`. The API's insert-only Queue and the
worker share the `process_file` job kind and `{ "file_id": "<uuid>" }` arguments.
ProcessFileArgs.InsertOpts selects the default queue and five maximum attempts.
Existing rows keep their stored maximum attempts. FileArgs is a compatibility
alias for the upload producer.

ProcessFile emits JSON slog records named `process_file started`, including job
ID, file ID, attempt, maximum attempts, and current status. Retries back off by
2, 4, 8, and 16 seconds for new five-attempt jobs; larger attempt counts cap at
one minute. Invalid UUID arguments cancel the job. Missing, soft-deleted, and ready
files are acknowledged without processing. Rejected files may retry a cleanup hook.

Configuration defaults are WORKER_CONCURRENCY=2, WORKER_JOB_TIMEOUT=1m, and
SHUTDOWN_TIMEOUT=10s. Worker startup requires DATABASE_URL. Docker Compose also
sets CPU/memory limits and GOMEMLIMIT. SIGINT/SIGTERM stop fetching and allow
in-flight work to finish; exceeding the shutdown deadline cancels remaining work.

ProcessFile locks the file row before checking status. Concurrent duplicate jobs
serialize on this lock, and each retry reloads the current status. The uploaded
status advances to scanning before the processor runs. Failed attempts commit
their status updates; a final failure marks scanning/processing files rejected.
Processor panics become retryable errors. Terminal statuses are preserved.

Processor receives transaction-scoped sqlc queries and the current file. It must
honor cancellation, make external storage operations idempotent, and set a terminal
status before returning success, or return ErrProcessingPending after a clean
scan to snooze without exhausting attempts. It must use the supplied queries for status
updates to avoid conflicting with the row lock. Its transaction is bounded by the
job timeout. A database error can abort that transaction; River handles the retry.

cmd/worker uses ScanProcessor with clamd and the configured storage backend.
Infected files are rejected and deleted from incoming storage. Clean files advance
to processing; while the image-processing stage is absent they snooze rather than
become ready or exhaust attempts. Later processing can be passed as ScanProcessor.Next.
See [scan setup and tests](../scan/README.md) for configuration and EICAR checks.

On startup and every 15 seconds, the worker reconciles discarded process_file
jobs with exhausted attempts against active file rows. This covers final-attempt
timeouts, process crashes, and database failures that prevented rejection inside
Work. Each reconciliation uses its own transaction and preserves terminal or
deleted files. An uploaded orphan advances through scanning before rejection.

Run `go test ./internal/jobs ./internal/config` for unit tests. Set TEST_DATABASE_URL
to a migrated test PostgreSQL database to also test actual River consumption,
pickup logging, backoff/rejection, duplicate serialization, and reconciliation.
The consuming integration test uses a unique queue so it cannot consume upload
tests' jobs when packages run concurrently.
