# Expiring shares

POST /v1/files/{id}/shares requires the configured Authenticator and an owned,
ready file. Send JSON with an RFC3339 expires_at in the future, optional password
(at most 72 UTF-8 bytes), and optional positive max_downloads. The 201 response
contains a relative URL such as /s/{token} and expires_at. Tokens contain 128
cryptographically random bits; only their SHA-256 hashes are stored. Optional
passwords use bcrypt cost 12. Token URLs are redacted from request logs.

GET /s/{token} returns 410 for expired, exhausted, invalid, missing, or deleted
links/files. Protected links require HTTP Basic authentication; the username is
ignored and the password is checked before reserving a download. Failed password
checks return 401 and do not consume a download.

Download reservations use a conditional SQL increment, so concurrent requests
cannot exceed max_downloads. Each authorized GET consumes one reservation even
if the client later disconnects. HEAD is not supported. Responses are private,
no-store attachments; the body streams directly from private storage.

SHARE_PRESIGN_GET=true permits a 307 redirect for unlimited links, with a URL
valid for at most 60 seconds and never beyond link expiry. Links with a download
cap always stream through the service because presigned URLs can be replayed.
Local storage without signing configuration falls back to direct streaming.
The configured S3 endpoint must be reachable from the downloading browser.

The River expire_shares job runs at startup and once per minute on the worker's
leader. It removes expired links in batches. A file is retained while any other
unexpired link references it; after the last link expires the file is marked
deleted and a durable deletion record is committed. Storage deletion then removes
the primary object, every recorded variant, and any original awaiting cleanup.
Only after successful deletion is the files row removed. Storage failures keep
the deletion record for retries and later periodic runs. File locking serializes
link creation against expiry cleanup. These operations are safe to repeat.

Run make migrate before starting the API and worker. The new share_file_deletions
table is an outbox for external storage deletion; application and River migrations
remain separate. Run go test ./internal/share for HTTP, password, token-hashing,
presigning, and concurrency tests. Set TEST_DATABASE_URL to a migrated test
database to also verify real SQL limits, active-link preservation, variant removal,
and cleanup recovery after storage failure.
For the full database-backed suite, use go test -p 1 ./... so maintenance tests
using separate temporary storage roots do not operate on another package's fixtures.

Public landing pages can request GET /s/{token}?info=1 for JSON containing
name, mime, size, and expires_at. This applies the same password, expiry,
file-ready, and download-limit checks without reading storage or reserving a
download. Missing links return 404 in metadata mode; expired or exhausted links
return 410, and password challenges return 401. The attachment endpoint keeps
its existing 410 behavior. Both modes send X-Robots-Tag: noindex, nofollow.
Serve browser HTML navigation to /s/{token} from the frontend; forward JSON
metadata requests and attachment requests to the API. The Vite dev proxy
distinguishes these by Accept. No public thumbnail URL is exposed.
