# SafeDrop frontend

React 19, Vite, TypeScript, Tailwind v4, shadcn-style local UI components with
Radix primitives, React Router, TanStack Query, react-hook-form, and zod.

```sh
cd frontend
cp .env.example .env
npm install
npm run dev
```

The Vite server prints its local URL. The content area is intentionally empty;
Upload, My files, Sign in, and Create account are route shells for later steps.
The user menu truthfully shows a signed-out state. Authentication is not wired.
The theme toggle supports keyboard activation, follows system preference until
saved, and persists light/dark preference when browser storage is available.

`npm run build` checks types and builds dist/. `npm test` covers theme switching,
keyboard menu navigation, toast dismissal, and typed API error handling.
The npm lockfile is disabled to keep tracked files below 150 lines; direct
dependencies are declared in package.json. Use a generated lockfile in deployment
if relaxing that size rule to pin transitive dependencies.

## Configuration and structure

- VITE_API_URL: API base including /v1, absolute HTTP(S) URL or same-origin path.
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

The Go API currently has no cross-origin browser policy. For API screens, route
the frontend and /v1 through one origin or configure an explicit backend CORS
policy before using a separate API origin. The empty shell makes no API calls.
Production hosting must fall back to index.html for client routes.

Validation: six tests and the production build passed. The Vite dev layout was
checked in the browser in both light and dark mode, including the user menu.

References: [Vite env](https://vite.dev/guide/env-and-mode),
[shadcn Vite](https://ui.shadcn.com/docs/installation/vite),
[Radix Toast](https://www.radix-ui.com/primitives/docs/components/toast).
