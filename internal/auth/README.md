# Accounts and file ownership

Set `JWT_SECRET` to a securely generated random secret of at least 32 bytes.
The API requires it when `DATABASE_URL` is configured. The development stub
remains available to tests but is no longer used by the API.

POST `/v1/auth/register` or `/v1/auth/login` with JSON:

```json
{"email":"person@example.com","password":"a long unique password"}
```

Registration returns 201; login returns 200. Both return JSON `{"token":"..."}`; tokens expire after 900 seconds.
Pass the token in the
`Authorization: Bearer <token>` header for uploads, shares, and files.
Tokens use HS256 with an enforced algorithm, issuer, audience, and expiry.
There is no refresh or revocation endpoint; rotating the secret invalidates tokens.

Emails are trimmed and lowercased. Passwords must contain 12–1024 bytes.
Passwords use Argon2id v19, 64 MiB memory, three iterations, one lane, a
random 16-byte salt, and a 32-byte output. At most two password operations
run concurrently per API process; excess requests return 429. Unknown login
emails run a dummy password check. Credentials and tokens are never logged.

GET `/v1/files?limit=20&cursor=<next_cursor>` lists the caller's undeleted
files newest first. Limit is 1–100. The opaque cursor uses creation time and
UUID, so equal timestamps are ordered consistently. Omit the cursor initially;
an absent `next_cursor` ends pagination. Pagination is not a database snapshot.

GET `/v1/files/{id}` returns metadata. DELETE `/v1/files/{id}` soft deletes
and returns 204. Missing, foreign-owned, and already deleted files return 404.
All three endpoints use owner-scoped SQL; private storage keys are omitted.
Soft deletion immediately prevents access and sharing; it does not immediately
remove storage objects. Public share download checks also exclude deleted files.

Run `go test ./...`. Set `TEST_DATABASE_URL` to a migrated disposable
PostgreSQL database and run `go test -p 1 ./... -count=1` for integration tests,
including registration/login, pagination, foreign-owner rejection, and deletion.
