import type { AppConfig } from '../types/models.js';

const STORAGE_KEY = 'mycal-settings';

// Detect week start day from browser locale using Intl.Locale API.
// Returns 0 for Sunday, 1 for Monday (matching JS Date.getDay() convention).
function getLocaleWeekStartDay(): number {
    try {
        const locale = new Intl.Locale(navigator.language);
        const weekInfo = (locale as any).weekInfo || ((locale as any).getWeekInfo && (locale as any).getWeekInfo());
        if (weekInfo && weekInfo.firstDay != null) {
            // Intl weekInfo.firstDay: 1=Monday … 7=Sunday; convert 7→0
            return weekInfo.firstDay === 7 ? 0 : weekInfo.firstDay;
        }
    } catch (e) {
        // fallback
    }
    return 1; // default to Monday
}

const DEFAULTS: AppConfig = {
    defaultView: 'week', // 'year', 'month', 'week', 'day', or 'schedule'
    dayStartHour: 8, // 0-23, hour to scroll to in week view
    weekStartDay: 1, // null = auto-detect from locale, 0 = Sunday, 1 = Monday
    defaultEventColor: 'dodgerblue', // fallback until server preferences load
    mapProvider: 'none', // 'none', 'openstreetmap', 'google'
    googleMapsApiKey: '', // only needed when mapProvider is 'google'
};

const VALID_VIEWS = ['year', 'month', 'week', 'day', 'schedule'] as const;
const VALID_MAP_PROVIDERS = ['none', 'openstreetmap', 'google'] as const;
const GOOGLE_API_KEY_RE = /^[A-Za-z0-9_-]{0,200}$/;
// CSS color: named colors, #hex, rgb(), hsl() — restrict to safe printable ASCII, no quotes or angle brackets
const CSS_COLOR_RE = /^[A-Za-z0-9#(),%. -]{1,100}$/;

function sanitize(parsed: unknown, localeWeekStart: number): AppConfig {
    if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
        return { ...DEFAULTS, weekStartDay: localeWeekStart };
    }
    const p = parsed as Record<string, unknown>;

    const defaultView = VALID_VIEWS.includes(p.defaultView as any)
        ? (p.defaultView as AppConfig['defaultView'])
        : DEFAULTS.defaultView;

    const rawHour = Number(p.dayStartHour);
    const dayStartHour = Number.isInteger(rawHour) && rawHour >= 0 && rawHour <= 23
        ? rawHour
        : DEFAULTS.dayStartHour;

    const rawWSD = p.weekStartDay;
    const weekStartDay = rawWSD != null
        ? (rawWSD === 0 || rawWSD === 1 ? (rawWSD as 0 | 1) : DEFAULTS.weekStartDay)
        : localeWeekStart;

    const defaultEventColor = typeof p.defaultEventColor === 'string' && CSS_COLOR_RE.test(p.defaultEventColor)
        ? p.defaultEventColor
        : DEFAULTS.defaultEventColor;

    const mapProvider = VALID_MAP_PROVIDERS.includes(p.mapProvider as any)
        ? (p.mapProvider as AppConfig['mapProvider'])
        : DEFAULTS.mapProvider;

    const googleMapsApiKey = typeof p.googleMapsApiKey === 'string' && GOOGLE_API_KEY_RE.test(p.googleMapsApiKey)
        ? p.googleMapsApiKey
        : DEFAULTS.googleMapsApiKey;

    let calendarColors: Record<number, string> | undefined;
    if (typeof p.calendarColors === 'object' && p.calendarColors !== null && !Array.isArray(p.calendarColors)) {
        const cc: Record<number, string> = {};
        for (const [k, v] of Object.entries(p.calendarColors as Record<string, unknown>)) {
            const id = Number(k);
            if (Number.isInteger(id) && id > 0 && typeof v === 'string' && CSS_COLOR_RE.test(v)) {
                cc[id] = v;
            }
        }
        calendarColors = cc;
    }

    return { defaultView, dayStartHour, weekStartDay, defaultEventColor, mapProvider, googleMapsApiKey, ...(calendarColors !== undefined && { calendarColors }) };
}

export function getConfig(): AppConfig {
    try {
        const stored = localStorage.getItem(STORAGE_KEY);
        if (stored) {
            const parsed = JSON.parse(stored);
            return sanitize(parsed, getLocaleWeekStartDay());
        }
    } catch (e) {
        // ignore corrupt data
    }
    return { ...DEFAULTS, weekStartDay: getLocaleWeekStartDay() };
}

export function hasUserDefaultView(): boolean {
    try {
        const stored = localStorage.getItem(STORAGE_KEY);
        if (stored) {
            return 'defaultView' in JSON.parse(stored);
        }
    } catch (e) {
        // ignore corrupt data
    }
    return false;
}

export function saveConfig(config: AppConfig): void {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
}
