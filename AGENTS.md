# AI coding agent instructions

This file provides guidance to AI coding agents when working with code in this repository.

## Project Overview

This is a calendar application with a Go backend, REST API and an embedded TypeScript web frontend. 
The codebase includes OpenAPI specs, SQLite storage, and multiple calendar view components (day, week, month, year, schedule).
There is also an Android (Kotlin/Compose) app using the same REST API in another repository.

## Build & Run

```bash
./build.sh                              # compile TypeScript, generate API code, build binary
./mycal                                 # serves on :8080
./mycal -port 3000 -data /path/to/data  # custom address and data path
```

`ogen`, `tsc` and `openapi-typescript` must be on `$PATH`

## Code generation

Two code generation steps run automatically in `build.sh`:

**TypeScript types from OpenAPI** (`openapi-typescript`):
- `web/ts/api/types.ts` is generated from `openapi.yaml` by `openapi-typescript` — never manually edit it.
- The frontend uses these generated types directly (e.g. `components['schemas']['Event']`), aliasing them per-file where needed. There is no hand-maintained re-export layer.
- App-only types not in the API (e.g. `AppConfig`) live next to their logic, e.g. exported from `web/ts/util/config.ts`.
- After changing `openapi.yaml`, run `openapi-typescript openapi.yaml -o web/ts/api/types.ts` (or `./build.sh`) before running `tsc`.

**Go HTTP server stubs** (`ogen`):
- `internal/api/` is generated from `openapi.yaml` using [ogen](https://ogen.dev/).
- Always run `go generate ./...` (or `./build.sh`) before building after changing `openapi.yaml`.
- Never manually edit any file in `internal/api/` — all changes are overwritten by `go generate`.
- To add or change API behaviour, edit `openapi.yaml` and regenerate, then update the implementation in `internal/handler/impl.go`.

## Tests

Go tests should use the `github.com/stretchr/testify` library for assertions. 
Use `require` for critical checks (like error handling) to stop test execution early, and `assert` for other checks.

```bash
go test ./...                           # all tests
go test ./internal/repository/ -v       # repository tests only
go test ./internal/repository/ -run TestList  # single test
```

## E2E Tests

Playwright end-to-end tests live in `e2e/`. The server must be running on port 8089 before running them.

```bash
# Start server for E2E tests (use a separate DB to avoid interference)
# -public-url must match the test baseURL origin (http://localhost:8089) or CSRF
# rejects every mutating request (POST/PATCH/DELETE) with 403 and those tests fail.
./mycal -port 8089 -public-url http://localhost:8089 -data /tmp/claude/ &

# Run tests
cd e2e && playwright-test
```

*Important:* Use the `playwright-test` command to run the e2e tests and nothing else.

## Verification

Use the `playwright-cli` skill to verify that the web frontend looks reasonable.

## Architecture

Go backend with embedded Preact+JSX frontend. TypeScript source in `web/ts/`, compiled by `tsc` to `web/static/` which Go embeds. Single binary serves both JSON API and static files.

**Layered backend** (`main.go` wires everything):
- **model** → Event struct, request types (Create/Update), validation. Datetimes are RFC 3339 strings throughout.
- **repository** → `EventRepository` interface + SQLite implementation (`modernc.org/sqlite`, pure Go). Schema auto-created on startup.
- **service** → Business logic wrapping repository. Returns typed sentinel errors (`ErrNotFound`, `ErrValidation`).
- **handler** → HTTP handlers using Go 1.22+ `ServeMux` routing patterns (`"GET /api/v1/events/{id}"`). JSON helpers and middleware (logging, recovery, CORS).

**Frontend** (`web/static/`, embedded via `web/embed.go`):
- Preact loaded from vendor files via import map in `index.html` (`vendor/preact/preact.module.js`, `…/hooks.module.js`, `…/jsx-runtime.module.js`)
- `web/ts/app.tsx` is the root component (compiled to `web/static/app.js` by `tsc`)
- Native `<dialog>` element for the event form (no client-side routing)
- API calls go through the centralized client in `api/client.ts`: a single `request<T>` helper with network retry, 401-reload, 404 `NotFoundError`, exposed as a namespaced `api.*` surface (`api.events`, `api.calendars`, `api.feeds`, `api.preferences`, `api.import`)
- Toasts are an observable store (`util/toast.ts`); any module calls `showToast(...)` / `showNetworkErrorToast(...)` and the `<Toast>` component subscribes
- Main stylesheet is `web/static/app.css`

**TypeScript** (`web/ts/`, compiled to `web/static/`):
- `web/ts/tsconfig.json`: target ES2020, module/moduleResolution Node16, `jsx: react-jsx` + `jsxImportSource: preact` (automatic runtime — no `import { h }`), `strict: true`
- `web/ts/package.json` contains `{ "type": "module" }` — **required** so `tsc` emits ESM under Node16; without it the output is CommonJS `require()` and the browser shows a blank page
- Directory taxonomy: `api/` (client + generated types), `util/` (lowercase utilities), `components/` (reusable widgets), `layout/` (Nav, sidebars), `views/` (calendar views). Component files are `PascalCase.tsx`; utilities are `lowercase.ts`
- Vendored preact `.d.ts` live under `web/ts/vendor/preact/` (referenced via tsconfig `paths`, excluded from the build); Leaflet/Quill ambient declarations live in `web/ts/vendor/`
- Relative imports use `.js` extensions (TypeScript ESM convention — tsc resolves `.ts`/`.tsx`, emits `.js`)
- Everything under `web/static/` except `web/static/vendor/**` is generated by `tsc` — do not edit directly

**Key design decisions:**
- `UpdateEventRequest` uses pointer fields for partial updates (nil = unchanged)
- Repository returns `nil, nil` for not-found on GetByID; service layer maps this to `ErrNotFound`
- Repository Delete returns `sql.ErrNoRows` for not-found; service maps to `ErrNotFound`
- List endpoint requires `from`/`to` query params; uses overlapping range query (`start_time < to AND end_time > from`)
- iCalendar (RFC 5545) feed at `/calendar.ics` and `/api/v1/events.ics` — `internal/ical` package encodes events, no external dependency
- The REST API is mounted at the specific `/api/v1/` prefix (not the broader `/api/`) so the compiled frontend module served at `/api/client.js` (from the `web/ts/api/` source dir) falls through to the static file handler. Keep this mount narrow.

## Go development

Always run `go mod tidy` after modifying the `go.mod` file.

## API

The API is defined in the OpenAPI spec in `openapi.yaml`, update it whenever the API is changed. 
The OpenAPI spec is used to generate server and client code, so it must be accurate.

In addition to the web UI here, the API is consumed by mobile clients outside of this repository. Keep the API backwards 
compatible whenever possible, but breaking changes are acceptable if hard to avoid. Make sure to report any breaking changes 
so that clients can be updated.

## Database

Make migrations when changing the database schema, assume there is production data that needs to be preserved. 

## Version control

Git is used for version control. When creating new files, make sure to add them to Git.

## Security Guidelines

### Input sanitization — apply uniformly across all write paths

- Sanitize HTML (using `sanitize.HTML`) on **every** write path: interactive create, interactive update, iCal import, and override import. Never sanitize only on creation and skip it on update.
- Apply the same validation to imported data that you apply to interactively-entered data. An import path is not a trusted source.
- In the frontend, never assign HTML to `innerHTML` directly. Use the library's own API (e.g. `quill.setContents(quill.clipboard.convert(value))`) so the library's sanitizer runs.

### URL and scheme validation

- Validate URL schemes (allow only `http:`, `https:`, `mailto:`) on **all** paths — both interactive and import. Never store a URL without scheme validation.
- In the frontend, validate `href` values before rendering them as links; fall back to plain text for disallowed schemes.

### SSRF prevention

- Validate URLs at **connection time** (inside a custom `DialContext`), not just before the fetch. Validate-then-fetch is a TOCTOU race (DNS rebinding).
- Set `CheckRedirect` to re-validate every redirect target; do not follow redirects blindly to potentially private addresses.
- Re-run URL validation on every periodic re-fetch of stored feed URLs, not just on initial registration.
- Return generic error messages to the client for DNS/network errors (e.g. "could not resolve URL host"). Log details server-side only.

### HTTP server hardening

- Apply a global request body size limit (`http.MaxBytesHandler` or `MaxBytesReader`) to **all** handlers, not just file upload endpoints.
- Set both `ReadTimeout` and `ReadHeaderTimeout` on `http.Server` to prevent Slowloris-style attacks.

### Security headers and CSP

- Keep the Content-Security-Policy tight. When adding new outbound resources (tile servers, CDNs, APIs), update **both** `connect-src` and any other relevant directive (`img-src`, `script-src`) at the same time.
- Include `frame-ancestors 'none'` in the CSP in addition to `X-Frame-Options: DENY`; modern browsers prefer the CSP directive.
- Avoid `'unsafe-inline'` in `style-src`; prefer nonces or hashes.

### API design

- GET requests must never modify the database or create side effects. Filtering by a non-existent resource name should return an empty result, not create the resource.
- Add `maxLength` constraints in `openapi.yaml` for all string query parameters, not just request body fields.
