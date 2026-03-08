const STORAGE_KEY = 'mycal-settings';

const DEFAULTS = {
    weekStartDay: 1,   // 0=Sunday, 1=Monday
    defaultView: 'week', // 'year', 'month', 'week', 'day', or 'schedule'
    dayStartHour: 8, // 0-23, hour to scroll to in week view
    defaultEventColor: 'dodgerblue', // fallback until server preferences load
    mapProvider: 'none', // 'none', 'openstreetmap', 'google'
    googleMapsApiKey: '', // only needed when mapProvider is 'google'
};

export function getConfig() {
    try {
        const stored = localStorage.getItem(STORAGE_KEY);
        if (stored) {
            return { ...DEFAULTS, ...JSON.parse(stored) };
        }
    } catch (e) {
        // ignore corrupt data
    }
    return { ...DEFAULTS };
}

export function hasUserDefaultView() {
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

export function saveConfig(config) {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
}
