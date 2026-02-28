// Derive base path from document base URI so the app works behind a reverse proxy on a sub-path.
const APP_BASE = new URL('.', document.baseURI).pathname.replace(/\/$/, '');
const BASE = APP_BASE + '/api/v1/events';

export async function listEvents(from, to) {
    const res = await fetch(`${BASE}?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`);
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function getEvent(id) {
    const res = await fetch(`${BASE}/${id}`);
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

export async function updateEvent(id, data, instanceStart) {
    let url = `${BASE}/${id}`;
    if (instanceStart) {
        url += `?instance_start=${encodeURIComponent(instanceStart)}`;
    }
    const res = await fetch(url, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
    });
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function deleteEvent(id, instanceStart) {
    let url = `${BASE}/${id}`;
    if (instanceStart) {
        url += `?instance_start=${encodeURIComponent(instanceStart)}`;
    }
    const res = await fetch(url, { method: 'DELETE' });
    if (instanceStart) {
        if (!res.ok) throw new Error((await res.json()).error);
        return res.json();
    }
    if (!res.ok) throw new Error((await res.json()).error);
}

export async function searchEvents(query) {
    const res = await fetch(`${BASE}?q=${encodeURIComponent(query)}`);
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function importEvents(icsContentOrUrl) {
    const isUrl = typeof icsContentOrUrl === 'string' && icsContentOrUrl.startsWith('http');
    const res = await fetch(APP_BASE + '/api/v1/import', {
        method: 'POST',
        headers: { 'Content-Type': isUrl ? 'application/json' : 'text/calendar' },
        body: isUrl ? JSON.stringify({ url: icsContentOrUrl }) : icsContentOrUrl,
    });
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}

export async function importSingleEvent(icsContentOrUrl) {
    const isUrl = typeof icsContentOrUrl === 'string' && icsContentOrUrl.startsWith('http');
    const res = await fetch(APP_BASE + '/api/v1/import-single', {
        method: 'POST',
        headers: { 'Content-Type': isUrl ? 'application/json' : 'text/calendar' },
        body: isUrl ? JSON.stringify({ url: icsContentOrUrl }) : icsContentOrUrl,
    });
    if (!res.ok) throw new Error((await res.json()).error);
    return res.json();
}
