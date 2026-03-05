// Derive base path from document base URI so the app works behind a reverse proxy on a sub-path.
const APP_BASE = new URL('.', document.baseURI).pathname.replace(/\/$/, '');
const BASE = APP_BASE + '/api/v1/events';

export async function listEvents(from, to) {
    const res = await fetch(`${BASE}?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`);
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function getEvent(id) {
    const res = await fetch(`${BASE}/${encodeURIComponent(id)}`);
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function createEvent(data) {
    const res = await fetch(BASE, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function updateEvent(id, data) {
    const res = await fetch(`${BASE}/${encodeURIComponent(id)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function deleteEvent(id) {
    const url = `${BASE}/${encodeURIComponent(id)}`;
    const res = await fetch(url, { method: 'DELETE' });
    if (!res.ok) throw new Error((await res.json()).error);
}

export async function searchEvents(query) {
    const res = await fetch(`${BASE}?q=${encodeURIComponent(query)}`);
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function importEvents(icsContentOrUrl, calendarName) {
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

export async function getPreferences() {
    const res = await fetch(APP_BASE + '/api/v1/preferences');
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function updatePreferences(prefs) {
    const res = await fetch(APP_BASE + '/api/v1/preferences', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(prefs),
    });
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function importSingleEvent(icsContentOrUrl, calendarName) {
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
