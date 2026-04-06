# Known Issues

## Security

### S1 — Authentication is optional
**File:** `main.go:102`  
HTTP Basic Auth is controlled by the `-basicauth` flag and disabled by default. Any deployment without it exposes all events, import, and delete endpoints publicly.  
**Fix:** Require authentication by default, or at minimum warn at startup when running without auth.

### S2 — Incomplete CSRF protection
**File:** `main.go:151`, `go-server-common@v1.1.0/csrf/csrf.go`  
The CSRF middleware only validates `Origin`/`Referer` headers — no synchronizer tokens. This is partial protection. Additionally, the `X-Forwarded-Host` header used to determine server identity can be spoofed if the reverse proxy does not strip it from incoming requests.  
**Fix:** Implement synchronizer token pattern (CSRF token in request header), and document reverse-proxy requirements.

### S5 — No rate limiting
No middleware limits request rate. The API is susceptible to DoS via bulk event creation, repeated search queries, or feed-refresh exhaustion.  
**Fix:** Add a per-IP rate-limiting middleware (e.g., `golang.org/x/time/rate`).

### S6 — Feed error messages leak details
**File:** `internal/service/feed_service.go:177`  
The raw Go error string (which can include URLs, IP addresses, or connection details) is stored and surfaced in the UI as `LastError`.  
**Fix:** Log the full error server-side; return a generic message to the client.

---

## Usability

### U1 — Silent async failures
**File:** `web/ts/app.tsx:71,104,132,287`  
Failures in `loadEvents()`, `loadCalendars()`, and drag operations are only logged to the console. Users see nothing — the calendar silently shows stale or empty data.  
**Fix:** Show a visible error toast or inline message on any failed load.

### U3 — No end-time validation against start time
**File:** `web/ts/components/event-form.tsx:241-303`  
The form allows saving an event whose end time is before its start time. The server rejects it, but the user only discovers this after submission with a raw error.  
**Fix:** Validate `end > start` client-side and show an inline message before form submission.

### U4 — Accessibility: icon buttons missing ARIA labels
**Files:** `web/ts/components/nav.tsx:26-41`, `web/ts/app.tsx:397-421`  
Navigation arrows (◀ ▶) and toolbar buttons (☀, ↻, ⬇, ⇊, 🔗, ⚙) have no `aria-label`. Screen readers announce nothing useful.  
**Fix:** Add `aria-label="Previous"`, `aria-label="Next"`, `aria-label="Refresh"`, etc. to every icon button.

### U5 — Accessibility: calendar grid not keyboard-navigable
**File:** `web/ts/components/calendar.tsx:44-84`  
The calendar uses `<div>` elements without `role`, `tabindex`, or keyboard event handlers. Users who rely on keyboards or screen readers cannot navigate to or create events.  
**Fix:** Add `role="button"` and `tabindex="0"` with `onKeyDown` handlers to day cells; use semantic `<table>` markup.

### U6 — Accessibility: color picker not keyboard-accessible
**File:** `web/ts/components/event-form.tsx:575-590`  
The color picker uses `<div onClick>` elements. Cannot be reached via Tab, and has no `role="radio"` or `aria-checked`.  
**Fix:** Replace with `<button>` elements, add `role="group"` wrapper, and use `aria-pressed`.

### U7 — No semantic landmarks in HTML
**File:** `web/static/index.html`  
The entire app renders into `<div id="app">` with no `<main>`, `<nav>`, or `<header>` elements. Screen readers have no structure to navigate.  
**Fix:** Add landmark elements in the root component render tree.

### U8 — No timezone indicator
**Files:** `web/ts/lib/date-utils.ts:65-172`, `web/ts/components/event-form.tsx`  
Times are shown in the user's local timezone with no label. Users scheduling across timezones or on shared computers have no indication of which timezone is in use.  
**Fix:** Display the local timezone abbreviation (e.g., "2:30 PM CET") next to event times.

### U9 — All-day end date exclusive/inclusive confusion
**File:** `web/ts/lib/date-utils.ts:98-116`  
All-day event end dates are stored as the next day (exclusive) but displayed as the last included day. This silent conversion is confusing and not documented in the UI.  
**Fix:** Add a comment in the form ("End date is inclusive") or verify the UI selection always maps correctly.

### U10 — iCalendar feed URL not discoverable from the UI
**Files:** `web/ts/components/feeds.tsx`, `web/ts/app.tsx:417-419`  
The app serves `/calendar.ics` but there is no place in the UI where a user can copy that URL or see instructions for subscribing in Google Calendar, Apple Calendar, etc.  
**Fix:** Add a "Your calendar feed" section in the Feeds dialog showing the feed URL with a copy button and brief setup instructions.

### U11 — Empty states lack calls to action
**Files:** `web/ts/components/schedule-view.tsx:100-102`, `web/ts/app.tsx:444-445`  
"No upcoming events" and "No events found" messages are plain text with no prompt to create an event or adjust search terms.  
**Fix:** Add a "Create event" button link in the schedule empty state; add a "Clear search" link in the search empty state.

### U12 — No loading indicators for primary data fetch
**File:** `web/ts/app.tsx:77-143`  
The calendar shows nothing (or stale data) while events load. There is no spinner, skeleton, or progress indicator for the initial event fetch or navigation between date ranges.  
**Fix:** Add a loading state flag and render a visible indicator while `listEvents` is in flight.

### U13 — Single mobile breakpoint, no tablet optimisation
**File:** `web/static/style.css:1241-1290`  
Only one media query at `max-width: 600px` exists. Tablets (600–1024 px) get the full desktop layout, which can be cramped. No orientation-change handler is present.  
**Fix:** Add a mid-range breakpoint (e.g., 768 px) adjusting sidebar and button sizing for tablets.
