import type { CalendarEvent, CalendarMeta, Feed, Preferences } from '../types/models.js';

// Derive base path from document base URI so the app works behind a reverse proxy on a sub-path.
const APP_BASE = new URL('.', document.baseURI).pathname.replace(/\/$/, '');
const BASE = APP_BASE + '/api/v1/events';

export async function listEvents(from: string, to: string, calendarIds?: number[] | null): Promise<CalendarEvent[]> {
    let url = `${BASE}?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`;
    if (calendarIds) {
        for (const id of calendarIds) {
            url += `&calendar_id=${encodeURIComponent(id)}`;
        }
    }
    const res = await fetch(url);
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function getEvent(id: string): Promise<CalendarEvent> {
    const res = await fetch(`${BASE}/${encodeURIComponent(id)}`);
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function createEvent(data: Partial<CalendarEvent>): Promise<CalendarEvent> {
    const res = await fetch(BASE, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function updateEvent(id: string, data: Partial<CalendarEvent>): Promise<CalendarEvent> {
    const res = await fetch(`${BASE}/${encodeURIComponent(id)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function deleteEvent(id: string): Promise<void> {
    const url = `${BASE}/${encodeURIComponent(id)}`;
    const res = await fetch(url, { method: 'DELETE' });
    if (!res.ok) throw new Error((await res.json()).error);
}

export async function searchEvents(query: string): Promise<CalendarEvent[]> {
    const res = await fetch(`${BASE}?q=${encodeURIComponent(query)}`);
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function importEvents(icsContentOrUrl: string, calendarName?: string): Promise<{ imported: number }> {
    const isUrl = typeof icsContentOrUrl === 'string' && icsContentOrUrl.startsWith('http');
    let url = APP_BASE + '/api/v1/import';
    if (calendarName) url += `?calendar=${encodeURIComponent(calendarName)}`;
    const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': isUrl ? 'application/json' : 'text/calendar' },
        body: isUrl ? JSON.stringify({ url: icsContentOrUrl }) : icsContentOrUrl,
    });
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function getPreferences(): Promise<Preferences> {
    const res = await fetch(APP_BASE + '/api/v1/preferences');
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function updatePreferences(prefs: Preferences): Promise<Preferences> {
    const res = await fetch(APP_BASE + '/api/v1/preferences', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(prefs),
    });
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

// Feed subscriptions
const FEEDS_BASE = APP_BASE + '/api/v1/feeds';

export async function listFeeds(): Promise<Feed[]> {
    const res = await fetch(FEEDS_BASE);
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function createFeed(data: Partial<Feed>): Promise<Feed> {
    const res = await fetch(FEEDS_BASE, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function deleteFeed(id: number): Promise<void> {
    const res = await fetch(`${FEEDS_BASE}/${encodeURIComponent(id)}`, { method: 'DELETE' });
    if (!res.ok) throw new Error((await res.json()).error);
}

export async function refreshFeed(id: number): Promise<Feed> {
    const res = await fetch(`${FEEDS_BASE}/${encodeURIComponent(id)}/refresh`, { method: 'POST' });
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function importSingleEvent(icsContentOrUrl: string, calendarName?: string): Promise<CalendarEvent> {
    const isUrl = typeof icsContentOrUrl === 'string' && icsContentOrUrl.startsWith('http');
    let url = APP_BASE + '/api/v1/import-single';
    if (calendarName) url += `?calendar=${encodeURIComponent(calendarName)}`;
    const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': isUrl ? 'application/json' : 'text/calendar' },
        body: isUrl ? JSON.stringify({ url: icsContentOrUrl }) : icsContentOrUrl,
    });
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

// Calendars
const CALENDARS_BASE = APP_BASE + '/api/v1/calendars';

export async function listCalendars(): Promise<CalendarMeta[]> {
    const res = await fetch(CALENDARS_BASE);
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function updateCalendar(id: number, data: Partial<CalendarMeta>): Promise<CalendarMeta> {
    const res = await fetch(`${CALENDARS_BASE}/${encodeURIComponent(id)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}
