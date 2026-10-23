# API contract

The base URL is `/v1`. JSON request bodies use Content-Type: application/json.
Protected routes require `Authorization: Bearer <token>`; tokens expire in 15 minutes.
Errors are JSON `{ "code": "...", "message": "..." }`, including routing and rate-limit errors.

## Authentication

POST `/auth/register` and POST `/auth/login` accept email and password.
Registration optionally accepts turnstile_token when Turnstile is configured.
Success returns `{ "token": "..." }` (201 registration, 200 login).

## Uploads

POST `/files` accepts multipart/form-data with exactly one part named `file`.
The part must include a display filename, without directory components.
The browser/client sets the multipart boundary; do not manually set Content-Type.

```sh
curl -F 'file=@image.png' -H "Authorization: Bearer $TOKEN" http://localhost:8080/v1/files
```

Success: 201 `{ "id": "uuid", "status": "uploaded" }`.
The detected type ignores the extension and part's Content-Type. File byte limits
exclude multipart overhead; envelope overhead has a separate 64 KiB allowance.
Legacy raw-body uploads with X-Filename remain supported. Files spool to disk;
multipart parsing never buffers a whole file in memory.

## Files and variants

GET `/files?cursor=&limit=` returns `{ "items": [...], "next_cursor": "..." }`.
An empty next_cursor ends pagination; default limit is 20, allowed range 1–100.
GET `/files/{id}` returns one item. DELETE `/files/{id}` soft-deletes it (204).
Both metadata and downloads enforce ownership; foreign or missing files return 404.

Items include id, original_name, mime, size, width, height, status, created_at,
reject_reason, and variants. Status is uploaded, scanning, processing, ready,
or rejected. Unknown dimensions/reject_reason are null. Rejected reasons are
`virus` or `processing_error`; scan signatures and internal errors are not exposed.

```json
{
  "variants": {
    "w375": {"width":375,"height":250,"size":12345,"mime":"image/webp","url":"/v1/files/uuid/variants/375"},
    "w768": null,
    "w1200": null
  }
}
```

Each available variant is a metadata object. Its URL requires the same Bearer
token; private storage keys never appear. Missing/not-ready variants are null.
GET `/files/{id}/variants/{width}` streams an owned ready variant (375/768/1200).

## Shares

POST `/files/{id}/share` accepts expires_at (future RFC3339), optional password,
and optional positive max_downloads. It requires an owned ready file and returns
201 `{ "url": "/s/token", "expires_at": "..." }`. The old plural POST route is an alias.
GET `/files/{id}/shares` returns `{ "items": [...] }` with each share's id,
file_id, expires_at, max_downloads, download_count, and password_protected.
Only the create response reveals the URL; stored token hashes cannot recreate it.
DELETE `/shares/{id}` revokes an owned link (204); foreign/missing IDs return 404.
Revocation cannot retract an already-authorized stream or unexpired presigned URL.

Public GET `/s/{token}` streams the attachment or redirects when presigning is
enabled. Password-protected links use HTTP Basic auth (username ignored).
Expired, exhausted, revoked, missing, and deleted links return 410; incorrect or
missing passwords return 401. Wrong passwords do not consume downloads.

## Usage

GET `/me/usage` returns bytes_today, bytes_limit, uploads_this_hour, uploads_limit.
Bytes count committed uploads for the UTC day. uploads_this_hour counts committed
uploads in the rolling last hour, including subsequently deleted files. Rejected
admissions do not add usage. uploads_limit is the configured per-user upload
window limit; the window remains UPLOAD_RATE_WINDOW (default 1m), not one hour.
The response is informational; in-memory admission attempts and concurrent uploads
can also cause 429, and daily byte quotas remain transactional in PostgreSQL.

## Errors and asynchronous rejection

| Status | code | Meaning |
| --- | --- | --- |
| 400 | invalid_request | Malformed body, minimum size, invalid image/settings. |
| 401 | unauthorized | Invalid JWT or wrong/missing share password. |
| 404 | not_found | Missing or foreign-owned resource. |
| 409 | conflict | Account already exists or file is not ready. |
| 410 | gone | Public link expired, exhausted, revoked, or unavailable. |
| 413 | too_large | Actual file bytes or bounded envelope exceed limits. |
| 415 | bad_type | File signature is outside the image allowlist. |
| 422 | pixel_limit | Declared dimensions exceed MAX_IMAGE_PIXELS. |
| 429 | rate_limited | Admission, concurrency, quota, or API bucket; Retry-After is seconds. |
| 500 / 503 | internal_error / unavailable | Operational failure; safe public message. |

Virus scanning is asynchronous: an accepted upload initially returns 201.
Poll GET `/files/{id}`; a virus yields status rejected and reject_reason virus.
Raw EICAR text is rejected with 415 by the image allowlist before scanning.
Apply migration 00005 and deploy API and worker together before using this contract.

Validation: sqlc generate, goose migrations, the full test suite with an isolated
PostgreSQL 18 database, go build ./..., and go vet ./... passed. Optional live
ClamAV and native govips checks remain environment-dependent.
