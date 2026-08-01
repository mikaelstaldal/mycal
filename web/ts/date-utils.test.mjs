// node:test coverage for web/ts/util/date-utils.ts (exercised via its compiled
// output, web/static/util/date-utils.js). Run via build.sh or directly:
//   node --test web/ts/date-utils.test.mjs
//
// What is under test is the arithmetic the calendar views depend on, not the
// one-line wrappers around the Intl formatters: the month grid, ISO week
// numbers, week starts, and the two date conversions that cross the local /
// UTC boundary. Those conversions are where a wrong answer is invisible — an
// all-day event silently lands a day off — so they are pinned in a timezone
// that is *not* UTC, on both sides of a DST change.
//
// Functions that render through toLocaleDateString / toLocaleTimeString
// (formatMonthYear, formatDate, formatTime, formatWeekRange, formatDayHeading,
// formatHour, getTimezoneAbbr) are deliberately not asserted here: their output
// is the host ICU locale's, so any assertion would pin the test machine's
// locale rather than this code.

// Must precede the first Date construction: Node re-reads process.env.TZ when
// it changes, and every expectation below assumes Europe/Stockholm (UTC+1,
// UTC+2 in summer).
process.env.TZ = 'Europe/Stockholm';

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const {
    startOfMonth,
    endOfMonth,
    addMonths,
    isSameDay,
    isToday,
    getCalendarDays,
    getWeekdays,
    toLocalDatetimeValue,
    fromLocalDatetimeValue,
    toLocalDateValue,
    exclusiveToInclusiveDate,
    inclusiveToExclusiveDate,
    getISOWeekNumber,
    startOfWeek,
    addWeeks,
    getWeekDays,
    eventStartStr,
    eventEndStr,
    isPastEvent,
} = await import(path.resolve(__dirname, '../static/util/date-utils.js'));

// A local calendar date as YYYY-MM-DD, so a failure reports the day that was
// produced rather than an instant in some other zone.
function localDate(d) {
    const pad = (n) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

// ---------------------------------------------------------------------------
// Month boundaries
// ---------------------------------------------------------------------------

test('endOfMonth lands on the last day, including a leap February', () => {
    assert.equal(localDate(endOfMonth(new Date(2024, 1, 10))), '2024-02-29');
    assert.equal(localDate(endOfMonth(new Date(2026, 1, 10))), '2026-02-28');
    assert.equal(localDate(endOfMonth(new Date(2026, 11, 5))), '2026-12-31');
});

test('startOfMonth keeps the month and drops the day', () => {
    assert.equal(localDate(startOfMonth(new Date(2026, 1, 28))), '2026-02-01');
});

test('addMonths rolls over the year in both directions', () => {
    assert.equal(localDate(addMonths(new Date(2026, 10, 15), 3)), '2027-02-01');
    assert.equal(localDate(addMonths(new Date(2026, 1, 15), -3)), '2025-11-01');
});

test('addMonths from a day the target month does not have stays in that month', () => {
    // Jan 31 + 1 month is Feb 1 here (the day is normalised to 1), not the
    // Mar 2/3 that naive day-preserving arithmetic would produce.
    assert.equal(localDate(addMonths(new Date(2026, 0, 31), 1)), '2026-02-01');
});

// ---------------------------------------------------------------------------
// Month grid
// ---------------------------------------------------------------------------

test('getCalendarDays always fills six rows', () => {
    for (let month = 0; month < 12; month++) {
        for (const weekStartDay of [0, 1]) {
            assert.equal(getCalendarDays(2026, month, weekStartDay).length, 42);
        }
    }
});

test('getCalendarDays pads with the previous month up to the week start', () => {
    // Feb 2026 starts on a Sunday, so a Monday-start week needs six leading days.
    const days = getCalendarDays(2026, 1, 1);
    assert.equal(localDate(days[0].date), '2026-01-26');
    assert.deepEqual(days.slice(0, 6).map((d) => d.currentMonth), [false, false, false, false, false, false]);
    assert.equal(localDate(days[6].date), '2026-02-01');
    assert.equal(days[6].currentMonth, true);
});

test('getCalendarDays pads with the next month to the end of the grid', () => {
    const days = getCalendarDays(2026, 1, 1);
    assert.equal(localDate(days[41].date), '2026-03-08');
    assert.equal(days[41].currentMonth, false);
    // 28 days in Feb 2026 + 6 leading = 34, so the last 8 are March.
    assert.equal(days.filter((d) => d.currentMonth).length, 28);
});

test('getCalendarDays needs no leading padding when the month starts on the week start', () => {
    // Feb 2026 starts on a Sunday — with a Sunday-start week the grid opens on Feb 1.
    const days = getCalendarDays(2026, 1, 0);
    assert.equal(localDate(days[0].date), '2026-02-01');
    assert.equal(days[0].currentMonth, true);
    assert.equal(localDate(days[41].date), '2026-03-14');
});

test('getCalendarDays returns consecutive days with no gap or repeat', () => {
    const days = getCalendarDays(2026, 2, 1); // March 2026, spans a DST change
    for (let i = 1; i < days.length; i++) {
        const prev = days[i - 1].date;
        const expected = new Date(prev.getFullYear(), prev.getMonth(), prev.getDate() + 1);
        assert.equal(localDate(days[i].date), localDate(expected));
    }
});

test('getWeekdays rotates the labels to the configured week start', () => {
    assert.deepEqual(getWeekdays(1), ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']);
    assert.deepEqual(getWeekdays(0), ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']);
    assert.deepEqual(getWeekdays(6), ['Sat', 'Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri']);
});

// ---------------------------------------------------------------------------
// Weeks
// ---------------------------------------------------------------------------

test('startOfWeek walks back to the configured week start', () => {
    const saturday = new Date(2026, 7, 1); // 2026-08-01, a Saturday
    assert.equal(localDate(startOfWeek(saturday, 1)), '2026-07-27'); // Monday
    assert.equal(localDate(startOfWeek(saturday, 0)), '2026-07-26'); // Sunday
});

test('startOfWeek on the week start itself is a no-op', () => {
    const monday = new Date(2026, 6, 27);
    assert.equal(localDate(startOfWeek(monday, 1)), '2026-07-27');
});

test('getWeekDays returns seven consecutive days from the week start', () => {
    const days = getWeekDays(new Date(2026, 7, 1), 1);
    assert.deepEqual(days.map(localDate), [
        '2026-07-27', '2026-07-28', '2026-07-29', '2026-07-30',
        '2026-07-31', '2026-08-01', '2026-08-02',
    ]);
});

test('getWeekDays crosses a DST change without dropping or repeating a day', () => {
    // Europe/Stockholm springs forward on 2026-03-29 — a week built by adding
    // 24h at a time would show Mar 29 twice and lose Mar 30.
    const days = getWeekDays(new Date(2026, 2, 25), 1);
    assert.deepEqual(days.map(localDate), [
        '2026-03-23', '2026-03-24', '2026-03-25', '2026-03-26',
        '2026-03-27', '2026-03-28', '2026-03-29',
    ]);
});

test('addWeeks moves whole weeks across a DST change and a month boundary', () => {
    assert.equal(localDate(addWeeks(new Date(2026, 2, 25), 1)), '2026-04-01');
    assert.equal(localDate(addWeeks(new Date(2026, 2, 25), -1)), '2026-03-18');
    assert.equal(localDate(addWeeks(new Date(2026, 0, 7), -2)), '2025-12-24');
});

// ---------------------------------------------------------------------------
// ISO week numbers (RFC 5545 / ISO 8601: week 1 is the one holding Jan 4)
// ---------------------------------------------------------------------------

test('getISOWeekNumber counts a normal year from week 1', () => {
    assert.equal(getISOWeekNumber(new Date(2026, 0, 1)), 1);  // Thu — week 1
    assert.equal(getISOWeekNumber(new Date(2026, 0, 4)), 1);  // Sun — still week 1
    assert.equal(getISOWeekNumber(new Date(2026, 0, 5)), 2);  // Mon — week 2
});

test('getISOWeekNumber assigns late-December days to week 1 of the next year', () => {
    // 2024-12-30 is a Monday whose Thursday falls in 2025, so it is 2025-W01.
    assert.equal(getISOWeekNumber(new Date(2024, 11, 30)), 1);
});

test('getISOWeekNumber assigns early-January days to week 53 of the previous year', () => {
    // 2021-01-01 is a Friday whose Thursday falls in 2020, so it is 2020-W53.
    assert.equal(getISOWeekNumber(new Date(2021, 0, 1)), 53);
    assert.equal(getISOWeekNumber(new Date(2027, 0, 3)), 53); // Sunday of 2026-W53
});

test('getISOWeekNumber gives a 53-week year its final week', () => {
    assert.equal(getISOWeekNumber(new Date(2026, 11, 31)), 53);
});

test('getISOWeekNumber is stable across a day, whatever the time', () => {
    assert.equal(getISOWeekNumber(new Date(2026, 7, 1, 0, 0, 0)),
                 getISOWeekNumber(new Date(2026, 7, 1, 23, 59, 59)));
});

// ---------------------------------------------------------------------------
// Same day / today
// ---------------------------------------------------------------------------

test('isSameDay compares the calendar day, not the instant', () => {
    assert.equal(isSameDay(new Date(2026, 7, 1, 0, 0), new Date(2026, 7, 1, 23, 59)), true);
    assert.equal(isSameDay(new Date(2026, 7, 1, 23, 59), new Date(2026, 7, 2, 0, 0)), false);
    // Same day-of-month in a different month or year is not the same day.
    assert.equal(isSameDay(new Date(2026, 7, 1), new Date(2026, 8, 1)), false);
    assert.equal(isSameDay(new Date(2026, 7, 1), new Date(2025, 7, 1)), false);
});

test('isToday is true for now and false for the same date a year out', () => {
    const now = new Date();
    assert.equal(isToday(now), true);
    assert.equal(isToday(new Date(now.getFullYear() + 1, now.getMonth(), now.getDate())), false);
});

// ---------------------------------------------------------------------------
// Form-field conversions (local wall-clock <-> RFC 3339 instant)
// ---------------------------------------------------------------------------

test('toLocalDatetimeValue renders an instant as local wall-clock time', () => {
    assert.equal(toLocalDatetimeValue('2026-06-15T12:30:00Z'), '2026-06-15T14:30'); // +02:00, summer
    assert.equal(toLocalDatetimeValue('2026-01-15T12:30:00Z'), '2026-01-15T13:30'); // +01:00, winter
});

test('toLocalDatetimeValue zero-pads every field', () => {
    assert.equal(toLocalDatetimeValue('2026-03-04T04:05:00Z'), '2026-03-04T05:05');
});

test('fromLocalDatetimeValue reads the value back as an instant', () => {
    assert.equal(fromLocalDatetimeValue('2026-06-15T14:30'), '2026-06-15T12:30:00.000Z');
    assert.equal(fromLocalDatetimeValue('2026-01-15T13:30'), '2026-01-15T12:30:00.000Z');
});

test('the datetime conversions round-trip', () => {
    for (const iso of ['2026-06-15T12:30:00.000Z', '2026-01-15T12:30:00.000Z', '2026-12-31T23:00:00.000Z']) {
        assert.equal(fromLocalDatetimeValue(toLocalDatetimeValue(iso)), iso);
    }
});

test('toLocalDateValue takes the local day, which can differ from the UTC one', () => {
    // 23:30Z in summer is already 01:30 the next morning locally.
    assert.equal(toLocalDateValue('2026-06-15T23:30:00Z'), '2026-06-16');
    assert.equal(toLocalDateValue('2026-06-15T09:30:00Z'), '2026-06-15');
});

test('the conversions treat an empty value as empty, not as the epoch', () => {
    assert.equal(toLocalDatetimeValue(''), '');
    assert.equal(fromLocalDatetimeValue(''), '');
    assert.equal(toLocalDateValue(''), '');
    assert.equal(exclusiveToInclusiveDate(''), '');
    assert.equal(inclusiveToExclusiveDate(''), '');
});

// ---------------------------------------------------------------------------
// All-day end dates (server stores exclusive, the UI shows inclusive)
// ---------------------------------------------------------------------------

test('exclusiveToInclusiveDate steps back one day across a month and a year', () => {
    assert.equal(exclusiveToInclusiveDate('2026-02-26'), '2026-02-25');
    assert.equal(exclusiveToInclusiveDate('2026-03-01'), '2026-02-28');
    assert.equal(exclusiveToInclusiveDate('2024-03-01'), '2024-02-29'); // leap year
    assert.equal(exclusiveToInclusiveDate('2027-01-01'), '2026-12-31');
});

test('inclusiveToExclusiveDate steps forward one day across a month and a year', () => {
    assert.equal(inclusiveToExclusiveDate('2026-02-25'), '2026-02-26');
    assert.equal(inclusiveToExclusiveDate('2026-02-28'), '2026-03-01');
    assert.equal(inclusiveToExclusiveDate('2024-02-28'), '2024-02-29'); // leap year
    assert.equal(inclusiveToExclusiveDate('2026-12-31'), '2027-01-01');
});

test('the all-day conversions round-trip', () => {
    for (const date of ['2026-01-01', '2026-02-28', '2024-02-29', '2026-12-31']) {
        assert.equal(exclusiveToInclusiveDate(inclusiveToExclusiveDate(date)), date);
    }
});

test('the all-day conversions ignore the local zone', () => {
    // Both work in UTC, so a stored date never shifts by the browser's offset —
    // 00:00Z minus a day must be the previous date, not two days back.
    assert.equal(exclusiveToInclusiveDate('2026-02-26T00:00:00Z'), '2026-02-25');
    assert.equal(inclusiveToExclusiveDate('2026-03-29'), '2026-03-30'); // local DST change day
});

// ---------------------------------------------------------------------------
// Event start/end selection
// ---------------------------------------------------------------------------

test('eventStartStr and eventEndStr pick the fields the event actually uses', () => {
    const timed = {
        all_day: false,
        start_time: '2026-08-01T09:00:00Z',
        end_time: '2026-08-01T10:00:00Z',
        start_date: '2026-08-01',
        end_date: '2026-08-02',
    };
    assert.equal(eventStartStr(timed), '2026-08-01T09:00:00Z');
    assert.equal(eventEndStr(timed), '2026-08-01T10:00:00Z');

    const allDay = { ...timed, all_day: true };
    assert.equal(eventStartStr(allDay), '2026-08-01');
    assert.equal(eventEndStr(allDay), '2026-08-02');
});

test('eventStartStr and eventEndStr fall back to an empty string when the field is absent', () => {
    assert.equal(eventStartStr({ all_day: true }), '');
    assert.equal(eventEndStr({ all_day: false }), '');
});

test('isPastEvent compares the end, not the start', () => {
    const hourMs = 60 * 60 * 1000;
    const at = (offset) => new Date(Date.now() + offset).toISOString();
    // Started an hour ago, ends in an hour — still running, so not past.
    assert.equal(isPastEvent({ all_day: false, start_time: at(-hourMs), end_time: at(hourMs) }), false);
    assert.equal(isPastEvent({ all_day: false, start_time: at(-2 * hourMs), end_time: at(-hourMs) }), true);
});

test('isPastEvent uses the all-day dates for an all-day event', () => {
    const year = new Date().getFullYear();
    assert.equal(isPastEvent({ all_day: true, start_date: '2000-01-01', end_date: '2000-01-02' }), true);
    assert.equal(isPastEvent({ all_day: true, start_date: `${year + 1}-01-01`, end_date: `${year + 1}-01-02` }), false);
});
