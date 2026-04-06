# Known Issues

## Security

---

## Usability

### U6 — Accessibility: color picker not keyboard-accessible
**File:** `web/ts/components/event-form.tsx:575-590`  
The color picker uses `<div onClick>` elements. Cannot be reached via Tab, and has no `role="radio"` or `aria-checked`.  
**Fix:** Replace with `<button>` elements, add `role="group"` wrapper, and use `aria-pressed`.

### U7 — No semantic landmarks in HTML
**File:** `web/static/index.html`  
The entire app renders into `<div id="app">` with no `<main>`, `<nav>`, or `<header>` elements. Screen readers have no structure to navigate.  
**Fix:** Add landmark elements in the root component render tree.

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
