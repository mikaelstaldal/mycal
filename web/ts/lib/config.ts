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

export function getConfig(): AppConfig {
    try {
        const stored = localStorage.getItem(STORAGE_KEY);
        if (stored) {
            const parsed = JSON.parse(stored);
            const localeDefault = getLocaleWeekStartDay();
            // If weekStartDay is null or absent, use locale detection
            const weekStartDay = parsed.weekStartDay != null ? parsed.weekStartDay : localeDefault;
            return { ...DEFAULTS, ...parsed, weekStartDay };
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
