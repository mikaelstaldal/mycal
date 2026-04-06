import type { CalendarEvent } from '../types/models.js';

export function startOfMonth(date: Date): Date {
    return new Date(date.getFullYear(), date.getMonth(), 1);
}

export function endOfMonth(date: Date): Date {
    return new Date(date.getFullYear(), date.getMonth() + 1, 0);
}

export function addMonths(date: Date, n: number): Date {
    return new Date(date.getFullYear(), date.getMonth() + n, 1);
}

export function toRFC3339(date: Date): string {
    return date.toISOString();
}

export function formatMonthYear(date: Date): string {
    return date.toLocaleDateString(undefined, { month: 'long', year: 'numeric' });
}

export function isSameDay(a: Date, b: Date): boolean {
    return a.getFullYear() === b.getFullYear() &&
           a.getMonth() === b.getMonth() &&
           a.getDate() === b.getDate();
}

export function isToday(date: Date): boolean {
    return isSameDay(date, new Date());
}

export function getCalendarDays(year: number, month: number, weekStartDay: number = 1): { date: Date; currentMonth: boolean }[] {
    const first = new Date(year, month, 1);
    const last = new Date(year, month + 1, 0);
    const firstDow = first.getDay(); // 0=Sun
    const offset = (firstDow - weekStartDay + 7) % 7;
    const days: { date: Date; currentMonth: boolean }[] = [];

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

export function getWeekdays(weekStartDay: number = 1): string[] {
    return [...ALL_WEEKDAYS.slice(weekStartDay), ...ALL_WEEKDAYS.slice(0, weekStartDay)];
}

export function formatTime(dateStr: string): string {
    const d = new Date(dateStr);
    return d.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' });
}

export function formatDate(date: Date): string {
    return date.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
}

export function toLocalDatetimeValue(dateStr: string): string {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function fromLocalDatetimeValue(val: string): string {
    if (!val) return '';
    return new Date(val).toISOString();
}

export function toLocalDateValue(dateStr: string): string {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())}`;
}

export function formatDateOnly(dateStr: string): string {
    const d = new Date(dateStr);
    return formatDate(d);
}

// Convert exclusive end date (server) to inclusive end date (UI).
// A single-day event on Feb 25 is stored as end=Feb 26; display as Feb 25.
export function exclusiveToInclusiveDate(dateStr: string): string {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    d.setUTCDate(d.getUTCDate() - 1);
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${d.getUTCFullYear()}-${pad(d.getUTCMonth()+1)}-${pad(d.getUTCDate())}`;
}

// Convert inclusive end date (UI) to exclusive end date (server).
// User enters Feb 25; send as Feb 26 to the server.
export function inclusiveToExclusiveDate(dateStr: string): string {
    if (!dateStr) return '';
    const d = new Date(dateStr + 'T00:00:00Z');
    d.setUTCDate(d.getUTCDate() + 1);
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${d.getUTCFullYear()}-${pad(d.getUTCMonth()+1)}-${pad(d.getUTCDate())}`;
}

export function getISOWeekNumber(date: Date): number {
    const d = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()));
    d.setUTCDate(d.getUTCDate() + 4 - (d.getUTCDay() || 7));
    const yearStart = new Date(Date.UTC(d.getUTCFullYear(), 0, 1));
    return Math.ceil(((d.getTime() - yearStart.getTime()) / 86400000 + 1) / 7);
}

export function startOfWeek(date: Date, weekStartDay: number = 1): Date {
    const d = new Date(date.getFullYear(), date.getMonth(), date.getDate());
    const day = d.getDay();
    const diff = (day - weekStartDay + 7) % 7;
    d.setDate(d.getDate() - diff);
    return d;
}

export function addWeeks(date: Date, n: number): Date {
    return new Date(date.getFullYear(), date.getMonth(), date.getDate() + n * 7);
}

export function getWeekDays(date: Date, weekStartDay: number = 1): Date[] {
    const start = startOfWeek(date, weekStartDay);
    const days: Date[] = [];
    for (let i = 0; i < 7; i++) {
        days.push(new Date(start.getFullYear(), start.getMonth(), start.getDate() + i));
    }
    return days;
}

export function formatWeekRange(date: Date, weekStartDay: number = 1): string {
    const days = getWeekDays(date, weekStartDay);
    const first = days[0];
    const last = days[6];
    const opts: Intl.DateTimeFormatOptions = { month: 'short', day: 'numeric' };
    const firstStr = first.toLocaleDateString(undefined, opts);
    const lastStr = last.toLocaleDateString(undefined, opts);
    const year = last.getFullYear();
    if (first.getMonth() === last.getMonth()) {
        return `${firstStr} \u2013 ${last.getDate()}, ${year}`;
    }
    return `${firstStr} \u2013 ${lastStr}, ${year}`;
}

export function isPastEvent(event: CalendarEvent): boolean {
    return new Date(event.end_time) < new Date();
}

export function formatDayHeading(date: Date): string {
    return date.toLocaleDateString(undefined, { weekday: 'long', month: 'short', day: 'numeric', year: 'numeric' });
}

export function formatHour(hour: number): string {
    const d = new Date(2000, 0, 1, hour, 0, 0);
    return d.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' });
}

export function getTimezoneAbbr(): string {
    return Intl.DateTimeFormat(undefined, { timeZoneName: 'short' })
        .formatToParts(new Date())
        .find(p => p.type === 'timeZoneName')?.value || '';
}
