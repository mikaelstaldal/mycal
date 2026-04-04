# API

[See OpenAPI specification](../openapi.yaml). 

All endpoints are under `/api/v1`. Datetimes use RFC 3339 format (or `YYYY-MM-DD` for all-day events). Errors return `{"error": "message"}`.

| Method | Path                             | Description                                                                 |
|--------|----------------------------------|-----------------------------------------------------------------------------|
| GET    | `/api/v1/preferences`            | Get all preferences                                                         |
| PATCH  | `/api/v1/preferences`            | Update preferences (partial)                                                |
| GET    | `/api/v1/calendars`              | List all calendars                                                          |
| PATCH  | `/api/v1/calendars/{id}`         | Update a calendar (name and/or color)                                       |
| GET    | `/api/v1/events?from=...&to=...` | List events in a time range (optional `calendar_id` or `calendar` filter)   |
| GET    | `/api/v1/events?q=...`           | Search events by text (optional `calendar_id` or `calendar` filter)         |
| POST   | `/api/v1/events`                 | Create an event                                                             |
| GET    | `/api/v1/events/{id}`            | Get a single event (id may be composite, URL-encoded)                       |
| PATCH  | `/api/v1/events/{id}`            | Update an event (use composite ID to override a single recurrence instance) |
| DELETE | `/api/v1/events/{id}`            | Delete an event (use composite ID to exclude a single recurrence instance)  |
| POST   | `/api/v1/import`                 | Import events from iCalendar data (optional `calendar` query param)         |
| POST   | `/api/v1/import-single`          | Import a single event from iCalendar data (optional `calendar` query param) |
| GET    | `/api/v1/events.ics`             | iCalendar feed (optional `calendar` filter)                                 |
| GET    | `/api/v1/feeds`                  | List all feed subscriptions                                                 |
| POST   | `/api/v1/feeds`                  | Create a feed subscription                                                  |
| GET    | `/api/v1/feeds/{id}`             | Get a feed subscription                                                     |
| PUT    | `/api/v1/feeds/{id}`             | Update a feed subscription                                                  |
| DELETE | `/api/v1/feeds/{id}`             | Delete a feed subscription                                                  |
| POST   | `/api/v1/feeds/{id}/refresh`     | Manually refresh a feed                                                     |
| GET    | `/calendar.ics`                  | iCalendar feed (convenience URL)                                            |

## Preferences

`GET /api/v1/preferences` returns all preferences with their current values (defaults filled in for unset keys):

```json
{}
```

`PATCH /api/v1/preferences` accepts a JSON object with a subset of preference keys to update. Only included keys are changed; omitted keys are left unchanged. Unknown keys return 400. Returns the full preferences state after update.

There are currently no user-configurable server-side preferences. The default event color is now managed per-calendar via the calendars table.

## Calendars

`GET /api/v1/calendars` returns all calendars:

```json
[{"id": 0, "name": "Default", "color": "dodgerblue"}]
```

Calendars are auto-created when importing events or creating feed subscriptions with a `calendar_name`. The default calendar (id=0) always exists.

`PATCH /api/v1/calendars/{id}` updates a calendar's name and/or color. Only included fields are changed:

```json
{"name": "Work", "color": "gold"}
```

Returns the updated calendar object.

## Event Fields

| Field                       | Type   | Description                                                                      |
|-----------------------------|--------|----------------------------------------------------------------------------------|
| `id`                        | string | Unique event ID (read-only, opaque). Must be URL-encoded in paths.               |
| `parent_id`                 | string | Parent event ID for recurrence instances (read-only, absent for non-instances)   |
| `title`                     | string | Event title (required, max 500 chars)                                            |
| `description`               | string | Event description (max 10000 chars)                                              |
| `start_time`                | string | Start time in RFC 3339 or `YYYY-MM-DD` for all-day (required)                    |
| `end_time`                  | string | End time in RFC 3339 or `YYYY-MM-DD` for all-day (required, must be after start) |
| `all_day`                   | bool   | Whether this is an all-day event                                                 |
| `color`                     | string | CSS3 color name per RFC 7986 (e.g. `dodgerblue`, `red`, `gold`)                  |
| `recurrence_freq`           | string | `""`, `"DAILY"`, `"WEEKLY"`, `"MONTHLY"`, or `"YEARLY"`                          |
| `recurrence_count`          | int    | Number of recurrences (0–1000)                                                   |
| `recurrence_until`          | string | Recurrence end date in RFC 3339                                                  |
| `recurrence_interval`       | int    | Repeat every N periods (0 or 1 = every, 2 = every other, etc.)                   |
| `recurrence_by_day`         | string | Comma-separated days: `"MO,WE,FR"` or ordinal `"2MO,-1FR"`                       |
| `recurrence_by_monthday`    | string | Comma-separated month days: `"15,30"` or negative `"-1"`                         |
| `recurrence_by_month`       | string | Comma-separated months (1–12): `"1,6"`                                           |
| `exdates`                   | string | Comma-separated RFC 3339 timestamps of excluded recurrence instances             |
| `rdates`                    | string | Comma-separated RFC 3339 timestamps of additional recurrence dates               |
| `recurrence_parent_id`      | int    | Parent event DB ID for instance overrides (read-only)                            |
| `recurrence_original_start` | string | Original start time of overridden instance (read-only)                           |
| `duration`                  | string | ISO 8601 duration (e.g. `PT1H`, `PT30M`, `P1D`) — alternative to `end_time`      |
| `categories`                | string | Comma-separated category tags (max 500 chars)                                    |
| `url`                       | string | Reference URL (must start with `http://` or `https://`, max 2000 chars)          |
| `reminder_minutes`          | int    | Minutes before event to remind (0–40320)                                         |
| `location`                  | string | Location text (max 500 chars)                                                    |
| `latitude`                  | float  | Location latitude (-90 to 90)                                                    |
| `longitude`                 | float  | Location longitude (-180 to 180)                                                 |
| `calendar_id`               | int    | Calendar ID (read-only, set via import)                                          |
| `calendar_name`             | string | Calendar name (read-only, derived from calendar_id)                              |
| `created_at`                | string | Creation timestamp (read-only)                                                   |
| `updated_at`                | string | Last update timestamp (read-only)                                                |

For `PATCH` updates, all fields are optional — only included fields are changed.

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

## Calendar Filtering

Events can be assigned to a calendar via the `calendar` query parameter on import endpoints. The `calendar_id` and `calendar_name` fields are read-only through the regular create/update API.

**Filtering by calendar** — pass one or more `calendar_id` (integer) or `calendar` (name, for backward compatibility) query parameters to list, search, and iCal feed endpoints:

```bash
# List events from a specific calendar by ID
curl 'http://localhost:8080/api/v1/events?from=2026-02-01T00:00:00Z&to=2026-03-01T00:00:00Z&calendar_id=1'

# List events from multiple calendars
curl 'http://localhost:8080/api/v1/events?from=2026-02-01T00:00:00Z&to=2026-03-01T00:00:00Z&calendar_id=1&calendar_id=2'

# Filter by calendar name (backward compatible)
curl 'http://localhost:8080/api/v1/events?from=2026-02-01T00:00:00Z&to=2026-03-01T00:00:00Z&calendar=work'

# Import events into a specific calendar
curl -X POST 'http://localhost:8080/api/v1/import?calendar=work' \
  -H 'Content-Type: text/calendar' \
  --data-binary @events.ics

# Filter iCal feed by calendar
curl 'http://localhost:8080/api/v1/events.ics?calendar_id=1'
```

When the `calendar_id`/`calendar` parameter is omitted, all events are returned regardless of calendar.

## Feed Subscriptions

Feed subscriptions periodically re-import events from an ICS URL with automatic deduplication (events with the same ICS UID are skipped).

### Feed Fields

| Field                      | Type   | Description                                                  |
|----------------------------|--------|--------------------------------------------------------------|
| `id`                       | string | Unique feed ID (read-only)                                   |
| `url`                      | string | URL to fetch iCalendar data from (required, max 2000 chars)  |
| `calendar_name`            | string | Calendar name (read-only, derived from calendar_id)          |
| `calendar_color`           | string | CSS color for the new calendar (only when creating a feed)   |
| `refresh_interval_minutes` | int    | How often to refresh (5–10080, default 60)                   |
| `last_refreshed_at`        | string | Last refresh timestamp (read-only)                           |
| `last_error`               | string | Last refresh error message (read-only, empty if OK)          |
| `enabled`                  | bool   | Whether automatic refresh is enabled (default true)          |
| `created_at`               | string | Creation timestamp (read-only)                               |
| `updated_at`               | string | Last update timestamp (read-only)                            |

### Create a feed

```bash
curl -X POST http://localhost:8080/api/v1/feeds \
  -H 'Content-Type: application/json' \
  -d '{
    "url": "https://example.com/calendar.ics",
    "calendar_name": "work",
    "refresh_interval_minutes": 60
  }'
```

### Manually refresh a feed

```bash
curl -X POST http://localhost:8080/api/v1/feeds/1/refresh
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
    "color": "dodgerblue"
  }'

# Create an all-day event
curl -X POST http://localhost:8080/api/v1/events \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "Holiday",
    "start_time": "2026-06-06",
    "end_time": "2026-06-07",
    "all_day": true,
    "color": "green"
  }'

# List events for February 2026
curl 'http://localhost:8080/api/v1/events?from=2026-02-01T00:00:00Z&to=2026-03-01T00:00:00Z'

# Search events
curl 'http://localhost:8080/api/v1/events?q=meeting'
```
