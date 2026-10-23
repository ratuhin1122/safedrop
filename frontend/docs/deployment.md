# Container deployment and tests

For the complete project, run make up from the repository root, then open
http://localhost:3001 (WEB_PORT overrides it). This includes migrations and all
backend dependencies. make down stops containers while retaining data.

For only the frontend container:

    docker compose up --build -d web

Open http://localhost:3001 (WEB_PORT overrides 3001). The nginx image runs as
a non-root user on container port 8080. /healthz checks static-server health.
The web service starts independently; API calls need the api service running.
For a working backend, configure JWT_SECRET, run migrations as described in the
root README, then start api and worker with their database and scanner services.

The build uses Node 24 and bakes in VITE_API_URL=/v1. Browser requests to /v1/
and non-HTML /s/ requests proxy to api:8080. HTML /s/{token} visits and deep
links such as /files get the SPA. Missing assets return 404, not index.html.
Uploads stream through nginx; the Go service enforces file-size limits.
Set TRUSTED_PROXY_CIDRS to the nginx container network CIDR if forwarded client
addresses are needed. nginx overwrites incoming forwarding headers.

VITE_TURNSTILE_SITE_KEY and MAX_UPLOAD_BYTES become frontend build arguments.
Rebuild web after changing them. Private backend secrets never enter this image.
The frontend build context excludes .env files, node_modules, and audit output.

CSP allows same-origin modules and API requests, blob/data image previews,
inline styles used by the UI, and the optional official Turnstile script/frame.
The theme bootstrap is a local external script, so inline scripts stay blocked.
If enabling presigned cross-origin S3 downloads, explicitly add the storage
origin to connect-src before deployment. Terminate HTTPS at your edge proxy.
Access logs omit query strings and redact share-token paths. Share-proxy and
SPA HTML error logs are suppressed to keep request URLs out of diagnostics.

From frontend/, npm test runs Vitest with Testing Library. Coverage includes
size boundaries, 429 seconds and HTTP-date countdowns, stepper transitions and
polling stops, and public 200/401/404/410 screens including download-time races.

Container smoke checks:

    docker compose config --quiet
    docker compose exec web nginx -t
    curl -I http://localhost:3001/files
    curl -H "Accept: text/html" -I http://localhost:3001/s/example
    curl -I http://localhost:3001/healthz

The SPA responses include Content-Security-Policy. API responses retain API
status codes and never fall back to HTML.
