const STORAGE_KEY = 'mycal-settings';

const DEFAULTS = {
    weekStartDay: 1,   // 0=Sunday, 1=Monday
    clockFormat: '24h', // '24h' or '12h'
    dateFormat: 'yyyy-MM-dd', // 'yyyy-MM-dd', 'MM/dd/yyyy', 'dd/MM/yyyy'
    defaultView: 'month', // 'month' or 'week'
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

export function saveConfig(config) {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
}
