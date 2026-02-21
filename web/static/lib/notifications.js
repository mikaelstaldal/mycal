const STORAGE_KEY = 'mycal_fired_notifications';

function getFired() {
    try {
        return JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}');
    } catch {
        return {};
    }
}

function setFired(fired) {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(fired));
}

function pruneOld(fired) {
    const cutoff = Date.now() - 24 * 60 * 60 * 1000;
    const pruned = {};
    for (const [key, ts] of Object.entries(fired)) {
        if (ts > cutoff) {
            pruned[key] = ts;
        }
    }
    return pruned;
}

export function checkAndNotify(events) {
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

        const key = `${event.id}-${event.recurrence_index || 0}`;
        if (fired[key]) {
            continue;
        }

        const startMs = new Date(event.start_time).getTime();
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

export function requestPermission() {
    if (typeof Notification !== 'undefined' && Notification.permission === 'default') {
        Notification.requestPermission();
    }
}
