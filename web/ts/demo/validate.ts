// Mirrors internal/service/validate.go, plus the model-level validators it
// calls (model.ValidateColor, ValidateURL, ValidateNoteSlug) and the parts of
// openapi.yaml that ogen would have enforced before the handler ran.
//
// Function names and error wording follow the Go original: the demo is only
// worth having if a request the server rejects is rejected here too, with the
// same message, since that message is what the UI shows.
//
// The Go code works on ogen's decoded structs, where a field carries a Set
// flag; here the request is the parsed JSON object, so "set" means the key is
// present with a value other than undefined. JSON null maps to ogen's Null,
// which validate.go treats as "no value" for the nullable coordinates.

const MAX_TITLE_LENGTH = 500;
const MAX_DESCRIPTION_LENGTH = 10000;
const MAX_LOCATION_LENGTH = 500;
const MAX_CATEGORIES_LENGTH = 500;
const MAX_REMINDER_MINUTES = 40320; // 4 weeks
const MAX_RECURRENCE_COUNT = 1000;
const MAX_RECURRENCE_INTERVAL = 999;
const MAX_RECURRENCE_LIST_LEN = 5000;
const MAX_EVENT_DURATION_MS = 366 * MILLIS_PER_DAY;
const MIN_YEAR = 1970;
const MAX_YEAR_OFFSET = 100;
const MAX_URL_LENGTH = 2000;
const MAX_NOTE_SLUG_LENGTH = 100;
const MAX_CALENDAR_NAME_LENGTH = 100;
/** handler.maxSearchQueryLength. */
const MAX_SEARCH_QUERY_LENGTH = 200;

const VALID_FREQS = new Set(['', 'DAILY', 'WEEKLY', 'MONTHLY', 'YEARLY']);
const VALID_WEEKDAYS = new Set(['SU', 'MO', 'TU', 'WE', 'TH', 'FR', 'SA']);

/** A decoded JSON request body. */
type RequestBody = Record<string, unknown>;

// ── Field accessors ──────────────────────────────────────────────────────────
//
// A wrong type is what ogen would have rejected while decoding, which the
// server answers with a 400; these throw the same status rather than letting a
// number reach a string comparison.

/** True when the field is present at all (ogen's Opt.Set). */
function isSet(req: RequestBody, key: string): boolean {
    return Object.prototype.hasOwnProperty.call(req, key) && req[key] !== undefined;
}

/** True when the field is present and JSON null (ogen's OptNil.Null). */
function isNull(req: RequestBody, key: string): boolean {
    return isSet(req, key) && req[key] === null;
}

function stringField(req: RequestBody, key: string): string {
    const v = req[key];
    if (typeof v !== 'string') throw new ApiError(400, 'invalid value for field ' + key + ': expected a string');
    return v;
}

function numberField(req: RequestBody, key: string): number {
    const v = req[key];
    if (typeof v !== 'number' || !isFinite(v)) {
        throw new ApiError(400, 'invalid value for field ' + key + ': expected a number');
    }
    return v;
}

function intField(req: RequestBody, key: string): number {
    const v = numberField(req, key);
    if (!Number.isInteger(v)) throw new ApiError(400, 'invalid value for field ' + key + ': expected an integer');
    return v;
}

function boolField(req: RequestBody, key: string): boolean {
    const v = req[key];
    if (typeof v !== 'boolean') throw new ApiError(400, 'invalid value for field ' + key + ': expected a boolean');
    return v;
}

/** The field as a string, or "" when unset — ogen's Opt[string].Or(""). */
function stringOr(req: RequestBody, key: string, fallback = ''): string {
    return isSet(req, key) ? stringField(req, key) : fallback;
}

/** The field as an int, or the fallback when unset — ogen's OptInt.Or(0). */
function intOr(req: RequestBody, key: string, fallback = 0): number {
    return isSet(req, key) ? intField(req, key) : fallback;
}

/** A nullable coordinate: the number, or null when unset or JSON null. */
function coordinateField(req: RequestBody, key: string): number | null {
    if (!isSet(req, key) || isNull(req, key)) return null;
    return numberField(req, key);
}

/**
 * The byte length of a string, which is what Go's len() counts. It matters for
 * the length limits: a description of 10000 emoji is far past the server's
 * limit even though it has fewer than 10000 UTF-16 units.
 */
const utf8Encoder = new TextEncoder();

function byteLength(s: string): number {
    return utf8Encoder.encode(s).length;
}

/** A date-time field, parsed the way ogen parses `format: date-time`. */
function dateTimeField(req: RequestBody, key: string): { canonical: string; date: Date } {
    const raw = stringField(req, key);
    const canonical = canonicalRFC3339(raw);
    const date = parseRFC3339(raw);
    if (canonical === null || date === null) {
        throw new ApiError(400, 'invalid value for field ' + key + ': expected an RFC 3339 date-time');
    }
    return { canonical, date };
}

/** A date field, parsed the way ogen parses `format: date`. */
function dateField(req: RequestBody, key: string): Date {
    const raw = stringField(req, key);
    const date = parseDateOnly(raw);
    if (date === null) {
        throw new ApiError(400, 'invalid value for field ' + key + ': expected a YYYY-MM-DD date');
    }
    return date;
}

// ── model-level validators ───────────────────────────────────────────────────

/** model.ValidateColor — a CSS colour name, or "" for "use the default". */
function validateColor(color: string): void {
    if (color === '') return;
    if (CSS_COLOR_NAMES.has(color)) return;
    throw validationError('invalid color: "' + color + '" is not a valid CSS color name');
}

/** model.ValidateURL — http(s) only, so a stored URL can never be a script. */
function validateURL(u: string): void {
    if (u === '') return;
    if (byteLength(u) > MAX_URL_LENGTH) {
        throw validationError('url must be at most ' + MAX_URL_LENGTH + ' characters');
    }
    if (!u.startsWith('http://') && !u.startsWith('https://')) {
        throw validationError('url must start with http:// or https://');
    }
}

/** model.ValidateNoteSlug — the MyNotes slug grammar; "" means no note linked. */
function validateNoteSlug(s: string): void {
    if (s === '') return;
    if (byteLength(s) > MAX_NOTE_SLUG_LENGTH) {
        throw validationError('note_slug must be at most ' + MAX_NOTE_SLUG_LENGTH + ' characters');
    }
    if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(s)) {
        throw validationError('note_slug must be a lowercase alphanumeric slug, e.g. my-note');
    }
}

// ── Event requests ───────────────────────────────────────────────────────────

/**
 * ValidateCreateEventRequest — validates a create request and returns the
 * normalised RFC 3339 start and end times.
 */
function validateCreateEventRequest(req: RequestBody): { startTime: string; endTime: string } {
    const title = stringOr(req, 'title');
    if (title === '') throw validationError('title is required');
    if (byteLength(title) > MAX_TITLE_LENGTH) {
        throw validationError('title must be at most ' + MAX_TITLE_LENGTH + ' characters');
    }
    if (isSet(req, 'description') && byteLength(stringField(req, 'description')) > MAX_DESCRIPTION_LENGTH) {
        throw validationError('description must be at most ' + MAX_DESCRIPTION_LENGTH + ' characters');
    }
    if (isSet(req, 'location') && byteLength(stringField(req, 'location')) > MAX_LOCATION_LENGTH) {
        throw validationError('location must be at most ' + MAX_LOCATION_LENGTH + ' characters');
    }
    if (isSet(req, 'categories') && byteLength(stringField(req, 'categories')) > MAX_CATEGORIES_LENGTH) {
        throw validationError('categories must be at most ' + MAX_CATEGORIES_LENGTH + ' characters');
    }
    if (isSet(req, 'url')) validateURL(stringField(req, 'url'));
    if (isSet(req, 'note_slug')) validateNoteSlug(stringField(req, 'note_slug'));
    if (isSet(req, 'color')) validateColor(stringField(req, 'color'));

    const allDay = isSet(req, 'all_day') ? boolField(req, 'all_day') : false;
    let startTime: string;
    let endTime: string;

    if (allDay) {
        if (!isSet(req, 'start_date')) throw validationError('start_date is required for all-day events');
        const startDate = dateField(req, 'start_date');
        validateDateRange(startDate.getUTCFullYear());

        let end: Date;
        const duration = stringOr(req, 'duration');
        if (duration !== '') {
            if (isSet(req, 'end_date')) throw validationError('cannot specify both duration and end_date');
            end = new Date(startDate.getTime() + parseDurationOrFail(duration));
        } else if (!isSet(req, 'end_date')) {
            end = addDays(startDate, 1);
        } else {
            const endDate = dateField(req, 'end_date');
            if (endDate.getTime() < startDate.getTime()) {
                throw validationError('end_date must not be before start_date');
            } else if (endDate.getTime() === startDate.getTime()) {
                end = addDays(startDate, 1);
            } else {
                validateDateRange(endDate.getUTCFullYear());
                if (endDate.getTime() - startDate.getTime() > MAX_EVENT_DURATION_MS) {
                    throw validationError('event duration must not exceed 366 days');
                }
                end = endDate;
            }
        }

        startTime = formatRFC3339(startDate);
        endTime = formatRFC3339(end);
    } else {
        if (!isSet(req, 'start_time')) throw validationError('start_time is required');
        const start = dateTimeField(req, 'start_time');
        validateDateRange(yearOf(start.canonical));

        const duration = stringOr(req, 'duration');
        if (duration !== '') {
            if (isSet(req, 'end_time')) throw validationError('cannot specify both duration and end_time');
            endTime = formatRFC3339At(
                new Date(start.date.getTime() + parseDurationOrFail(duration)),
                offsetOf(start.canonical),
            );
        } else if (!isSet(req, 'end_time')) {
            throw validationError('end_time is required');
        } else {
            const end = dateTimeField(req, 'end_time');
            validateDateRange(yearOf(end.canonical));
            if (end.date.getTime() <= start.date.getTime()) {
                throw validationError('end_time must be after start_time');
            }
            if (end.date.getTime() - start.date.getTime() > MAX_EVENT_DURATION_MS) {
                throw validationError('event duration must not exceed 366 days');
            }
            endTime = end.canonical;
        }
        startTime = start.canonical;
    }

    const freq = stringOr(req, 'recurrence_freq');
    const count = intOr(req, 'recurrence_count');
    const until = stringOr(req, 'recurrence_until');
    const interval = intOr(req, 'recurrence_interval');
    const byDay = stringOr(req, 'recurrence_by_day');
    const byMonthDay = stringOr(req, 'recurrence_by_monthday');
    const byMonth = stringOr(req, 'recurrence_by_month');
    const exDates = stringOr(req, 'exdates');
    const rDates = stringOr(req, 'rdates');

    if (!VALID_FREQS.has(freq)) {
        throw validationError('recurrence_freq must be one of: DAILY, WEEKLY, MONTHLY, YEARLY');
    }
    if (count < 0) throw validationError('recurrence_count must be >= 0');
    if (count > MAX_RECURRENCE_COUNT) {
        throw validationError('recurrence_count must be at most ' + MAX_RECURRENCE_COUNT);
    }
    if (until !== '' && canonicalRFC3339(until) === null) {
        throw validationError('recurrence_until must be RFC 3339 format');
    }
    validateRecurrenceFields(freq, count, until, interval, byDay, byMonthDay, byMonth, exDates, rDates);

    if (isSet(req, 'reminder_minutes')) {
        const minutes = intField(req, 'reminder_minutes');
        if (minutes < 0) throw validationError('reminder_minutes must be >= 0');
        if (minutes > MAX_REMINDER_MINUTES) {
            throw validationError('reminder_minutes must be at most ' + MAX_REMINDER_MINUTES);
        }
    }

    validateCoordinates(coordinateField(req, 'latitude'), coordinateField(req, 'longitude'));

    return { startTime, endTime };
}

/** ValidateUpdateEventRequest — every field is optional, so each is checked alone. */
function validateUpdateEventRequest(req: RequestBody): void {
    if (isSet(req, 'title')) {
        const title = stringField(req, 'title');
        if (title === '') throw validationError('title cannot be empty');
        if (byteLength(title) > MAX_TITLE_LENGTH) {
            throw validationError('title must be at most ' + MAX_TITLE_LENGTH + ' characters');
        }
    }
    if (isSet(req, 'description') && byteLength(stringField(req, 'description')) > MAX_DESCRIPTION_LENGTH) {
        throw validationError('description must be at most ' + MAX_DESCRIPTION_LENGTH + ' characters');
    }
    if (isSet(req, 'location') && byteLength(stringField(req, 'location')) > MAX_LOCATION_LENGTH) {
        throw validationError('location must be at most ' + MAX_LOCATION_LENGTH + ' characters');
    }
    if (isSet(req, 'categories') && byteLength(stringField(req, 'categories')) > MAX_CATEGORIES_LENGTH) {
        throw validationError('categories must be at most ' + MAX_CATEGORIES_LENGTH + ' characters');
    }
    if (isSet(req, 'url')) validateURL(stringField(req, 'url'));
    if (isSet(req, 'note_slug')) validateNoteSlug(stringField(req, 'note_slug'));
    if (isSet(req, 'color')) validateColor(stringField(req, 'color'));
    if (isSet(req, 'duration')) {
        const duration = stringField(req, 'duration');
        if (duration !== '') parseDurationOrFail(duration);
    }

    if (isSet(req, 'start_time') && isSet(req, 'end_time')) {
        const start = dateTimeField(req, 'start_time').date;
        const end = dateTimeField(req, 'end_time').date;
        if (end.getTime() <= start.getTime()) throw validationError('end_time must be after start_time');
        if (end.getTime() - start.getTime() > MAX_EVENT_DURATION_MS) {
            throw validationError('event duration must not exceed 366 days');
        }
    }

    if (isSet(req, 'recurrence_freq') && !VALID_FREQS.has(stringField(req, 'recurrence_freq'))) {
        throw validationError('recurrence_freq must be one of: DAILY, WEEKLY, MONTHLY, YEARLY');
    }
    if (isSet(req, 'recurrence_count')) {
        const count = intField(req, 'recurrence_count');
        if (count < 0) throw validationError('recurrence_count must be >= 0');
        if (count > MAX_RECURRENCE_COUNT) {
            throw validationError('recurrence_count must be at most ' + MAX_RECURRENCE_COUNT);
        }
    }
    if (isSet(req, 'recurrence_until')) {
        const until = stringField(req, 'recurrence_until');
        if (until !== '' && canonicalRFC3339(until) === null) {
            throw validationError('recurrence_until must be RFC 3339 format');
        }
    }
    if (isSet(req, 'recurrence_interval')) {
        const interval = intField(req, 'recurrence_interval');
        if (interval < 0) throw validationError('recurrence_interval must be >= 0');
        if (interval > MAX_RECURRENCE_INTERVAL) {
            throw validationError('recurrence_interval must be at most ' + MAX_RECURRENCE_INTERVAL);
        }
    }
    if (isSet(req, 'recurrence_by_day')) validateByDay(stringField(req, 'recurrence_by_day'));
    if (isSet(req, 'recurrence_by_monthday')) validateByMonthDay(stringField(req, 'recurrence_by_monthday'));
    if (isSet(req, 'recurrence_by_month')) validateByMonth(stringField(req, 'recurrence_by_month'));
    if (isSet(req, 'exdates')) validateDateList(stringField(req, 'exdates'), 'exdates');
    if (isSet(req, 'rdates')) validateDateList(stringField(req, 'rdates'), 'rdates');
    if (isSet(req, 'reminder_minutes')) {
        const minutes = intField(req, 'reminder_minutes');
        if (minutes < 0) throw validationError('reminder_minutes must be >= 0');
        if (minutes > MAX_REMINDER_MINUTES) {
            throw validationError('reminder_minutes must be at most ' + MAX_REMINDER_MINUTES);
        }
    }

    validateCoordinates(coordinateField(req, 'latitude'), coordinateField(req, 'longitude'));
}

// ── private validation helpers ───────────────────────────────────────────────

/** model.ParseDuration wrapped in validate.go's error wording. */
function parseDurationOrFail(s: string): number {
    try {
        return parseDuration(s);
    } catch (err) {
        throw validationError('invalid duration: ' + (err as Error).message);
    }
}

function validateByDay(s: string): void {
    if (s === '') return;
    if (byteLength(s) > MAX_RECURRENCE_LIST_LEN) {
        throw validationError('recurrence_by_day must be at most ' + MAX_RECURRENCE_LIST_LEN + ' characters');
    }
    for (const raw of s.split(',')) {
        const part = raw.trim();
        if (part.length < 2) throw validationError('recurrence_by_day contains invalid entry: "' + part + '"');
        const dayAbbr = part.slice(-2);
        if (!VALID_WEEKDAYS.has(dayAbbr)) {
            throw validationError('recurrence_by_day contains invalid weekday: "' + dayAbbr + '"');
        }
        if (part.length > 2) {
            const offsetStr = part.slice(0, -2);
            if (!/^[+-]?\d+$/.test(offsetStr)) {
                throw validationError('recurrence_by_day contains invalid offset: "' + offsetStr + '"');
            }
            const offset = parseInt(offsetStr, 10);
            if (offset === 0 || offset < -53 || offset > 53) {
                throw validationError('recurrence_by_day offset must be between -53 and 53, not zero');
            }
        }
    }
}

function validateByMonthDay(s: string): void {
    if (s === '') return;
    if (byteLength(s) > MAX_RECURRENCE_LIST_LEN) {
        throw validationError('recurrence_by_monthday must be at most ' + MAX_RECURRENCE_LIST_LEN + ' characters');
    }
    for (const raw of s.split(',')) {
        const part = raw.trim();
        if (!/^[+-]?\d+$/.test(part)) {
            throw validationError('recurrence_by_monthday contains invalid number: "' + part + '"');
        }
        const n = parseInt(part, 10);
        if (n === 0 || n < -31 || n > 31) {
            throw validationError('recurrence_by_monthday values must be between -31 and 31, not zero');
        }
    }
}

function validateByMonth(s: string): void {
    if (s === '') return;
    if (byteLength(s) > MAX_RECURRENCE_LIST_LEN) {
        throw validationError('recurrence_by_month must be at most ' + MAX_RECURRENCE_LIST_LEN + ' characters');
    }
    for (const raw of s.split(',')) {
        const part = raw.trim();
        if (!/^[+-]?\d+$/.test(part)) {
            throw validationError('recurrence_by_month contains invalid number: "' + part + '"');
        }
        const n = parseInt(part, 10);
        if (n < 1 || n > 12) throw validationError('recurrence_by_month values must be between 1 and 12');
    }
}

function validateDateList(s: string, fieldName: string): void {
    if (s === '') return;
    if (byteLength(s) > MAX_RECURRENCE_LIST_LEN) {
        throw validationError(fieldName + ' must be at most ' + MAX_RECURRENCE_LIST_LEN + ' characters');
    }
    for (const raw of s.split(',')) {
        const part = raw.trim();
        if (part === '') continue;
        if (canonicalRFC3339(part) === null) {
            throw validationError(fieldName + ' contains invalid RFC 3339 datetime: "' + part + '"');
        }
    }
}

function validateRecurrenceFields(
    freq: string,
    count: number,
    until: string,
    interval: number,
    byDay: string,
    byMonthDay: string,
    byMonth: string,
    exDates: string,
    rDates: string,
): void {
    if (interval < 0) throw validationError('recurrence_interval must be >= 0');
    if (interval > MAX_RECURRENCE_INTERVAL) {
        throw validationError('recurrence_interval must be at most ' + MAX_RECURRENCE_INTERVAL);
    }
    if (count > 0 && until !== '') {
        throw validationError('recurrence_count and recurrence_until are mutually exclusive');
    }
    if (freq === '') {
        if (count > 0 || until !== '' || interval > 0 || byDay !== '' || byMonthDay !== '' ||
            byMonth !== '' || exDates !== '' || rDates !== '') {
            throw validationError('recurrence fields require recurrence_freq to be set');
        }
        return;
    }
    validateByDay(byDay);
    validateByMonthDay(byMonthDay);
    validateByMonth(byMonth);
    validateDateList(exDates, 'exdates');
    validateDateList(rDates, 'rdates');
}

/**
 * validateDateRange — the year must be plausible. The upper bound moves with
 * the clock, exactly as the Go code's time.Now().Year() does.
 */
function validateDateRange(year: number): void {
    const maxYear = new Date().getUTCFullYear() + MAX_YEAR_OFFSET;
    if (year < MIN_YEAR || year > maxYear) {
        throw validationError('date must be between year ' + MIN_YEAR + ' and ' + maxYear);
    }
}

function validateCoordinates(latitude: number | null, longitude: number | null): void {
    if (latitude !== null && (latitude < -90 || latitude > 90)) {
        throw validationError('latitude must be between -90 and 90');
    }
    if (longitude !== null && (longitude < -180 || longitude > 180)) {
        throw validationError('longitude must be between -180 and 180');
    }
}

/**
 * model.cssColorNames — the CSS Colors Level 4 names, the only values the
 * server accepts for an event or calendar colour.
 */
const CSS_COLOR_NAMES = new Set([
    'aliceblue', 'antiquewhite', 'aqua', 'aquamarine', 'azure',
    'beige', 'bisque', 'black', 'blanchedalmond', 'blue', 'blueviolet', 'brown', 'burlywood',
    'cadetblue', 'chartreuse', 'chocolate', 'coral', 'cornflowerblue', 'cornsilk', 'crimson', 'cyan',
    'darkblue', 'darkcyan', 'darkgoldenrod', 'darkgray', 'darkgreen', 'darkgrey', 'darkkhaki',
    'darkmagenta', 'darkolivegreen', 'darkorange', 'darkorchid', 'darkred', 'darksalmon',
    'darkseagreen', 'darkslateblue', 'darkslategray', 'darkslategrey', 'darkturquoise', 'darkviolet',
    'deeppink', 'deepskyblue', 'dimgray', 'dimgrey', 'dodgerblue',
    'firebrick', 'floralwhite', 'forestgreen', 'fuchsia',
    'gainsboro', 'ghostwhite', 'gold', 'goldenrod', 'gray', 'green', 'greenyellow', 'grey',
    'honeydew', 'hotpink', 'indianred', 'indigo', 'ivory', 'khaki',
    'lavender', 'lavenderblush', 'lawngreen', 'lemonchiffon', 'lightblue', 'lightcoral',
    'lightcyan', 'lightgoldenrodyellow', 'lightgray', 'lightgreen', 'lightgrey', 'lightpink',
    'lightsalmon', 'lightseagreen', 'lightskyblue', 'lightslategray', 'lightslategrey',
    'lightsteelblue', 'lightyellow', 'lime', 'limegreen', 'linen',
    'magenta', 'maroon', 'mediumaquamarine', 'mediumblue', 'mediumorchid', 'mediumpurple',
    'mediumseagreen', 'mediumslateblue', 'mediumspringgreen', 'mediumturquoise', 'mediumvioletred',
    'midnightblue', 'mintcream', 'mistyrose', 'moccasin',
    'navajowhite', 'navy', 'oldlace', 'olive', 'olivedrab', 'orange', 'orangered', 'orchid',
    'palegoldenrod', 'palegreen', 'paleturquoise', 'palevioletred', 'papayawhip', 'peachpuff',
    'peru', 'pink', 'plum', 'powderblue', 'purple',
    'rebeccapurple', 'red', 'rosybrown', 'royalblue',
    'saddlebrown', 'salmon', 'sandybrown', 'seagreen', 'seashell', 'sienna', 'silver', 'skyblue',
    'slateblue', 'slategray', 'slategrey', 'snow', 'springgreen', 'steelblue',
    'tan', 'teal', 'thistle', 'tomato', 'turquoise',
    'violet', 'wheat', 'white', 'whitesmoke', 'yellow', 'yellowgreen',
]);
