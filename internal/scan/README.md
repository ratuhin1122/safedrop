# clamd scanning

NewClient takes a TCP host:port, a positive scan timeout, and a maximum byte count.
Scan streams an io.Reader with a fixed 32 KiB buffer and returns Clean or Infected
with a signature name. Errors have no clean verdict. Callers normally own and close
their source; cancellation can also close an io.Closer source to unblock reads.

Each file gets a separate TCP connection and the NUL-terminated `zINSTREAM`
command. Four-byte big-endian lengths precede data chunks; a zero length finishes
the stream. Replies are NUL-terminated and limited to 4 KiB. These formats follow
the [clamd protocol](https://docs.clamav.net/manual/Usage/ClamdProtocol.html).
Only `stream: OK` is clean. FOUND is infected; daemon errors, truncation, unknown
replies, timeouts, and connection failures are errors and fail closed.

cmd/worker uses CLAMD_ADDRESS (default localhost:3310), CLAMD_TIMEOUT (30s), and
MAX_UPLOAD_BYTES (25 MiB). Compose supplies clamd:3310. Keep the daemon's
StreamMaxLength at least as large as MAX_UPLOAD_BYTES and maintain its production
signature database through ClamAV's normal update process.

ScanProcessor reads private incoming storage, closes the reader, and rejects and
deletes infected objects. If deletion fails, River retries cleanup on the rejected
file without rescanning. Storage outages that exhaust all retries require job
replay or later rejected-object cleanup. Missing incoming objects are rejected.
Successful scans advance scanning to processing. A processing retry skips scanning.
The next image-processing stage can be supplied through ScanProcessor.Next.
Until then, clean jobs snooze for one minute without exhausting their attempts;
they remain private in incoming storage and are not marked ready.

Run `go test ./internal/scan` for protocol, streaming, timeout, and fail-closed
tests. The [EICAR test string](https://www.eicar.org/download-anti-malware-testfile/)
is constructed only in memory to avoid host antivirus quarantining a fixture.

To run real clamd integration tests with an isolated test signature database:

```sh
docker run --rm -p 127.0.0.1:13310:3310 --entrypoint clamd \
  -v "${PWD}/internal/scan/testdata:/testdb:ro" \
  clamav/clamav:stable_base --config-file=/testdb/clamd.conf
```

In another shell, export TEST_CLAMD_ADDRESS=127.0.0.1:13310 and run
`go test ./internal/scan -run TestLiveClamd -count=1 -v`. Also set TEST_DATABASE_URL
to a migrated PostgreSQL test database and run
`go test ./internal/jobs -run 'TestWorkerLiveClamd|TestScan|TestInfected|TestClean' -count=1 -v`.
The committed test database contains only an EICAR hash signature; use it only for
these tests. Production Compose uses the normal clamav/clamav image and databases.
