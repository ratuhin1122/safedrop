# Security integration tests

Use a disposable PostgreSQL database; the suite creates and deletes fixture rows.
Apply application migrations with goose, then export TEST_DATABASE_URL to the
same database. River's embedded migrations run from the integration fixtures.

```sh
export TEST_DATABASE_URL='postgres://safedrop:safedrop@localhost:5432/safedrop_test?sslmode=disable'
goose -dir migrations postgres "$TEST_DATABASE_URL" up
go test -p 1 ./... -count=1
```

In PowerShell, set `$env:TEST_DATABASE_URL` instead of using `export`.
Without TEST_DATABASE_URL, database integration tests explicitly skip.

| Scenario | Integration test and assertions |
| --- | --- |
| Image bomb | TestUploadSafetyPostgres: valid PNG header declares 100000×100000 pixels; HTTP 422 for pixel limit, no row/job/usage/object/temp-file leaks. |
| PNG renamed .jpg | TestUploadSafetyPostgres: HTTP 201 despite JPEG filename and Content-Type, stored MIME is image/png, original bytes preserved, one row/job/usage entry. |
| EICAR upload | TestUploadSafetyPostgres: raw EICAR receives 415 under the image allowlist, with no persisted effects. |
| EICAR background scan | TestUploadAndScanShareInjectedMetrics: PNG containing EICAR is uploaded and enqueued, then rejected; incoming object deleted and no final variant published. |
| Oversize | TestUploadSafetyPostgres: declared length and chunked body both receive 413 without persisted effects. |
| Below minimum | TestUploadSafetyPostgres: declared length and chunked tiny bodies both receive 400 without persisted effects. |
| Expired share | TestPostgresGoneResponses: expired link returns 410 without consuming a download. |
| Cross-user access | TestAccountsAndFileOwnership: foreign GET/DELETE return 404, lists include only owned files, owner can still read the foreign-access target. |
| Rate limit | TestUploadWindowPostgres: first 20 uploads commit, 21st returns 429 with positive Retry-After before reading its body; only 20 rows/jobs/usage increments. |

The deterministic scan pipeline uses the production clamd INSTREAM client with
a local TCP protocol fixture that recognizes EICAR; conversion uses a test
encoder. It does not establish that an installed ClamAV database or libvips works.
For live scanner checks, additionally set TEST_CLAMD_ADDRESS to clamd's TCP
address and run `go test ./internal/scan ./internal/jobs -run Live -count=1`.
Native govips tests require a supported CGO build with libvips installed.

Validation: `go test -mod=readonly -p 1 ./... -count=1` passed with
TEST_DATABASE_URL set to an isolated local PostgreSQL 18 database. All eight
requested scenarios ran; optional live ClamAV and native govips checks were skipped.
