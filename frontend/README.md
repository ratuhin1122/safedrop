# SafeDrop frontend

React 19, Vite, TypeScript, Tailwind v4, shadcn-style local UI components with
Radix primitives, React Router, TanStack Query, react-hook-form, and zod.

```sh
cd frontend
cp .env.example .env
npm install
npm run dev
```

The Vite server prints its local URL. Home has a local upload queue; My files remains a content shell.
Sign in and Create account are working validated forms. My files requires login.
The user menu reflects the session and offers sign-out when authenticated.
The theme toggle supports keyboard activation, follows system preference until
saved, and persists light/dark preference when browser storage is available.

`npm run build` checks types and builds dist/. `npm test` covers theme switching,
keyboard menu navigation, toast dismissal, and typed API error handling.
The npm lockfile is disabled to keep tracked files below 150 lines; direct
dependencies are declared in package.json. Use a generated lockfile in deployment
if relaxing that size rule to pin transitive dependencies.

## Configuration and structure

- VITE_API_URL: API base including /v1; default /v1 uses the development proxy.
- API_PROXY_TARGET: Vite-only Go API origin; configured in .env, not bundled.
- VITE_MAX_UPLOAD_BYTES: per-file pre-check limit; default 26214400 (shown as 25 MB), matching the API's 25 MiB.
- VITE_TURNSTILE_SITE_KEY: public browser key; no private keys belong in VITE_*.
- src/api: schema-validated client; callers pass a zod response schema and token.
- src/components: header/layout, theme/toast providers, editable shadcn UI primitives.
- src/pages: empty route shells and a not-found page.
- src/hooks: useTheme and useToast.
- src/lib: env validation, theme utilities, and class-name merging.
- src/styles.css: semantic light/dark colors, radii, and responsive spacing tokens.

Toast usage: `const { toast } = useToast()` then
`toast({ title: "Upload failed", description: error.message, variant: "destructive" })`.
F8 focuses notifications; Escape dismisses the focused toast. Closing pauses
while hovered/focused, and reduced-motion preferences disable transitions.

The Go API has no cross-origin browser policy. The example env makes Vite proxy
/v1 and /s to API_PROXY_TARGET. Start the migrated Go service separately. In
production, proxy these paths through one origin or configure backend CORS.
Production hosting must fall back to index.html for client routes.

## Authentication

AuthProvider stores the JWT in memory and sessionStorage (per origin/tab), restores
it on reload, and clears it on sign-out or a protected 401. ProtectedRoute redirects
signed-out /files visits to /login and preserves the local return path. Session
changes clear TanStack Query's cache. The backend remains the authorization authority.
Forms provide associated inline errors, loading/disabled states, password visibility,
and registration passphrase guidance. VITE_TURNSTILE_SITE_KEY enables the explicit
Cloudflare widget on registration only; empty skips its script and widget. Configure
the matching TURNSTILE_SECRET on the API. Expiry, errors, and failed submissions
invalidate tokens; widgets are removed on unmount. Turnstile tokens are not logged or stored.
See [Cloudflare rendering](https://developers.cloudflare.com/turnstile/get-started/client-side-rendering/).

## Upload queue

Home supports browsing and drag/drop of multiple JPEG, PNG, WebP, and HEIC files.
Limits are visible before selection. Extension checks are hints, not content
validation; the backend verifies bytes and scans accepted uploads.
Rejected rows show per-file size/type/empty-file messages and can be removed.
Selecting or dropping files makes no network calls and reads no file contents.
The queue is held only in component memory and resets when leaving the page.
Upload files starts the selected files using authenticated XMLHttpRequest,
with up to three active requests, per-file progress, cancel, and manual retry.
A 201 shows "Uploaded, checking file..."; scanning completion is not assumed.
413/415/422 errors have specific messages. A 429 pauses new uploads and retries
with a live Retry-After countdown; missing/invalid headers default to 60 seconds.
Waiting files resume when the pause expires; the failed file needs manual retry.
Unmounting aborts active requests. Cancellation cannot undo server-side commits.
Retries after an uncertain network failure may create another server file.
Validation: production build and all 40 tests pass, including mocked XHR progress,
three concurrent slots, cancellation/retry, and a live 429 countdown that disables Upload files.

## Processing status

Uploaded queue rows use useFileStatus with GET /files/{id} through TanStack Query.
Polling starts immediately, then waits 2, 4, 8, and at most 10 seconds between
checks. Ready/rejected states stop automatic requests, including on remount.
Hidden tabs disable polling and cancel in-flight requests; visible tabs resume
unfinished queries. Query cancellation also propagates on unmount.
The stepper shows Uploaded, Scanning, Processing, Ready. Known virus, invalid-image,
and pixel-limit reasons get friendly rejection messages; unknown reasons use a
safe fallback. The current backend emits virus or generic processing_error.
Temporary status-request failures show the API message and a Check again action;
401/403/404/410 stop automatic checks. Status 429 respects Retry-After when polling.
Validation: production build and all 47 tests pass, including step transitions,
terminal polling stop, the 10-second cap, visibility pause/resume, and cancellation.
Reference: [TanStack Query polling](https://tanstack.com/query/latest/docs/framework/react/guides/polling).

Validation: type checking, production build, auth/route/session tests, and mocked
Turnstile lifecycle tests pass. Browser checks confirmed /files redirects to
/login, inline validation, and dark registration. Live Turnstile verification
requires a configured site key and matching server secret.

References: [Vite env](https://vite.dev/guide/env-and-mode),
[shadcn Vite](https://ui.shadcn.com/docs/installation/vite),
[Radix Toast](https://www.radix-ui.com/primitives/docs/components/toast).

## Container and regression tests

See [container deployment and tests](docs/deployment.md) for the web service,
same-origin proxy, CSP, environment build arguments, and verification commands.
