# API

All endpoints are under `/api/v1`. Datetimes use RFC 3339 format (or `YYYY-MM-DD` for all-day events). Errors return `{"error": "message"}`.

| Method | Path                             | Description                                                                               |
|--------|----------------------------------|-------------------------------------------------------------------------------------------|
| GET    | `/api/v1/events?from=...&to=...` | List events in a time range                                                               |
| GET    | `/api/v1/events?q=...`           | Search events by text                                                                     |
| POST   | `/api/v1/events`                 | Create an event                                                                           |
| GET    | `/api/v1/events/{id}`            | Get a single event                                                                        |
| PUT    | `/api/v1/events/{id}`            | Update an event (partial)                                                                 |
| PUT    | `/api/v1/events/{id}?instance_start=<RFC3339>` | Override a single recurrence instance                                       |
| DELETE | `/api/v1/events/{id}`            | Delete an event and all its overrides (add `?instance_start=<RFC3339>` to exclude a single recurrence instance) |
| POST   | `/api/v1/import`                 | Import events from iCalendar data                                                         |
| POST   | `/api/v1/import-single`          | Import a single event from iCalendar data                                                 |
| GET    | `/api/v1/events.ics`             | iCalendar feed (all events)                                                               |
| GET    | `/calendar.ics`                  | iCalendar feed (convenience URL)                                                          |

## Event Fields

| Field                    | Type   | Description                                                                      |
|--------------------------|--------|----------------------------------------------------------------------------------|
| `id`                     | int    | Event ID (read-only)                                                             |
| `title`                  | string | Event title (required, max 500 chars)                                            |
| `description`            | string | Event description (max 10000 chars)                                              |
| `start_time`             | string | Start time in RFC 3339 or `YYYY-MM-DD` for all-day (required)                    |
| `end_time`               | string | End time in RFC 3339 or `YYYY-MM-DD` for all-day (required, must be after start) |
| `all_day`                | bool   | Whether this is an all-day event                                                 |
| `color`                  | string | Color hex code                                                                   |
| `recurrence_freq`        | string | `""`, `"DAILY"`, `"WEEKLY"`, `"MONTHLY"`, or `"YEARLY"`                          |
| `recurrence_count`       | int    | Number of recurrences (0–1000)                                                   |
| `recurrence_until`       | string | Recurrence end date in RFC 3339                                                  |
| `recurrence_interval`    | int    | Repeat every N periods (0 or 1 = every, 2 = every other, etc.)                   |
| `recurrence_by_day`      | string | Comma-separated days: `"MO,WE,FR"` or ordinal `"2MO,-1FR"`                       |
| `recurrence_by_monthday` | string | Comma-separated month days: `"15,30"` or negative `"-1"`                         |
| `recurrence_by_month`    | string | Comma-separated months (1–12): `"1,6"`                                           |
| `exdates`                | string | Comma-separated RFC 3339 timestamps of excluded recurrence instances             |
| `rdates`                 | string | Comma-separated RFC 3339 timestamps of additional recurrence dates               |
| `recurrence_parent_id`   | int    | Parent event ID for instance overrides (read-only)                               |
| `recurrence_original_start` | string | Original start time of overridden instance (read-only)                        |
| `duration`               | string | ISO 8601 duration (e.g. `PT1H`, `PT30M`, `P1D`) — alternative to `end_time`     |
| `categories`             | string | Comma-separated category tags (max 500 chars)                                    |
| `url`                    | string | Reference URL (must start with `http://` or `https://`, max 2000 chars)          |
| `reminder_minutes`       | int    | Minutes before event to remind (0–40320)                                         |
| `location`               | string | Location text (max 500 chars)                                                    |
| `latitude`               | float  | Location latitude (-90 to 90)                                                    |
| `longitude`              | float  | Location longitude (-180 to 180)                                                 |
| `created_at`             | string | Creation timestamp (read-only)                                                   |
| `updated_at`             | string | Last update timestamp (read-only)                                                |

For `PUT` updates, all fields are optional — only included fields are changed.

## Import

The import endpoints accept iCalendar data in two ways:

**Direct iCalendar content** — send the raw `.ics` data as the request body with `Content-Type: text/calendar`:

```bash
curl -X POST http://localhost:8080/api/v1/import \
  -H 'Content-Type: text/calendar' \
  --data-binary @events.ics
```

**URL import** — send a JSON body with `Content-Type: application/json`:

```json
{"url": "https://example.com/calendar.ics"}
```

## iCalendar Feed

Subscribe to your calendar from any app that supports iCalendar (Google Calendar, Apple Calendar, Thunderbird, etc.) using:

```
http://localhost:8080/calendar.ics
```

## Example

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
