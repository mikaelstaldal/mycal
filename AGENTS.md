# CLAUDE.md

This file provides guidance to AI coding agents when working with code in this repository.

## Build & Run

```bash
go build -o mycal .        # build single binary
./mycal                     # serves on :8080, uses mycal.db
./mycal -addr :3000 -db /path/to/cal.db  # custom address and DB path
```

## Tests

```bash
go test ./...                           # all tests
go test ./internal/repository/ -v       # repository tests only
go test ./internal/repository/ -run TestList  # single test
```

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
