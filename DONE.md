## Color picker UI improvement

- **Status:** completed
- **Completed:** 2026-03-08 12:18:32

<details>
<summary>Prompt</summary>

In the webUI, make the black circle around the selected color slightly thicker to make it easier to see.

</details>

---

## Week numbers clarification

- **Status:** completed
- **Completed:** 2026-03-08 12:19:52

<details>
<summary>Prompt</summary>

In the web UI, show the week number as "week 11" instead of "W11".

</details>

---

## Bundle web dependencies

- **Status:** completed
- **Completed:** 2026-03-08 12:24:38

<details>
<summary>Prompt</summary>

Download and bundle all external web dependencies (.js and .css) to not depend on the CDN:s at runtime.

</details>

---

## Consistent date format

- **Status:** completed
- **Completed:** 2026-03-08 14:22:10
- **Commit:** e561a47

<details>
<summary>Prompt</summary>

In web UI, when displaying date/time (e.g. event details view), do it in a way which uses the browser's locale settings,
so it will be consistent with editing date/time. Also remove the "Date format" and "Clock format" settings if they are 
not used for anything else.

</details>

<details>
<summary>Report</summary>

Here's a summary of the changes:

**date-utils.js**: Replaced `formatTime()`, `formatDate()`, `formatDateOnly()`, and `formatHour()` to use the browser's `toLocaleTimeString()`/`toLocaleDateString()` instead of custom format strings. Removed the `clockFormat`/`dateFormat` parameters.

**event-form.js**: Updated `formatDatetime()` to use `toLocaleString()` with locale-aware options. Removed `formatDate` import and format parameter usage from `displayStart()`/`displayEnd()`.

**config.js**: Removed `clockFormat` and `dateFormat` from defaults.

**settings.js**: Removed the "Clock format" and "Date format" dropdowns. Updated the "Week view starts at" dropdown to use `formatHour()` (now locale-aware).

**calendar.js, week-view.js, day-view.js, schedule-view.js**: Removed `config.clockFormat` arguments from `formatTime()`/`formatHour()` calls.

</details>

---
