import type { components } from '../api/types.js';
type CalendarEvent = components['schemas']['Event'];

const STORAGE_KEY = 'mycal_fired_notifications';

function getFired(): Record<string, number> {
    try {
        const parsed = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}');
        if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
            return {};
        }
        const result: Record<string, number> = {};
        for (const [k, v] of Object.entries(parsed)) {
            if (typeof k === 'string' && typeof v === 'number' && Number.isFinite(v)) {
                result[k] = v;
            }
        }
        return result;
    } catch {
        return {};
    }
}

function setFired(fired: Record<string, number>): void {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(fired));
}

function pruneOld(fired: Record<string, number>): Record<string, number> {
    const cutoff = Date.now() - 24 * 60 * 60 * 1000;
    const pruned: Record<string, number> = {};
    for (const [key, ts] of Object.entries(fired)) {
        if (ts > cutoff) {
            pruned[key] = ts;
        }
    }
    return pruned;
}

export function checkAndNotify(events: CalendarEvent[]): void {
    if (typeof Notification === 'undefined' || Notification.permission !== 'granted') {
        return;
    }

    const now = Date.now();
    const fired = pruneOld(getFired());
    let changed = false;

    for (const event of events) {
        if (event.all_day || !event.reminder_minutes || event.reminder_minutes <= 0) {
            continue;
        }

        const key = event.id;
        if (fired[key]) {
            continue;
        }

        const startMs = new Date(event.start_time!).getTime();
        const reminderMs = startMs - event.reminder_minutes * 60 * 1000;

        if (now >= reminderMs && now < startMs) {
            try {
                new Notification(event.title, {
                    body: `Starting in ${event.reminder_minutes} minutes`,
                    tag: key,
                });
            } catch (e) {
                console.error('Notification failed:', e);
            }
            fired[key] = now;
            changed = true;
        }
    }

    if (changed) {
        setFired(fired);
    }
}

export function requestPermission(): void {
    if (typeof Notification !== 'undefined' && Notification.permission === 'default') {
        Notification.requestPermission();
    }
}
