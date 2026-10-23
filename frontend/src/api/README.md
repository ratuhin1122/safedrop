# Typed API client

Import endpoint functions and inferred types from `@/api`.
Every JSON response is validated with zod; input IDs, pagination, credentials,
share settings, and upload File objects are validated before sending.

| Function | Endpoint |
| --- | --- |
| register / login | POST /v1/auth/register, POST /v1/auth/login |
| uploadFile | POST /v1/files, one multipart file part |
| listFiles / getFile / deleteFile | GET /v1/files, GET/DELETE /v1/files/{id} |
| createShare | POST /v1/files/{id}/share |
| listShares / deleteShare | GET /v1/files/{id}/shares, DELETE /v1/shares/{id} |
| getUsage | GET /v1/me/usage |
| downloadShare | Public GET /s/{token}, optional Basic password |
| downloadVariant | GET /v1/files/{id}/variants/{width} |
| getHealth | Public GET /healthz |

VITE_API_URL includes /v1 (and any deployment prefix). Public routes remove
that final /v1 segment while preserving origin and prefix. No host is hardcoded.
The server's returned URLs are metadata; download functions construct trusted
endpoint paths rather than sending JWTs to response-provided URLs.

Successful register/login stores the JWT in memory and sessionStorage. AuthProvider
restores it on reload; setAccessToken(null) clears both copies and notifies consumers.
Protected requests automatically
include the current JWT; apiRequest also accepts an explicit token override.

Protected 401 responses clear the token and redirect to /login, unless already
there. Login/register failures and public share-password challenges throw an
ApiError without redirecting or sending the user's JWT. Their forms can display
the server's specific message and retry. Public share passwords use UTF-8 Basic
authentication; they are never placed in a URL.

ApiError exposes code, message, status, and the original Retry-After header on
429. retryAfterSeconds accepts both integer seconds and HTTP dates. Missing or
invalid retry headers yield null. Network failures and invalid response formats
have explicit error codes. An aborted request preserves its cancellation error.
Each endpoint accepts an optional AbortSignal as its final argument.

deleteFile/deleteShare validate a 204 empty response and return undefined.
Downloads return metadata plus a ReadableStream; pipe it to a destination or
cancel it when unused. The client never collects downloaded files into a blob
or array buffer. Public presigned redirects follow normal fetch behavior; the
storage origin must permit CORS for browser streaming when using another origin.

Validation: `npx tsc --noEmit` passed; `npm test` passed all 18 tests from frontend/.
