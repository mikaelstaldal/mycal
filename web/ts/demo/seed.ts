// The demo's starting content: a week of sample events, written on first run.
//
// This mirrors no Go package. The real server has no seeding command — a fresh
// MyCal is meant to be empty — but a demo that opens on an empty grid shows
// nothing of what MyCal does, so the browser-side store starts populated
// instead. Clearing the site's data brings this back (see store.ts).
//
// A classic worker script: no imports, no exports (see model.ts).
//
// Everything is placed relative to the day the visitor first opens the demo, not
// to the day the bundle was built. A static bundle published once and served for
// months would otherwise drift further from "this week" with every visit.

/**
 * The first day of the week the app will be showing, as a local-midnight Date.
 *
 * Ported from startOfWeek + getLocaleWeekStartDay in web/ts/util/config.ts and
 * web/ts/util/date-utils.ts. The seed only ever runs against a fresh store, at
 * which point there are no stored preferences, so the locale is what the app
 * will use too — and the events land in the week the visitor actually sees
 * rather than a Monday-based one their locale does not draw.
 */
function seedWeekStart(now: Date): Date {
    const firstDay = localeWeekStartDay();
    const d = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    d.setDate(d.getDate() - ((d.getDay() - firstDay + 7) % 7));
    return d;
}

/** 0 = Sunday, 1 = Monday. Mirrors config.ts getLocaleWeekStartDay. */
function localeWeekStartDay(): number {
    try {
        const locale = new Intl.Locale(navigator.language);
        const anyLocale = locale as unknown as {
            weekInfo?: { firstDay?: number };
            getWeekInfo?: () => { firstDay?: number };
        };
        const weekInfo = anyLocale.weekInfo ?? anyLocale.getWeekInfo?.();
        if (weekInfo != null && weekInfo.firstDay != null) {
            // Intl weekInfo.firstDay: 1=Monday … 7=Sunday; convert 7→0.
            return weekInfo.firstDay === 7 ? 0 : weekInfo.firstDay;
        }
    } catch {
        // Older engine, or a language tag Intl.Locale rejects: fall through.
    }
    return 1;
}

/**
 * The date of a given weekday within the displayed week, as a local-midnight
 * Date. Addressing days by weekday rather than by offset from the week start is
 * what lets the content below say "Saturday" and mean it: the displayed week
 * holds each weekday exactly once whether it runs Monday-first or Sunday-first.
 */
function seedDay(weekStart: Date, weekday: number): Date {
    const offset = (weekday - weekStart.getDay() + 7) % 7;
    return new Date(weekStart.getFullYear(), weekStart.getMonth(), weekStart.getDate() + offset);
}

const SUN = 0, MON = 1, TUE = 2, WED = 3, THU = 4, FRI = 5, SAT = 6;

/** A timed sample event, given in the visitor's own wall-clock time. */
interface SeedTimed {
    weekday: number;
    /** Local start, as [hour, minute]. */
    from: [number, number];
    /** Local end, as [hour, minute]. */
    to: [number, number];
    title: string;
    color: string;
    description?: string;
    location?: string;
    latitude?: number;
    longitude?: number;
    categories?: string;
    url?: string;
    reminder_minutes?: number;
}

/** An all-day sample event, spanning `days` whole days from `weekday`. */
interface SeedAllDay {
    weekday: number;
    days: number;
    title: string;
    color: string;
    description?: string;
    location?: string;
    categories?: string;
    url?: string;
}

// Every weekday carries something, so whichever day the visitor arrives on, the
// day and schedule views open on a populated calendar rather than an empty one.
const SEED_TIMED: SeedTimed[] = [
    {
        weekday: MON, from: [9, 0], to: [9, 15],
        title: 'Team standup', color: 'dodgerblue', categories: 'Work',
        reminder_minutes: 5,
    },
    {
        weekday: MON, from: [14, 0], to: [15, 30],
        title: 'Design review: calendar sync', color: 'cornflowerblue', categories: 'Work',
        description: '<p>Walk through the <strong>sync conflict</strong> cases:</p>'
            + '<ul><li>Same event edited on two devices</li><li>Deleted upstream, edited locally</li></ul>',
    },
    {
        // Not before 08:00: the week view scrolls to config.dayStartHour (8) on
        // load, so an earlier event would open clipped above the fold.
        weekday: TUE, from: [8, 0], to: [9, 0],
        title: 'Morning swim', color: 'mediumturquoise', categories: 'Health',
    },
    {
        weekday: TUE, from: [12, 0], to: [13, 0],
        title: 'Lunch with Sofia', color: 'salmon', categories: 'Personal',
        location: 'Café Pascal, Norrtullsgatan 4, Stockholm',
        latitude: 59.3448, longitude: 18.0505,
    },
    {
        weekday: WED, from: [17, 30], to: [18, 30],
        title: 'Guitar lesson', color: 'orange', categories: 'Personal',
    },
    {
        weekday: THU, from: [9, 30], to: [10, 30],
        title: '1:1 with Priya', color: 'green', categories: 'Work',
        description: '<p>Bring the roadmap draft.</p>',
    },
    {
        weekday: FRI, from: [11, 0], to: [12, 0],
        title: 'Sprint demo', color: 'dodgerblue', categories: 'Work',
        reminder_minutes: 15,
    },
    {
        weekday: FRI, from: [15, 30], to: [16, 30],
        title: 'Retro and fika', color: 'orange', categories: 'Work',
    },
    {
        weekday: SAT, from: [19, 0], to: [23, 0],
        title: 'Dinner party', color: 'salmon', categories: 'Personal',
        location: 'At home',
    },
    {
        weekday: SUN, from: [10, 0], to: [16, 0],
        title: 'Hike in Tyresta', color: 'green', categories: 'Outdoors',
        description: '<p>Take the 09:10 bus. Pack lunch.</p>',
    },
];

const SEED_ALL_DAY: SeedAllDay[] = [
    {
        weekday: WED, days: 2,
        title: 'Nordic Calendar Conf', color: 'gold', categories: 'Work',
        location: 'Stockholm',
        url: 'https://example.com/nordic-calendar-conf',
        description: '<p>Two days of talks. <em>Badge is in the hallway drawer.</em></p>',
    },
    {
        weekday: SAT, days: 1,
        title: "Anna's birthday", color: 'red', categories: 'Personal',
    },
];

/**
 * The rows a fresh demo store starts with, ordered and numbered the way the
 * emulated AUTOINCREMENT would have handed them out.
 *
 * The rows are built directly rather than pushed through the create handler,
 * because they are storage-layer content — the equivalent of rows the server
 * would have been started with. Everything that a create request would have had
 * normalised is therefore applied here: descriptions go through sanitizeHTML,
 * timed events are stored as UTC (which is what the web UI sends), and an
 * all-day event is UTC midnight to an exclusive end, exactly as validate.ts
 * writes them.
 */
function seedEvents(now: Date): DemoEvent[] {
    const weekStart = seedWeekStart(now);
    const timestamp = formatRFC3339(now);
    const events: DemoEvent[] = [];

    for (const s of SEED_TIMED) {
        const day = seedDay(weekStart, s.weekday);
        const start = new Date(day.getFullYear(), day.getMonth(), day.getDate(), s.from[0], s.from[1]);
        const end = new Date(day.getFullYear(), day.getMonth(), day.getDate(), s.to[0], s.to[1]);
        events.push({
            ...newEvent(),
            title: s.title,
            description: sanitizeHTML(s.description ?? ''),
            // The web UI sends toISOString(), so a stored timed event is UTC.
            start_time: formatRFC3339(start),
            end_time: formatRFC3339(end),
            all_day: false,
            color: s.color,
            categories: s.categories ?? '',
            url: s.url ?? '',
            reminder_minutes: s.reminder_minutes ?? 0,
            location: s.location ?? '',
            latitude: s.latitude ?? null,
            longitude: s.longitude ?? null,
            created_at: timestamp,
            updated_at: timestamp,
        });
    }

    for (const s of SEED_ALL_DAY) {
        const day = seedDay(weekStart, s.weekday);
        // An all-day event is anchored to the calendar date, so its timestamps
        // are UTC midnight regardless of where the visitor is, and the end is
        // exclusive — the day after the last one the event covers.
        const start = new Date(Date.UTC(day.getFullYear(), day.getMonth(), day.getDate()));
        events.push({
            ...newEvent(),
            title: s.title,
            description: sanitizeHTML(s.description ?? ''),
            start_time: formatRFC3339(start),
            end_time: formatRFC3339(addDays(start, s.days)),
            all_day: true,
            color: s.color,
            categories: s.categories ?? '',
            url: s.url ?? '',
            location: s.location ?? '',
            created_at: timestamp,
            updated_at: timestamp,
        });
    }

    events.sort(compareEvents);
    events.forEach((e, i) => {
        e.id = i + 1;
    });
    return events;
}
