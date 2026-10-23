# POST /v1/files

Send multipart/form-data with exactly one file part named `file` and a display
filename. Pass Authorization: Bearer <token>. Filenames are at most 255 bytes,
without paths/control characters. Extensions and client MIME do not determine type.

```sh
curl -F 'file=@image.png' -H "Authorization: Bearer $TOKEN" http://localhost:8080/v1/files
```

Success is 201 with `{"id":"<uuid>","status":"uploaded"}`. The private object
starts in incoming/. Dimensions on the row are saved by background processing.
Legacy raw-body requests with X-Filename remain supported.

Authentication, admission, and cheap header checks precede body reads. Multipart
parsing streams; validation spools a bounded file to private temporary disk,
checks actual bytes/type/dimensions, then streams it to incoming/ storage.
MAX_UPLOAD_BYTES applies to file bytes; a separate 64 KiB envelope allowance
bounds multipart headers. Additional parts are rejected before persistence.

A PostgreSQL transaction charges usage, creates the file, inserts the River job,
and records a usage event. Definite failures roll back and remove the incoming
object. An uncertain commit retains the object for later reconciliation.

JSON errors have code and message. Size minimum/invalid image returns 400,
maximum size 413, type 415, pixel limit 422, and admission/quota 429 with
Retry-After. Virus rejection is asynchronous and exposed through reject_reason.

Apply goose and River migrations with `make migrate` before starting API/worker.
Compose does not migrate automatically; see the root README for container setup.
Go binaries read exported env variables; .env is loaded by Compose only.

See [API contract](../../docs/api.md), [accounts](../auth/README.md), and
[integration tests](../../docs/integration-tests.md). Set TEST_DATABASE_URL to a
migrated disposable database to run database-backed HTTP tests.
