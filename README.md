# mycal

A personal calendar application with a Go backend, SQLite storage, and a Preact web frontend.

## Features

- Monthly calendar grid with event display
- Create, view, edit, and delete events
- Color-coded events
- iCalendar (RFC 5545) feed for subscribing from other calendar apps
- JSON REST API for future native clients
- Single binary with embedded frontend — no JS build step

## Getting Started

```bash
go build -o mycal . && ./mycal
```

Open http://localhost:8080 in your browser.

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:8080` | Listen address |
| `-db` | `mycal.db` | SQLite database file path |
| `-basic-auth-file` | *(disabled)* | Path to htpasswd file for HTTP basic authentication |

### Authentication

HTTP basic authentication can be enabled by providing an [htpasswd](https://httpd.apache.org/docs/current/programs/htpasswd.html) file with bcrypt-hashed passwords:

```bash
# Create a new htpasswd file with a user (requires apache2-utils or httpd-tools)
htpasswd -Bc htpasswd admin

# Start with authentication enabled
./mycal -basic-auth-file htpasswd
```

When enabled, all endpoints (UI, API, and iCalendar feed) require valid credentials. The browser will prompt for a username and password automatically.

## API

All endpoints are under `/api/v1`. Datetimes use RFC 3339 format (or `YYYY-MM-DD` for all-day events). Errors return `{"error": "message"}`.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/events?from=...&to=...` | List events in a time range |
| GET | `/api/v1/events?q=...` | Search events by text |
| POST | `/api/v1/events` | Create an event |
| GET | `/api/v1/events/{id}` | Get a single event |
| PUT | `/api/v1/events/{id}` | Update an event (partial) |
| DELETE | `/api/v1/events/{id}` | Delete an event (add `?instance_start=<RFC3339>` to exclude a single recurrence instance) |
| POST | `/api/v1/import` | Import events from iCalendar data |
| POST | `/api/v1/import-single` | Import a single event from iCalendar data |
| GET | `/api/v1/events.ics` | iCalendar feed (all events) |
| GET | `/calendar.ics` | iCalendar feed (convenience URL) |

### Event Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | int | Event ID (read-only) |
| `title` | string | Event title (required, max 500 chars) |
| `description` | string | Event description (max 10000 chars) |
| `start_time` | string | Start time in RFC 3339 or `YYYY-MM-DD` for all-day (required) |
| `end_time` | string | End time in RFC 3339 or `YYYY-MM-DD` for all-day (required, must be after start) |
| `all_day` | bool | Whether this is an all-day event |
| `color` | string | Color hex code |
| `recurrence_freq` | string | `""`, `"DAILY"`, `"WEEKLY"`, `"MONTHLY"`, or `"YEARLY"` |
| `recurrence_count` | int | Number of recurrences (0–1000) |
| `recurrence_until` | string | Recurrence end date in RFC 3339 |
| `recurrence_interval` | int | Repeat every N periods (0 or 1 = every, 2 = every other, etc.) |
| `recurrence_by_day` | string | Comma-separated days: `"MO,WE,FR"` or ordinal `"2MO,-1FR"` |
| `recurrence_by_monthday` | string | Comma-separated month days: `"15,30"` or negative `"-1"` |
| `recurrence_by_month` | string | Comma-separated months (1–12): `"1,6"` |
| `exdates` | string | Comma-separated RFC 3339 timestamps of excluded recurrence instances |
| `rdates` | string | Comma-separated RFC 3339 timestamps of additional recurrence dates |
| `reminder_minutes` | int | Minutes before event to remind (0–40320) |
| `location` | string | Location text (max 500 chars) |
| `latitude` | float | Location latitude (-90 to 90) |
| `longitude` | float | Location longitude (-180 to 180) |
| `created_at` | string | Creation timestamp (read-only) |
| `updated_at` | string | Last update timestamp (read-only) |

For `PUT` updates, all fields are optional — only included fields are changed.

### Import

The import endpoints accept either iCalendar content directly or a URL to fetch it from:

```json
{"ics_content": "BEGIN:VCALENDAR..."}
```

or

```json
{"url": "https://example.com/calendar.ics"}
```

### iCalendar Feed

Subscribe to your calendar from any app that supports iCalendar (Google Calendar, Apple Calendar, Thunderbird, etc.) using:

```
http://localhost:8080/calendar.ics
```

### Example

```bash
# Create an event
curl -X POST http://localhost:8080/api/v1/events \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "Team Meeting",
    "start_time": "2026-02-17T14:00:00Z",
    "end_time": "2026-02-17T15:00:00Z",
    "color": "#4285f4"
  }'

# Create an all-day event
curl -X POST http://localhost:8080/api/v1/events \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "Holiday",
    "start_time": "2026-06-06",
    "end_time": "2026-06-07",
    "all_day": true,
    "color": "#0b8043"
  }'

# List events for February 2026
curl 'http://localhost:8080/api/v1/events?from=2026-02-01T00:00:00Z&to=2026-03-01T00:00:00Z'

# Search events
curl 'http://localhost:8080/api/v1/events?q=meeting'
```

## Tech Stack

- **Backend:** Go with `net/http` (Go 1.22+ routing)
- **Database:** SQLite via [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) (pure Go, no CGO)
- **Frontend:** [Preact](https://preactjs.com/) + [HTM](https://github.com/developit/htm) loaded from CDN

## License

Copyright 2026 Mikael Ståldal.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
