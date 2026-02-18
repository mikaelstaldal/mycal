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
go build -o mycal .
./mycal
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

All endpoints are under `/api/v1`. Datetimes use RFC 3339 format. Errors return `{"error": "message"}`.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/events?from=...&to=...` | List events in a time range |
| POST | `/api/v1/events` | Create an event |
| GET | `/api/v1/events/{id}` | Get a single event |
| PUT | `/api/v1/events/{id}` | Update an event (partial) |
| DELETE | `/api/v1/events/{id}` | Delete an event |
| GET | `/api/v1/events.ics` | iCalendar feed (all events) |
| GET | `/calendar.ics` | iCalendar feed (convenience URL) |

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

# List events for February 2026
curl 'http://localhost:8080/api/v1/events?from=2026-02-01T00:00:00Z&to=2026-03-01T00:00:00Z'
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
