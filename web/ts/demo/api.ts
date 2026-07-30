// Mirrors internal/handler + internal/service — the emulated REST API.
//
// Everything below answers a request the way the Go server answers it: the same
// status codes, the same JSON shapes (openapi.yaml), and the same error
// wording, because that wording is what the UI puts in front of the visitor.
// Handler functions keep the names of the Go methods they stand in for, so a
// change on either side is easy to find on the other.
//
// What is deliberately missing is iCalendar and feeds: a browser cannot fetch a
// third-party .ics (CORS) and internal/ical is not ported, so those routes
// answer 501 and the UI that would reach them is hidden in demo mode.

/** A route handler. `params` holds the decoded path parameters of the match. */
type RouteHandler = (ctx: { request: Request; url: URL; params: string[] }) => Promise<Response>;

interface Route {
    method: string;
    pattern: RegExp;
    handler: RouteHandler;
}

/**
 * Answers an API request, where path is what follows /api/v1 (with its leading
 * slash). Anything unexpected becomes a JSON error rather than a rejected
 * promise, which the browser would turn into an opaque network failure.
 */
async function handleApiRequest(path: string, request: Request): Promise<Response> {
    try {
        const url = new URL(request.url);
        for (const route of ROUTES) {
            const match = route.pattern.exec(path);
            if (match === null) continue;
            if (route.method !== request.method) continue;
            const params = match.slice(1).map(decodePathSegment);
            return await route.handler({ request, url, params });
        }
        return errorResponse(new ApiError(404, 'no such endpoint: ' + path));
    } catch (err) {
        return errorResponse(err);
    }
}

const ROUTES: Route[] = [
    { method: 'GET', pattern: /^\/events$/, handler: apiV1EventsGet },
    { method: 'POST', pattern: /^\/events$/, handler: apiV1EventsPost },
    { method: 'GET', pattern: /^\/events\/([^/]+)$/, handler: apiV1EventsIDGet },
    { method: 'PATCH', pattern: /^\/events\/([^/]+)$/, handler: apiV1EventsIDPatch },
    { method: 'DELETE', pattern: /^\/events\/([^/]+)$/, handler: apiV1EventsIDDelete },
    { method: 'GET', pattern: /^\/calendars$/, handler: apiV1CalendarsGet },
    { method: 'PATCH', pattern: /^\/calendars\/([^/]+)$/, handler: apiV1CalendarsIDPatch },
    { method: 'GET', pattern: /^\/preferences$/, handler: apiV1PreferencesGet },
    { method: 'PATCH', pattern: /^\/preferences$/, handler: apiV1PreferencesPatch },

    // Not emulated. 501 keeps a later port a drop-in, and says plainly that the
    // endpoint exists but this backend does not implement it.
    { method: 'GET', pattern: /^\/events\/([^/]+)\/ics$/, handler: notImplemented('iCalendar export') },
    { method: 'GET', pattern: /^\/events\.ics$/, handler: notImplemented('iCalendar export') },
    { method: 'POST', pattern: /^\/import$/, handler: notImplemented('iCalendar import') },
    { method: 'POST', pattern: /^\/import-single$/, handler: notImplemented('iCalendar import') },
    { method: 'GET', pattern: /^\/feeds$/, handler: notImplemented('feed subscriptions') },
    { method: 'POST', pattern: /^\/feeds$/, handler: notImplemented('feed subscriptions') },
    { method: 'DELETE', pattern: /^\/feeds\/([^/]+)$/, handler: notImplemented('feed subscriptions') },
    { method: 'POST', pattern: /^\/feeds\/([^/]+)\/refresh$/, handler: notImplemented('feed subscriptions') },
];

// ── Responses ────────────────────────────────────────────────────────────────

/**
 * Nothing the demo answers may be cached: the "server" is in this browser, and
 * a cached list would go stale the moment the visitor edits an event.
 */
const NO_STORE = { 'Content-Type': 'application/json', 'Cache-Control': 'no-store' };

function jsonResponse(body: unknown, status = 200): Response {
    return new Response(JSON.stringify(body), { status, headers: NO_STORE });
}

function noContentResponse(): Response {
    return new Response(null, { status: 204, headers: { 'Cache-Control': 'no-store' } });
}

/** The api.Error shape the client parses: `{"error": "…"}`. */
function errorResponse(err: unknown): Response {
    if (err instanceof ApiError) return jsonResponse({ error: err.message }, err.status);
    const message = err instanceof Error ? err.message : String(err);
    return jsonResponse({ error: 'internal server error: ' + message }, 500);
}

function notImplemented(what: string): RouteHandler {
    return () =>
        Promise.resolve(jsonResponse({ error: what + ' is not available in the demo' }, 501));
}

/** A path parameter, which the client percent-encodes (see api/client.ts). */
function decodePathSegment(s: string): string {
    try {
        return decodeURIComponent(s);
    } catch {
        return s;
    }
}

/** The JSON body of a request, which must be an object. */
async function requestBody(request: Request): Promise<RequestBody> {
    let parsed: unknown;
    try {
        parsed = await request.json();
    } catch {
        throw new ApiError(400, 'invalid JSON body');
    }
    if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
        throw new ApiError(400, 'invalid JSON body: expected an object');
    }
    return parsed as RequestBody;
}

// ── Event serialisation ──────────────────────────────────────────────────────

/**
 * modelEventToAPI — the openapi Event for a stored row. Fields that are unset
 * are left out entirely, as ogen leaves out an unset optional, and an all-day
 * event reports dates where a timed event reports timestamps.
 */
function modelEventToAPI(e: DemoEvent, calendarName: string): Record<string, unknown> {
    const ids = eventStringID(e);
    const ae: Record<string, unknown> = { id: ids.id, title: e.title };
    if (e.all_day) {
        ae.all_day = true;
        const start = parseRFC3339(e.start_time);
        const end = parseRFC3339(e.end_time);
        if (start !== null) ae.start_date = formatDateOnly(start);
        if (end !== null) ae.end_date = formatDateOnly(end);
    } else {
        if (parseRFC3339(e.start_time) !== null) ae.start_time = e.start_time;
        if (parseRFC3339(e.end_time) !== null) ae.end_time = e.end_time;
    }
    if (ids.parentID !== '') ae.parent_id = ids.parentID;
    if (e.description !== '') ae.description = e.description;
    if (e.color !== '') ae.color = e.color;
    if (e.recurrence_freq !== '') ae.recurrence_freq = e.recurrence_freq;
    if (e.recurrence_count !== 0) ae.recurrence_count = e.recurrence_count;
    if (canonicalRFC3339(e.recurrence_until) !== null) ae.recurrence_until = e.recurrence_until;
    if (e.recurrence_interval !== 0) ae.recurrence_interval = e.recurrence_interval;
    if (e.recurrence_by_day !== '') ae.recurrence_by_day = e.recurrence_by_day;
    if (e.recurrence_by_monthday !== '') ae.recurrence_by_monthday = e.recurrence_by_monthday;
    if (e.recurrence_by_month !== '') ae.recurrence_by_month = e.recurrence_by_month;
    if (e.exdates !== '') ae.exdates = e.exdates;
    if (e.rdates !== '') ae.rdates = e.rdates;
    if (e.recurrence_parent_id !== null) ae.recurrence_parent_id = e.recurrence_parent_id;
    if (canonicalRFC3339(e.recurrence_original_start) !== null) {
        ae.recurrence_original_start = e.recurrence_original_start;
    }
    if (e.duration !== '') ae.duration = e.duration;
    if (e.categories !== '') ae.categories = e.categories;
    if (e.url !== '') ae.url = e.url;
    if (e.note_slug !== '') ae.note_slug = e.note_slug;
    if (e.reminder_minutes !== 0) ae.reminder_minutes = e.reminder_minutes;
    if (e.location !== '') ae.location = e.location;
    if (e.latitude !== null) ae.latitude = e.latitude;
    if (e.longitude !== null) ae.longitude = e.longitude;
    ae.calendar_id = e.calendar_id;
    if (calendarName !== '') ae.calendar_name = calendarName;
    if (canonicalRFC3339(e.created_at) !== null) ae.created_at = e.created_at;
    if (canonicalRFC3339(e.updated_at) !== null) ae.updated_at = e.updated_at;
    return ae;
}

/** eventsToAPI — serialises a list, resolving each row's calendar name once. */
function eventsToAPI(state: DemoState, events: DemoEvent[]): Array<Record<string, unknown>> {
    return events.map((e) => modelEventToAPI(e, calendarNameOf(state, e.calendar_id)));
}

/** The LEFT JOIN on calendars: the name, or "" when the calendar is gone. */
function calendarNameOf(state: DemoState, calendarID: number): string {
    const cal = state.calendars.find((c) => c.id === calendarID);
    return cal === undefined ? '' : cal.name;
}

/**
 * The calendar name a freshly inserted row reports. repository.Create only
 * looks the name up when the row names a calendar, so an event that landed in
 * the default calendar comes back without one — and does carry it on the next
 * read, which goes through the join. Quirk of the server, mirrored here so a
 * create response is identical on both backends.
 */
function calendarNameAfterCreate(state: DemoState, e: DemoEvent): string {
    return e.calendar_id === 0 ? '' : calendarNameOf(state, e.calendar_id);
}

// ── Events ───────────────────────────────────────────────────────────────────

/** APIV1EventsGet — the list window, or a search when `q` is given. */
async function apiV1EventsGet(ctx: { url: URL }): Promise<Response> {
    const params = ctx.url.searchParams;
    const q = params.get('q') ?? '';
    if (byteLength(q) > MAX_SEARCH_QUERY_LENGTH) throw new ApiError(400, 'search query too long');

    const calendarIDs = parseCalendarIDsFromParams(params);

    if (q !== '') {
        const from = params.get('from') ?? '';
        const to = params.get('to') ?? '';
        return withStore('read', (state) =>
            jsonResponse(eventsToAPI(state, search(state, q, normalizeParamTime(from), normalizeParamTime(to), calendarIDs))),
        );
    }

    if (!params.has('from') || !params.has('to')) {
        throw new ApiError(400, 'from and to query parameters are required');
    }
    const from = normalizeParamTime(params.get('from') ?? '');
    const to = normalizeParamTime(params.get('to') ?? '');
    if (from === '' || to === '') {
        throw new ApiError(400, 'from and to must be RFC 3339 timestamps');
    }
    return withStore('read', (state) => jsonResponse(eventsToAPI(state, listEvents(state, from, to, calendarIDs))));
}

/**
 * A query-parameter timestamp, normalised to UTC the way the handler does
 * (`params.From.Value.UTC().Format(time.RFC3339)`) so it can be compared with
 * the stored strings. "" when it is not a timestamp at all.
 */
function normalizeParamTime(raw: string): string {
    if (raw === '') return '';
    const t = parseRFC3339(raw);
    return t === null ? '' : formatRFC3339(t);
}

/**
 * parseCalendarIDsFromParams — null means "every calendar", an empty array
 * means "no calendar matches", which is how the server answers a filter naming
 * only unknown calendars.
 */
function parseCalendarIDsFromParams(params: URLSearchParams): number[] | null {
    const raw = params.getAll('calendar_id');
    if (raw.length === 0) return null;
    const ids: number[] = [];
    for (const value of raw) {
        const id = Number(value);
        if (Number.isInteger(id)) ids.push(id);
    }
    return ids;
}

/** True when the row belongs to one of the requested calendars (calendarIDFilter). */
function matchesCalendarFilter(e: DemoEvent, calendarIDs: number[] | null): boolean {
    if (calendarIDs === null) return true;
    return calendarIDs.includes(e.calendar_id);
}

/**
 * EventService.List — the events overlapping [from, to).
 *
 * This is repository.List: plain rows only, since a recurring parent is
 * expanded into instances instead (added in the recurrence port) and an
 * override row is reached through its parent.
 */
function listEvents(state: DemoState, from: string, to: string, calendarIDs: number[] | null): DemoEvent[] {
    const events = state.events.filter(
        (e) =>
            e.start_time < to &&
            e.end_time > from &&
            e.recurrence_freq === '' &&
            e.recurrence_parent_id === null &&
            matchesCalendarFilter(e, calendarIDs),
    );
    return events.sort(compareEvents);
}

/**
 * EventService.Search — the FTS5 index emulated as "every term must match a
 * token of the title or the description", ordered by start_time descending.
 * See AGENTS.md for why this is close enough to SQLite's unicode61 tokenizer.
 */
function search(
    state: DemoState,
    query: string,
    from: string,
    to: string,
    calendarIDs: number[] | null,
): DemoEvent[] {
    const terms = foldForSearch(query).split(/\s+/).filter((t) => t !== '');
    if (terms.length === 0) return []; // sanitizeFTSQuery produced nothing

    const matches = state.events.filter((e) => {
        if (!matchesCalendarFilter(e, calendarIDs)) return false;
        if (from !== '' && to !== '' && !(e.start_time < to && e.end_time > from)) return false;
        const haystack = searchTokens(e);
        return terms.every((term) => haystack.some((token) => token === term));
    });
    return matches.sort((a, b) => (a.start_time === b.start_time ? a.id - b.id : a.start_time < b.start_time ? 1 : -1));
}

/** The tokens FTS5 would index for a row: its title and its description. */
function searchTokens(e: DemoEvent): string[] {
    const description = e.description.replace(/<[^>]*>/g, ' ');
    return foldForSearch(e.title + ' ' + description)
        .split(/\s+/)
        .filter((t) => t !== '');
}

/**
 * Case- and diacritic-folding, plus splitting on punctuation, which is what the
 * unicode61 tokenizer does before matching.
 */
function foldForSearch(s: string): string {
    return s
        .normalize('NFD')
        .replace(/[\u0300-\u036f]/g, '')
        .toLowerCase()
        .replace(/[^\p{L}\p{N}]+/gu, ' ')
        .trim();
}

/** APIV1EventsPost — create. */
async function apiV1EventsPost(ctx: { request: Request }): Promise<Response> {
    const req = await requestBody(ctx.request);
    const { startTime, endTime } = validateCreateEventRequest(req);

    return withStore('write', (state) => {
        const e = newEvent();
        e.id = nextEventID(state);
        e.title = sanitizeHTML(stringOr(req, 'title'));
        e.description = sanitizeHTML(stringOr(req, 'description'));
        e.start_time = startTime;
        e.end_time = endTime;
        e.all_day = isSet(req, 'all_day') ? boolField(req, 'all_day') : false;
        e.color = stringOr(req, 'color');
        e.recurrence_freq = stringOr(req, 'recurrence_freq');
        e.recurrence_count = intOr(req, 'recurrence_count');
        e.recurrence_until = stringOr(req, 'recurrence_until');
        e.recurrence_interval = intOr(req, 'recurrence_interval');
        e.recurrence_by_day = stringOr(req, 'recurrence_by_day');
        e.recurrence_by_monthday = stringOr(req, 'recurrence_by_monthday');
        e.recurrence_by_month = stringOr(req, 'recurrence_by_month');
        e.exdates = stringOr(req, 'exdates');
        e.rdates = stringOr(req, 'rdates');
        e.duration = stringOr(req, 'duration');
        e.categories = sanitizeHTML(stringOr(req, 'categories'));
        e.note_slug = stringOr(req, 'note_slug');
        e.reminder_minutes = intOr(req, 'reminder_minutes');
        e.location = sanitizeHTML(stringOr(req, 'location'));
        if (isSet(req, 'url')) e.url = stringField(req, 'url');
        e.latitude = coordinateField(req, 'latitude');
        e.longitude = coordinateField(req, 'longitude');
        e.created_at = nowRFC3339();
        e.updated_at = e.created_at;

        state.events.push(e);
        return jsonResponse(modelEventToAPI(e, calendarNameAfterCreate(state, e)), 201);
    });
}

/** APIV1EventsIDGet — one event, or one instance of a recurring event. */
async function apiV1EventsIDGet(ctx: { params: string[] }): Promise<Response> {
    const parsed = parseEventID(ctx.params[0]);
    if (parsed === null) throw new ApiError(400, 'invalid id');
    return withStore('read', (state) => {
        const e =
            parsed.instanceStart !== ''
                ? getInstance(state, parsed.dbID, parsed.instanceStart)
                : getByID(state, parsed.dbID);
        return jsonResponse(modelEventToAPI(e, calendarNameOf(state, e.calendar_id)));
    });
}

/** APIV1EventsIDPatch — update an event, or override a single instance. */
async function apiV1EventsIDPatch(ctx: { request: Request; params: string[] }): Promise<Response> {
    const parsed = parseEventID(ctx.params[0]);
    if (parsed === null) throw new ApiError(400, 'invalid id');
    const req = await requestBody(ctx.request);

    return withStore('write', (state) => {
        // A newly created override is a fresh row, so it reports its calendar
        // the way any insert does; an updated one was read back through the join.
        const created = parsed.instanceStart !== '' && getOverride(state, parsed.dbID, parsed.instanceStart) === undefined;
        const e =
            parsed.instanceStart !== ''
                ? createOrUpdateOverride(state, parsed.dbID, parsed.instanceStart, req)
                : updateEvent(state, parsed.dbID, req);
        const calendarName = created ? calendarNameAfterCreate(state, e) : calendarNameOf(state, e.calendar_id);
        return jsonResponse(modelEventToAPI(e, calendarName));
    });
}

/**
 * APIV1EventsIDDelete — deleting an event answers 204, while deleting a single
 * instance excludes that occurrence and answers with the updated parent.
 */
async function apiV1EventsIDDelete(ctx: { params: string[] }): Promise<Response> {
    const parsed = parseEventID(ctx.params[0]);
    if (parsed === null) throw new ApiError(400, 'invalid id');
    return withStore('write', (state) => {
        if (parsed.instanceStart !== '') {
            const e = addExDate(state, parsed.dbID, parsed.instanceStart);
            return jsonResponse(modelEventToAPI(e, calendarNameOf(state, e.calendar_id)));
        }
        deleteEvent(state, parsed.dbID);
        return noContentResponse();
    });
}

// ── Event service ────────────────────────────────────────────────────────────

/** repository.GetByID + the service's not-found mapping. */
function getByID(state: DemoState, id: number): DemoEvent {
    const e = state.events.find((row) => row.id === id);
    if (e === undefined) throw notFoundError();
    return e;
}

/** repository.GetOverride — the override row for one instance, if there is one. */
function getOverride(state: DemoState, parentID: number, originalStart: string): DemoEvent | undefined {
    return state.events.find(
        (row) => row.recurrence_parent_id === parentID && row.recurrence_original_start === originalStart,
    );
}

/**
 * EventService.GetInstance — the override if one exists, otherwise the instance
 * synthesised from the parent, keeping the parent's duration.
 */
function getInstance(state: DemoState, parentID: number, instanceStart: string): DemoEvent {
    if (canonicalRFC3339(instanceStart) === null) {
        throw validationError('instance_start must be RFC 3339 format');
    }
    const override = getOverride(state, parentID, instanceStart);
    if (override !== undefined) return override;

    const parent = state.events.find((row) => row.id === parentID);
    if (parent === undefined || !isRecurring(parent)) throw notFoundError();

    const inst: DemoEvent = { ...parent };
    inst.start_time = instanceStart;
    inst.end_time = endFromStart(instanceStart, eventDuration(parent));
    inst.recurrence_index = 1; // non-zero: this is an expanded instance
    return inst;
}

/** The milliseconds between a stored event's start and end. */
function eventDuration(e: DemoEvent): number {
    return instantOf(e.end_time) - instantOf(e.start_time);
}

/** A stored timestamp as milliseconds, or NaN when the row is malformed. */
function instantOf(rfc3339: string): number {
    const t = parseRFC3339(rfc3339);
    return t === null ? NaN : t.getTime();
}

/**
 * The end of an event that starts at `start` and lasts `durationMs`, written in
 * the offset the start carries — Go's start.Add(d).Format(time.RFC3339).
 */
function endFromStart(start: string, durationMs: number): string {
    return formatRFC3339At(new Date(instantOf(start) + durationMs), offsetOf(start));
}

/** EventService.Update — a partial update of a stored event. */
function updateEvent(state: DemoState, id: number, req: RequestBody): DemoEvent {
    validateUpdateEventRequest(req);
    const existing = getByID(state, id);

    // Sanitising on update as well as create is not optional: an event edited
    // to carry a script must be stripped just like one created with it.
    if (isSet(req, 'title')) existing.title = sanitizeHTML(stringField(req, 'title'));
    if (isSet(req, 'description')) existing.description = sanitizeHTML(stringField(req, 'description'));
    if (isSet(req, 'all_day')) existing.all_day = boolField(req, 'all_day');
    if (isSet(req, 'color')) existing.color = stringField(req, 'color');
    if (isSet(req, 'recurrence_freq')) existing.recurrence_freq = stringField(req, 'recurrence_freq');
    if (isSet(req, 'recurrence_count')) existing.recurrence_count = intField(req, 'recurrence_count');
    if (isSet(req, 'recurrence_until')) existing.recurrence_until = stringField(req, 'recurrence_until');
    if (isSet(req, 'recurrence_interval')) existing.recurrence_interval = intField(req, 'recurrence_interval');
    if (isSet(req, 'recurrence_by_day')) existing.recurrence_by_day = stringField(req, 'recurrence_by_day');
    if (isSet(req, 'recurrence_by_monthday')) {
        existing.recurrence_by_monthday = stringField(req, 'recurrence_by_monthday');
    }
    if (isSet(req, 'recurrence_by_month')) existing.recurrence_by_month = stringField(req, 'recurrence_by_month');
    if (isSet(req, 'exdates')) existing.exdates = stringField(req, 'exdates');
    if (isSet(req, 'rdates')) existing.rdates = stringField(req, 'rdates');
    if (isSet(req, 'duration')) existing.duration = stringField(req, 'duration');
    if (isSet(req, 'categories')) existing.categories = sanitizeHTML(stringField(req, 'categories'));
    if (isSet(req, 'url')) existing.url = stringField(req, 'url');
    if (isSet(req, 'note_slug')) existing.note_slug = stringField(req, 'note_slug');
    if (isSet(req, 'reminder_minutes')) existing.reminder_minutes = intField(req, 'reminder_minutes');
    if (isSet(req, 'location')) existing.location = sanitizeHTML(stringField(req, 'location'));
    if (isSet(req, 'latitude') && !isNull(req, 'latitude')) existing.latitude = numberField(req, 'latitude');
    if (isSet(req, 'longitude') && !isNull(req, 'longitude')) existing.longitude = numberField(req, 'longitude');

    // A new duration recomputes the end from the (still unchanged) start.
    if (isSet(req, 'duration') && stringField(req, 'duration') !== '') {
        const dur = parseDurationOrFail(stringField(req, 'duration'));
        if (parseRFC3339(existing.start_time) === null) {
            throw validationError('invalid start_time for duration computation');
        }
        existing.end_time = endFromStart(existing.start_time, dur);
    }

    if (isSet(req, 'start_date')) {
        existing.start_time = formatRFC3339(dateField(req, 'start_date'));
    }
    if (isSet(req, 'start_time')) {
        existing.start_time = formatRFC3339(dateTimeField(req, 'start_time').date);
    }
    if (isSet(req, 'end_date')) {
        let endT = dateField(req, 'end_date');
        const startParsed = parseRFC3339(existing.start_time);
        if (startParsed !== null && endT.getTime() <= startParsed.getTime()) {
            endT = addDays(startParsed, 1);
        }
        existing.end_time = formatRFC3339(endT);
    }
    if (isSet(req, 'end_time')) {
        existing.end_time = formatRFC3339(dateTimeField(req, 'end_time').date);
    }

    // Turning an event into an all-day one without saying when moves it to the
    // whole day it already started on.
    if (isSet(req, 'all_day') && boolField(req, 'all_day') && !isSet(req, 'start_date') && !isSet(req, 'start_time')) {
        const start = parseRFC3339(existing.start_time);
        if (start !== null) {
            const normalized = truncateToUTCDay(start);
            existing.start_time = formatRFC3339(normalized);
            if (!isSet(req, 'end_date') && !isSet(req, 'end_time')) {
                existing.end_time = formatRFC3339(addDays(normalized, 1));
            }
        }
    }

    if (!(instantOf(existing.end_time) > instantOf(existing.start_time))) {
        throw validationError('end_time must be after start_time');
    }

    existing.updated_at = nowRFC3339();
    return existing;
}

/**
 * EventService.CreateOrUpdateOverride — the per-instance edit: an override row
 * is a copy of the parent pinned to one occurrence through
 * recurrence_parent_id + recurrence_original_start.
 */
function createOrUpdateOverride(
    state: DemoState,
    parentID: number,
    instanceStart: string,
    req: RequestBody,
): DemoEvent {
    if (canonicalRFC3339(instanceStart) === null) {
        throw validationError('instance_start must be RFC 3339 format');
    }
    validateUpdateEventRequest(req);

    const parent = state.events.find((row) => row.id === parentID);
    if (parent === undefined) throw notFoundError();
    if (!isRecurring(parent)) throw validationError('event is not recurring');

    const existing = getOverride(state, parentID, instanceStart);
    if (existing !== undefined) return updateEvent(state, existing.id, req);

    const override = newEvent();
    override.id = nextEventID(state);
    override.title = parent.title;
    override.description = parent.description;
    override.start_time = instanceStart;
    override.all_day = parent.all_day;
    override.color = parent.color;
    override.duration = parent.duration;
    override.categories = parent.categories;
    override.url = parent.url;
    override.note_slug = parent.note_slug;
    override.reminder_minutes = parent.reminder_minutes;
    override.location = parent.location;
    override.latitude = parent.latitude;
    override.longitude = parent.longitude;
    override.calendar_id = parent.calendar_id;
    override.recurrence_parent_id = parentID;
    override.recurrence_original_start = instanceStart;
    override.end_time = endFromStart(instanceStart, eventDuration(parent));

    if (isSet(req, 'title')) override.title = sanitizeHTML(stringField(req, 'title'));
    if (isSet(req, 'description')) override.description = sanitizeHTML(stringField(req, 'description'));
    if (isSet(req, 'start_time')) override.start_time = formatRFC3339(dateTimeField(req, 'start_time').date);
    if (isSet(req, 'start_date')) override.start_time = formatRFC3339(dateField(req, 'start_date'));
    if (isSet(req, 'end_time')) override.end_time = formatRFC3339(dateTimeField(req, 'end_time').date);
    if (isSet(req, 'end_date')) override.end_time = formatRFC3339(dateField(req, 'end_date'));
    if (isSet(req, 'all_day')) override.all_day = boolField(req, 'all_day');
    if (isSet(req, 'color')) override.color = stringField(req, 'color');
    if (isSet(req, 'duration')) {
        override.duration = stringField(req, 'duration');
        if (override.duration !== '') {
            override.end_time = endFromStart(override.start_time, parseDurationOrFail(override.duration));
        }
    }
    if (isSet(req, 'categories')) override.categories = sanitizeHTML(stringField(req, 'categories'));
    if (isSet(req, 'url')) override.url = stringField(req, 'url');
    if (isSet(req, 'note_slug')) override.note_slug = stringField(req, 'note_slug');
    if (isSet(req, 'reminder_minutes')) override.reminder_minutes = intField(req, 'reminder_minutes');
    if (isSet(req, 'location')) override.location = sanitizeHTML(stringField(req, 'location'));
    if (isSet(req, 'latitude') && !isNull(req, 'latitude')) override.latitude = numberField(req, 'latitude');
    if (isSet(req, 'longitude') && !isNull(req, 'longitude')) override.longitude = numberField(req, 'longitude');

    override.created_at = nowRFC3339();
    override.updated_at = override.created_at;
    state.events.push(override);
    return override;
}

/**
 * EventService.AddExDate — cancelling one occurrence: the instance start is
 * appended to the parent's EXDATE list, and any override for it is dropped.
 */
function addExDate(state: DemoState, id: number, instanceStart: string): DemoEvent {
    if (canonicalRFC3339(instanceStart) === null) {
        throw validationError('instance_start must be RFC 3339 format');
    }
    const existing = getByID(state, id);
    if (!isRecurring(existing)) throw validationError('event is not recurring');

    existing.exdates = existing.exdates === '' ? instanceStart : existing.exdates + ',' + instanceStart;
    existing.updated_at = nowRFC3339();

    const override = getOverride(state, id, instanceStart);
    if (override !== undefined) {
        state.events = state.events.filter((row) => row.id !== override.id);
    }
    return existing;
}

/** EventService.Delete — the event and, with it, every override of it. */
function deleteEvent(state: DemoState, id: number): void {
    const exists = state.events.some((row) => row.id === id);
    state.events = state.events.filter((row) => row.id !== id && row.recurrence_parent_id !== id);
    if (!exists) throw notFoundError();
}

// ── Calendars ────────────────────────────────────────────────────────────────

/** APIV1CalendarsGet — every calendar, ordered by id. */
async function apiV1CalendarsGet(): Promise<Response> {
    return withStore('read', (state) =>
        jsonResponse(
            [...state.calendars]
                .sort((a, b) => a.id - b.id)
                .map((c) => ({ id: c.id, name: c.name, color: c.color })),
        ),
    );
}

/** APIV1CalendarsIDPatch — rename or recolour a calendar. */
async function apiV1CalendarsIDPatch(ctx: { request: Request; params: string[] }): Promise<Response> {
    const id = Number(ctx.params[0]);
    if (!Number.isInteger(id)) throw new ApiError(400, 'invalid id');
    const req = await requestBody(ctx.request);
    const name = stringOr(req, 'name');
    const color = stringOr(req, 'color');
    if (byteLength(name) > MAX_CALENDAR_NAME_LENGTH) {
        throw validationError('calendar name must be at most ' + MAX_CALENDAR_NAME_LENGTH + ' characters');
    }
    validateColor(color);

    return withStore('write', (state) => {
        const cal = state.calendars.find((c) => c.id === id);
        if (cal === undefined) throw notFoundError();
        if (name !== '') cal.name = name;
        if (color !== '') cal.color = color;
        return jsonResponse({ id: cal.id, name: cal.name, color: cal.color });
    });
}

// ── Preferences ──────────────────────────────────────────────────────────────

/**
 * APIV1PreferencesGet / APIV1PreferencesPatch — the server's allowlist of
 * preference keys is currently empty (service.allowedPreferences), so it always
 * answers with an empty map and rejects any key. The demo does the same rather
 * than inventing a setting the real deployment does not have.
 */
async function apiV1PreferencesGet(): Promise<Response> {
    return withStore('read', (state) => jsonResponse(allowedPreferences(state)));
}

async function apiV1PreferencesPatch(ctx: { request: Request }): Promise<Response> {
    const req = await requestBody(ctx.request);
    return withStore('write', (state) => {
        for (const key of Object.keys(req)) {
            if (!(key in ALLOWED_PREFERENCES)) {
                throw validationError('unknown preference key: ' + key);
            }
            state.preferences[key] = stringField(req, key);
        }
        return jsonResponse(allowedPreferences(state));
    });
}

/** service.allowedPreferences — key to default value. Empty, as on the server. */
const ALLOWED_PREFERENCES: Record<string, string> = {};

/** PreferencesService.GetAll — stored value or default, for every allowed key. */
function allowedPreferences(state: DemoState): Record<string, string> {
    const result: Record<string, string> = {};
    for (const [key, def] of Object.entries(ALLOWED_PREFERENCES)) {
        result[key] = key in state.preferences ? state.preferences[key] : def;
    }
    return result;
}
