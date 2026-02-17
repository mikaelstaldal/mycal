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

export function getCalendarDays(year, month) {
    const first = new Date(year, month, 1);
    const last = new Date(year, month + 1, 0);
    const startDay = first.getDay(); // 0=Sun
    const days = [];

    // Previous month padding
    for (let i = startDay - 1; i >= 0; i--) {
        const d = new Date(year, month, -i);
        days.push({ date: d, currentMonth: false });
    }

    // Current month
    for (let i = 1; i <= last.getDate(); i++) {
        days.push({ date: new Date(year, month, i), currentMonth: true });
    }

    // Next month padding to fill grid (6 rows)
    while (days.length < 42) {
        const d = new Date(year, month + 1, days.length - startDay - last.getDate() + 1);
        days.push({ date: d, currentMonth: false });
    }

    return days;
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
