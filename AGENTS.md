# AI coding agent instructions

This file provides guidance to AI coding agents when working with code in this repository.

## Build & Run

```bash
go build -tags netgo -o /tmp/claude/mycal .       # build single binary
/tmp/claude/mycal                     # serves on :8080, uses mycal.db
/tmp/claude/mycal -addr :3000 -db /path/to/cal.db  # custom address and DB path
```

The coding agent should always compile the application into `/tmp/claude` and run it from there.

## Tests

```bash
go test ./...                           # all tests
go test ./internal/repository/ -v       # repository tests only
go test ./internal/repository/ -run TestList  # single test
```

## E2E Tests

Playwright end-to-end tests live in `e2e/`. The server must be running on port 8089 before running them.

```bash
# Start server for E2E tests (use a separate DB to avoid interference)
/tmp/claude/mycal -addr :8089 -db /tmp/claude/mycal-e2e.db &

# Run tests
cd e2e && bash playwright-test
```

## Verification

Use the `playwright` MCP to verify that the web frontend looks reasonable.

## Architecture

Go backend with embedded Preact+HTM frontend (no JS build step). Single binary serves both JSON API and static files.

**Layered backend** (`main.go` wires everything):
- **model** → Event struct, request types (Create/Update), validation. Datetimes are RFC 3339 strings throughout.
- **repository** → `EventRepository` interface + SQLite implementation (`modernc.org/sqlite`, pure Go). Schema auto-created on startup.
- **service** → Business logic wrapping repository. Returns typed sentinel errors (`ErrNotFound`, `ErrValidation`).
- **handler** → HTTP handlers using Go 1.22+ `ServeMux` routing patterns (`"GET /api/v1/events/{id}"`). JSON helpers and middleware (logging, recovery, CORS).

**Frontend** (`web/static/`, embedded via `web/embed.go`):
- Preact + HTM loaded from esm.sh CDN via import map in `index.html`
- `app.js` is the root component managing state and coordinating child components
- Native `<dialog>` element for the event form (no client-side routing)
- API calls go through `lib/api.js` fetch wrapper

**Key design decisions:**
- `UpdateEventRequest` uses pointer fields for partial updates (nil = unchanged)
- Repository returns `nil, nil` for not-found on GetByID; service layer maps this to `ErrNotFound`
- Repository Delete returns `sql.ErrNoRows` for not-found; service maps to `ErrNotFound`
- List endpoint requires `from`/`to` query params; uses overlapping range query (`start_time < to AND end_time > from`)
- iCalendar (RFC 5545) feed at `/calendar.ics` and `/api/v1/events.ics` — `internal/ical` package encodes events, no external dependency

## API

The API docs are in `docs/API.md`, update them whenever the API is changed. 
Disregard backwards compatibility, there are no other users of the API than the internal web frontend.

## Database

Do not do database migrations, assume that we will start with an empty database.

## Version control

Git is used for version control. When creating new files, make sure to add them to Git.

## Feature Backlog

See `TODO.md` for the list of planned features.

**Important**: Always read `TODO.md` at the start of a session and take note of the not yet done features in the "Current" section. 
When implementing a new feature from the list, mark it as complete (`[x]`) in TODO.md after the implementation is done, tested, verified and commited.
