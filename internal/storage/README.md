# Storage

`Store` provides streaming `Put`, `Get`, `Delete`, `Move`, and `PresignGet`.
Keys must be relative paths under `incoming/` or `final/`. Generate unique keys
for each upload. Pipeline code must scan and process an incoming object before
moving it to final storage. Both backends remain private; only final objects
can receive presigned download URLs.

`NewLocal(LocalConfig{Root: "./data"})` opens a volume confined by `os.Root`.
Close the store on shutdown and close every reader returned by `Get`.
Uploads are copied to private temporary files and renamed after success.
Failed writes preserve any previous object. Local storage ignores the provided
content type; download responses use attachment disposition and octet-stream.

To enable local presigning, set `BaseURL` to an HTTP(S) download endpoint and
provide a secret `SigningKey` of at least 32 bytes. Mount the Local instance as
an HTTP handler at that URL's exact path. Without these options, `PresignGet`
returns `ErrPresignDisabled`. Signed URLs support GET and HEAD and expire
after the requested duration. Both backends accept TTLs of 1 second to 7 days.

`NewS3(S3Config{...})` accepts an endpoint, region, bucket, static credentials,
an optional session token, and a path-style addressing flag. The bucket must
already exist and remain private. The endpoint must also be reachable by the
browser that uses a presigned URL; configure a public storage hostname if needed.
Large uploads use AWS SDK v2 multipart uploads with 5 MiB parts, one upload
worker, bounded part buffers, and automatic abort on failure. Object size is
subject to S3 multipart limits. Configure bucket lifecycle cleanup for abandoned
multipart uploads as protection against process termination.

S3 moves stream Get into Put, then delete the source; this also supports objects
larger than a single CopyObject request. Moves preserve content type. A failed
source deletion can leave both objects. Local moves use filesystem rename.
Delete is idempotent for missing objects. Missing Get/Move sources match
`ErrNotFound` through `errors.Is`. Operations honor context cancellation between
reads; the caller must unblock its input reader if that reader itself blocks.

Run `go test ./internal/storage`. Tests use temporary directories and a mock
S3 HTTP endpoint; no storage account or running S3 server is required.
