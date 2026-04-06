# Known Issues

## Performance Issues

### Network & Assets

#### P16: No HTTP response compression
- **File:** `main.go`
- No gzip/brotli middleware applied. Repeated JSON field names (e.g. `start_time`, `end_time` × 100 events) compress to ~30% of original size.

#### P17: No cache headers on API responses or static assets
- **Files:** `internal/handler/impl.go`, `main.go:145-149`
- API responses have no `Cache-Control`, `ETag`, or `Last-Modified` headers. Static JS/CSS files served via `http.FileServer` with no versioning or cache headers, causing redownloads on every refresh.

#### P18: All 30+ event columns fetched for every query
- **File:** `internal/repository/sqlite.go:159`
- `selectColumnsBase` fetches all columns including `latitude`, `longitude`, `categories`, `url`, `exdates`, `recurrence_by_month`, etc. on every query, even for list views that only display title and time.

#### P19: Leaflet (200KB+) loaded unconditionally
- **File:** `web/ts/components/event-form.tsx`
- The Leaflet map library is imported regardless of whether the user ever sets a location on an event.

---

## Accessibility Issues

### Critical (WCAG 2.1 Level A)

#### A1: Form validation errors not announced to screen readers
- **File:** `web/ts/components/event-form.tsx:425`
- Validation error messages use a `.error` CSS class but lack `role="alert"`, so screen readers do not announce them.

#### A2: Toast notifications not announced
- **File:** `web/ts/components/toast.tsx:18`
- Toast elements have no `role="status"` or `aria-live="polite"`, so they are invisible to screen readers.

#### A3: Search results have no live region
- **File:** `web/ts/app.tsx:442-459`
- Search results appear dynamically but there is no `aria-live` region to announce when results load.

#### A4: Dialog elements missing `aria-labelledby`
- **Files:** `web/ts/components/event-form.tsx:402`, `web/ts/components/import-form.tsx:48`, `web/ts/components/feeds.tsx:86`, `web/ts/components/settings.tsx:88`, `web/ts/lib/confirm.ts:60`
- All native `<dialog>` elements lack `aria-labelledby` pointing to their title headings.

#### A5: Settings API key error uses inline style with no `role="alert"`
- **File:** `web/ts/components/settings.tsx:143-146`
- Validation error uses hardcoded `color: #c62828` with no semantic error role.

---

### High (WCAG 2.1 Level AA)

#### A6: No landmark regions (`<header>`, `<main>`, `<aside>`)
- **Files:** `web/ts/app.tsx:397-509`, `web/static/index.html:39`
- The app root uses generic `<div>` containers throughout. The left sidebar, main content area, and header are not wrapped in semantic landmark elements.

#### A7: Insufficient color contrast on hover states
- **File:** `web/static/style.css:164-167, 969, 1017-1019, 637-644, 769-780`
- Several hover and disabled-text colors fail WCAG AA contrast (4.5:1 for normal text):
  - `.week-number:hover`: `#4285f4` on `#eef` ≈ 2.5:1
  - `.year-month-header:hover`: `#1a73e8` on `#e8f0fe` ≈ 2.8:1
  - `.year-day-other` text `#bbb` on white ≈ 1.8:1
  - `.allday-label` and `.time-gutter`: `var(--text-disabled)` (`#999`) ≈ 2.3:1

#### A8: Keyboard-inaccessible interactive elements
- **Files:** `web/ts/components/mini-month.tsx:37`, `web/ts/components/year-view.tsx:54, 64`
- Month title and year-view week numbers have `onClick` handlers but no `role="button"`, `tabIndex`, or `onKeyDown` — unreachable via keyboard.

#### A9: Heading hierarchy — dialogs use `<h2>`/`<h3>` instead of `<h1>`
- **Files:** `web/ts/components/event-form.tsx:405`, `web/ts/components/feeds.tsx:87`, `web/ts/components/settings.tsx:88`, `web/ts/lib/confirm.ts`
- Dialog titles should be `<h1>` since they are the primary heading within their modal context.

#### A10: Focus not set on dialog open for edit mode
- **File:** `web/ts/components/event-form.tsx:167-175`
- Focus is moved to the title input only for new events (line 172). When editing an existing event, focus is not explicitly managed.

#### A11: `mini-month.tsx` navigation buttons use `title` only, not `aria-label`
- **File:** `web/ts/components/mini-month.tsx:36-40`
- Previous/next month buttons rely on `title` attributes which are not reliably read by all screen readers; `aria-label` should be used instead.

---

### Medium (Best Practice & Usability)

#### A12: No `aria-required` or visible required indicator on form fields
- **File:** `web/ts/components/event-form.tsx:428-432`
- The Title field is required but has no `aria-required="true"` on the input and no visible `*` indicator for sighted users.

#### A13: Color picker buttons lack accessible color names
- **File:** `web/ts/components/event-form.tsx:590-607`
- Color swatch buttons use `aria-pressed` correctly but have no text alternative — screen readers announce only "pressed" with no color name.

#### A14: Week-view all-day expand toggle uses bare Unicode arrows
- **File:** `web/ts/components/week-view.tsx:119-121`
- The `▲`/`▼` expand toggle has no `aria-label`, so its purpose is not announced to screen readers.

#### A15: Import hint text not associated with file input
- **File:** `web/ts/components/import-form.tsx:77`
- The helper text "The file or URL must contain exactly one event" is not linked to the file input via `aria-describedby`.

#### A16: Confirm dialog built with `innerHTML` — poor screen reader support
- **File:** `web/ts/lib/confirm.ts:39-64`
- The confirm dialog constructs its DOM via `innerHTML`, making it harder for assistive technology to process. It also lacks `aria-labelledby` and `aria-describedby`.

#### A17: Calendar sidebar checkboxes use `readOnly` instead of controlled inputs
- **File:** `web/ts/components/calendar-sidebar.tsx:49-51, 80-81`
- "Select all" toggle is semantically a button but rendered as a `readOnly` checkbox. This is confusing for screen reader users.

#### A18: Loading states have no accessible announcements
- **Files:** `web/ts/components/import-form.tsx:25-45`, `web/ts/components/feeds.tsx:29-39`
- Spinner/loading text appears visually but is not announced via `aria-live` regions.
