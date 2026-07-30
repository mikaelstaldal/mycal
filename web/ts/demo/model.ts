// Mirrors internal/model — the row shapes and the small helpers around them.
//
// A classic worker script: no imports, no exports. demo-sw.ts pulls this and
// its siblings into one scope with importScripts, so everything declared here
// is visible to the other demo/*.ts files (and to nothing else — the app is
// compiled as ES modules by web/ts/tsconfig.json, which excludes this
// directory).
//
// Field names are the SQLite column names of internal/repository/db.go rather
// than the JSON names of openapi.yaml: this is the storage layer, and keeping
// the column names makes the port line up with the Go code it emulates.

/** A row of the events table (internal/model.Event, minus the transient fields). */
interface DemoEvent {
    id: number;
    title: string;
    description: string;
    start_time: string;
    end_time: string;
    all_day: boolean;
    color: string;
    recurrence_freq: string;
    recurrence_count: number;
    recurrence_until: string;
    recurrence_interval: number;
    recurrence_by_day: string;
    recurrence_by_monthday: string;
    recurrence_by_month: string;
    exdates: string;
    rdates: string;
    /** Set only on an override row; null on parents and plain events. */
    recurrence_parent_id: number | null;
    recurrence_original_start: string;
    duration: string;
    categories: string;
    url: string;
    note_slug: string;
    reminder_minutes: number;
    location: string;
    latitude: number | null;
    longitude: number | null;
    calendar_id: number;
    created_at: string;
    updated_at: string;
    /**
     * Non-zero on an instance synthesised by the recurrence expansion, the way
     * model.Event.RecurrenceIndex is. Never stored: it exists so formatEventID
     * can tell an expanded instance from its parent.
     */
    recurrence_index?: number;
}

/** A row of the calendars table (internal/model.Calendar). */
interface DemoCalendar {
    id: number;
    name: string;
    color: string;
}

/** The whole emulated database, stored as one IndexedDB value. */
interface DemoState {
    /** Schema version of this document, so a later format change can migrate. */
    version: number;
    /** Stands in for the events table's AUTOINCREMENT. */
    next_event_id: number;
    /** Stands in for the calendars table's AUTOINCREMENT. */
    next_calendar_id: number;
    events: DemoEvent[];
    calendars: DemoCalendar[];
    preferences: Record<string, string>;
}

/**
 * An error carrying the HTTP status the Go server answers with, so a handler
 * can throw where the Go code returns a typed error (service.ErrValidation →
 * 400, service.ErrNotFound → 404).
 */
class ApiError extends Error {
    readonly status: number;

    constructor(status: number, message: string) {
        super(message);
        this.name = 'ApiError';
        this.status = status;
    }
}

/** Mirrors `fmt.Errorf("%w: …", service.ErrValidation)` — a 400. */
function validationError(message: string): ApiError {
    return new ApiError(400, 'validation error: ' + message);
}

/** Mirrors service.ErrNotFound — a 404 with the wording the Go server uses. */
function notFoundError(): ApiError {
    return new ApiError(404, 'not found');
}

/** The default calendar, inserted by schema v1 with the reserved id 0. */
const DEFAULT_CALENDAR: DemoCalendar = { id: 0, name: 'Default', color: 'dodgerblue' };

/** The colour a calendar created on the fly gets (service.resolveCalendarName). */
const DEFAULT_CALENDAR_COLOR = 'dodgerblue';

// ── Time helpers ─────────────────────────────────────────────────────────────
//
// Every timestamp in the store is RFC 3339 in UTC with second precision, which
// is what Go's time.RFC3339 produces for a UTC time and what SQLite's
// strftime('%Y-%m-%dT%H:%M:%SZ') writes for created_at/updated_at. Keeping to
// that one shape means string comparison orders timestamps correctly, exactly
// as the SQL the queries are ported from relies on.

/** Formats a Date the way Go's time.Time.UTC().Format(time.RFC3339) does. */
function formatRFC3339(t: Date): string {
    return t.toISOString().replace(/\.\d{3}Z$/, 'Z');
}

/**
 * Parses an RFC 3339 timestamp, or returns null when it is not one. Mirrors
 * time.Parse(time.RFC3339, s): a bare date or a datetime without an offset is
 * rejected, which Date's own parser would happily accept.
 */
function parseRFC3339(s: string): Date | null {
    if (!/^\d{4}-\d{2}-\d{2}[Tt]\d{2}:\d{2}:\d{2}(\.\d+)?([Zz]|[+-]\d{2}:\d{2})$/.test(s)) {
        return null;
    }
    const t = new Date(s);
    return isNaN(t.getTime()) ? null : t;
}

/**
 * The RFC 3339 string Go would write for a timestamp it just parsed:
 * time.Parse keeps the offset the input carried, and Format then re-emits it
 * with second precision. So a create request naming "+01:00" is stored with
 * "+01:00", exactly as validate.go stores it, while the update paths — which
 * call .UTC() first — go through formatRFC3339 instead.
 *
 * Returns null when s is not an RFC 3339 timestamp.
 */
function canonicalRFC3339(s: string): string | null {
    const m = /^(\d{4}-\d{2}-\d{2})[Tt](\d{2}:\d{2}:\d{2})(?:\.\d+)?([Zz]|[+-]\d{2}:\d{2})$/.exec(s);
    if (m === null || parseRFC3339(s) === null) return null;
    return m[1] + 'T' + m[2] + (m[3] === 'z' ? 'Z' : m[3]);
}

/** The calendar year of an RFC 3339 timestamp, in its own offset. */
function yearOf(rfc3339: string): number {
    return Number(rfc3339.slice(0, 4));
}

/** The offset an RFC 3339 timestamp carries: "Z" or "+01:00". */
function offsetOf(rfc3339: string): string {
    const m = /([Zz]|[+-]\d{2}:\d{2})$/.exec(rfc3339);
    if (m === null) return 'Z';
    return m[1] === 'z' ? 'Z' : m[1];
}

/**
 * Formats an instant in a given offset, which is what Go does when it derives
 * one time from another: time.Time.Add keeps the location, so an end computed
 * from a start given as "+01:00" is written as "+01:00" too. Same instant
 * either way, but the stored string should match the server's byte for byte.
 */
function formatRFC3339At(t: Date, offset: string): string {
    if (offset === 'Z') return formatRFC3339(t);
    const sign = offset[0] === '-' ? -1 : 1;
    const minutes = sign * (Number(offset.slice(1, 3)) * 60 + Number(offset.slice(4, 6)));
    return new Date(t.getTime() + minutes * 60000).toISOString().slice(0, 19) + offset;
}

/**
 * Parses a `YYYY-MM-DD` date as UTC midnight, the way ogen decodes a
 * `format: date` value and validate.go then normalises it.
 */
function parseDateOnly(s: string): Date | null {
    const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(s);
    if (m === null) return null;
    const t = new Date(Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3])));
    return isNaN(t.getTime()) ? null : t;
}

/** Formats a Date as the `YYYY-MM-DD` an openapi `format: date` field carries. */
function formatDateOnly(t: Date): string {
    return t.toISOString().slice(0, 10);
}

/** UTC midnight of the day t falls on, as time.Date(…, 0,0,0,0, time.UTC) does. */
function truncateToUTCDay(t: Date): Date {
    return new Date(Date.UTC(t.getUTCFullYear(), t.getUTCMonth(), t.getUTCDate()));
}

/** Adds whole days, mirroring Go's AddDate(0, 0, n) for a UTC time. */
function addDays(t: Date, days: number): Date {
    return new Date(t.getTime() + days * 86400000);
}

/** Milliseconds in the units the ported code talks about. */
const MILLIS_PER_DAY = 86400000;

/**
 * Parses an ISO 8601 duration such as PT1H, PT30M, P1D or P1DT2H30M, returning
 * milliseconds. Ported from model.ParseDuration, error wording included.
 */
function parseDuration(s: string): number {
    if (s === '') throw new Error('empty duration');
    s = s.toUpperCase();
    if (!s.startsWith('P')) throw new Error('duration must start with P');
    s = s.slice(1);

    let total = 0;
    let inTime = false;
    let num = '';
    for (const c of s) {
        if (c === 'T') {
            inTime = true;
            continue;
        }
        if (c >= '0' && c <= '9') {
            num += c;
            continue;
        }
        const n = parseInt(num, 10);
        if (num === '' || isNaN(n)) throw new Error('invalid duration number: ' + num);
        num = '';

        if (inTime) {
            switch (c) {
                case 'H':
                    total += n * 3600000;
                    break;
                case 'M':
                    total += n * 60000;
                    break;
                case 'S':
                    total += n * 1000;
                    break;
                default:
                    throw new Error('unknown time unit: ' + c);
            }
        } else {
            switch (c) {
                case 'D':
                    total += n * MILLIS_PER_DAY;
                    break;
                case 'W':
                    total += n * 7 * MILLIS_PER_DAY;
                    break;
                default:
                    throw new Error('unknown date unit: ' + c);
            }
        }
    }

    if (total <= 0) throw new Error('duration must be positive');
    return total;
}

// ── Composite event ids ──────────────────────────────────────────────────────

/** A parsed event id: the row id, plus the instance start of a recurrence instance. */
interface ParsedEventID {
    dbID: number;
    instanceStart: string;
}

/**
 * Parses a composite id, mirroring model.ParseEventID: "12" is a row, and
 * "12_2025-03-04T09:00:00Z" is one instance of recurring event 12. Returns null
 * when the numeric part does not parse, which the handler answers with a 400.
 */
function parseEventID(s: string): ParsedEventID | null {
    const idx = s.indexOf('_');
    const idPart = idx < 0 ? s : s.slice(0, idx);
    if (!/^-?\d+$/.test(idPart)) return null;
    const dbID = Number(idPart);
    if (!Number.isSafeInteger(dbID)) return null;
    return { dbID, instanceStart: idx < 0 ? '' : s.slice(idx + 1) };
}

/**
 * The public id of a row or instance, mirroring model.Event.SetStringID: an
 * override reports itself under the parent and its original start, an expanded
 * instance under the parent and its own start, everything else under its row id.
 */
function eventStringID(e: DemoEvent): { id: string; parentID: string } {
    if (e.recurrence_parent_id !== null) {
        const parent = String(e.recurrence_parent_id);
        return { id: parent + '_' + e.recurrence_original_start, parentID: parent };
    }
    if (e.recurrence_freq !== '' && (e.recurrence_index ?? 0) > 0) {
        const parent = String(e.id);
        return { id: parent + '_' + e.start_time, parentID: parent };
    }
    return { id: String(e.id), parentID: '' };
}

/** True when the row is a recurring parent (model.Event.IsRecurring). */
function isRecurring(e: DemoEvent): boolean {
    return e.recurrence_freq !== '';
}

/**
 * Orders events the way every listing query does: `ORDER BY e.start_time,
 * e.created_at`. Both are fixed-width UTC strings, so SQLite's byte comparison
 * and JavaScript's string comparison agree. The row id breaks the remaining
 * ties, which SQLite leaves unspecified but a demo may as well keep stable.
 */
function compareEvents(a: DemoEvent, b: DemoEvent): number {
    if (a.start_time !== b.start_time) return a.start_time < b.start_time ? -1 : 1;
    if (a.created_at !== b.created_at) return a.created_at < b.created_at ? -1 : 1;
    return a.id - b.id;
}

/** A blank row with every column at its schema default (see schemaV1). */
function newEvent(): DemoEvent {
    return {
        id: 0,
        title: '',
        description: '',
        start_time: '',
        end_time: '',
        all_day: false,
        color: '',
        recurrence_freq: '',
        recurrence_count: 0,
        recurrence_until: '',
        recurrence_interval: 0,
        recurrence_by_day: '',
        recurrence_by_monthday: '',
        recurrence_by_month: '',
        exdates: '',
        rdates: '',
        recurrence_parent_id: null,
        recurrence_original_start: '',
        duration: '',
        categories: '',
        url: '',
        note_slug: '',
        reminder_minutes: 0,
        location: '',
        latitude: null,
        longitude: null,
        calendar_id: 0,
        created_at: '',
        updated_at: '',
    };
}
