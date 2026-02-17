export function startOfMonth(date) {
    return new Date(date.getFullYear(), date.getMonth(), 1);
}

export function endOfMonth(date) {
    return new Date(date.getFullYear(), date.getMonth() + 1, 0);
}

export function addMonths(date, n) {
    return new Date(date.getFullYear(), date.getMonth() + n, 1);
}

export function toRFC3339(date) {
    return date.toISOString();
}

export function formatMonthYear(date) {
    return date.toLocaleDateString(undefined, { month: 'long', year: 'numeric' });
}

export function isSameDay(a, b) {
    return a.getFullYear() === b.getFullYear() &&
           a.getMonth() === b.getMonth() &&
           a.getDate() === b.getDate();
}

export function isToday(date) {
    return isSameDay(date, new Date());
}

export function getCalendarDays(year, month, weekStartDay = 1) {
    const first = new Date(year, month, 1);
    const last = new Date(year, month + 1, 0);
    const firstDow = first.getDay(); // 0=Sun
    const offset = (firstDow - weekStartDay + 7) % 7;
    const days = [];

    // Previous month padding
    for (let i = offset - 1; i >= 0; i--) {
        days.push({ date: new Date(year, month, -i), currentMonth: false });
    }

    // Current month
    for (let i = 1; i <= last.getDate(); i++) {
        days.push({ date: new Date(year, month, i), currentMonth: true });
    }

    // Next month padding to fill grid (6 rows)
    while (days.length < 42) {
        const nextDay = days.length - offset - last.getDate() + 1;
        days.push({ date: new Date(year, month + 1, nextDay), currentMonth: false });
    }

    return days;
}

const ALL_WEEKDAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

export function getWeekdays(weekStartDay = 1) {
    return [...ALL_WEEKDAYS.slice(weekStartDay), ...ALL_WEEKDAYS.slice(0, weekStartDay)];
}

export function formatTime(dateStr, clockFormat = '24h') {
    const d = new Date(dateStr);
    const h = d.getHours();
    const m = String(d.getMinutes()).padStart(2, '0');
    if (clockFormat === '12h') {
        const period = h >= 12 ? 'PM' : 'AM';
        const h12 = h % 12 || 12;
        return `${h12}:${m} ${period}`;
    }
    return `${String(h).padStart(2, '0')}:${m}`;
}

export function formatDate(date, dateFormat = 'yyyy-MM-dd') {
    const y = date.getFullYear();
    const m = String(date.getMonth() + 1).padStart(2, '0');
    const d = String(date.getDate()).padStart(2, '0');
    switch (dateFormat) {
        case 'MM/dd/yyyy': return `${m}/${d}/${y}`;
        case 'dd/MM/yyyy': return `${d}/${m}/${y}`;
        default: return `${y}-${m}-${d}`;
    }
}

export function toLocalDatetimeValue(dateStr) {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    const pad = n => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function fromLocalDatetimeValue(val) {
    if (!val) return '';
    return new Date(val).toISOString();
}
