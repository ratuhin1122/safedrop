# POST /v1/files

Send the raw file bytes with `Authorization: Bearer <token>` and one
`X-Filename` header containing a display filename (at most 255 bytes, no paths
or control characters). The filename extension and client Content-Type do not
determine the detected type. Multipart form encoding is not supported here.

Example using a configured development stub:

```sh
curl --data-binary @image.png \
  -H "Authorization: Bearer $STUB_AUTH_TOKEN" \
  -H 'X-Filename: image.png' \
  http://localhost:8080/v1/files
```

A successful response is `201` with `{"id":"<uuid>"}`. The row initially has
status `uploaded`, and its private storage key starts with `incoming/`.
Dimensions on the row remain unset until background processing completes.

Authentication, filename checks, and declared size checks happen before reading
the body. Actual streamed bytes are validated independently of Content-Length.
Validation spools to a private temporary file, then streams that file to storage.
Only after storage succeeds does a short PostgreSQL transaction insert the file
row and call River `InsertTx` with a `process_file` job containing `file_id`.
This step enqueues jobs; worker execution is implemented separately.

Validation errors return 400 (image, dimensions, minimum size), 413 (maximum
size), or 415 (type). Authentication failures return 401. Operational failures
return 500. A definite database failure rolls back the row and job and removes
the incoming object. If a commit acknowledgment is lost, the object is retained
because PostgreSQL may have committed; reconciliation can remove genuine orphans.

Export DATABASE_URL and apply both schema sets with `make migrate`, which runs
goose followed by River's embedded migrations. Existing goose-only databases
can run `make migrate-river`. Migration commands never run automatically in the
API process. The migration binary supports River up migrations only; goose down
commands affect application tables only.

Set STORAGE_BACKEND and storage settings from `.env.example`. Without DATABASE_URL,
the API still exposes health checks but uploads return 503. With a database,
authentication denies all uploads until an Authenticator is provided or both
development settings STUB_AUTH_USER_ID and STUB_AUTH_TOKEN are explicitly set.
The stub token needs at least 32 non-whitespace bytes and maps to an already
existing users row. It is an authentication interface stub, not user registration.

Unit tests: `go test ./internal/httpx ./internal/auth ./internal/config`.
Database integration tests: set TEST_DATABASE_URL to a migrated test PostgreSQL
database and run `go test ./internal/httpx -run TestUpload -count=1`.
They apply River migrations, verify HTTP 201 plus a committed row and job, and
force a post-enqueue failure to verify both inserts roll back together.
