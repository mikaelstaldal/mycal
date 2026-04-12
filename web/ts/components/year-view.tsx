import { h } from 'preact';
import type { VNode } from 'preact';
import { useMemo } from 'preact/hooks';
import { getCalendarDays, getWeekdays, isToday, getISOWeekNumber, eventStartStr } from '../lib/date-utils.js';
import type { CalendarEvent, AppConfig } from '../types/models.js';

interface YearViewProps {
    currentDate: Date;
    events: CalendarEvent[];
    onMonthClick: (month: number) => void;
    onWeekClick: (date: Date) => void;
    onDayClick: (date: Date) => void;
    config: AppConfig;
    highlightEventId?: string | null;
}

export function YearView({ currentDate, events, onMonthClick, onWeekClick, onDayClick, config, highlightEventId }: YearViewProps): VNode | null {
    const year = currentDate.getFullYear();
    const weekStartDay = config.weekStartDay;
    const weekdays = getWeekdays(weekStartDay);

    const pad = (n: number) => String(n).padStart(2, '0');
    const toDateStr = (y: number, mo: number, d: number) => `${y}-${pad(mo + 1)}-${pad(d)}`;

    // Pre-index events by date string for O(1) per-day lookup instead of O(n) per day
    const eventsByDate = useMemo(() => {
        const map = new Map<string, CalendarEvent[]>();
        for (const e of events) {
            if (e.all_day) {
                const endStr = e.end_date ?? '';
                const cursor = new Date((e.start_date ?? '') + 'T00:00:00');
                while (true) {
                    const key = toDateStr(cursor.getFullYear(), cursor.getMonth(), cursor.getDate());
                    if (key >= endStr) break;
                    let list = map.get(key);
                    if (!list) map.set(key, list = []);
                    list.push(e);
                    cursor.setDate(cursor.getDate() + 1);
                }
            } else {
                const start = new Date(e.start_time!);
                const end = new Date(e.end_time!);
                const cursor = new Date(start.getFullYear(), start.getMonth(), start.getDate());
                while (end > cursor) {
                    const key = toDateStr(cursor.getFullYear(), cursor.getMonth(), cursor.getDate());
                    let list = map.get(key);
                    if (!list) map.set(key, list = []);
                    list.push(e);
                    cursor.setDate(cursor.getDate() + 1);
                }
            }
        }
        return map;
    }, [events]);

    function renderMonth(month: number): VNode {
        const monthDate = new Date(year, month, 1);
        const monthName = monthDate.toLocaleDateString(undefined, { month: 'long' });
        const days = getCalendarDays(year, month, weekStartDay);

        let weeks = [];
        for (let i = 0; i < days.length; i += 7) {
            weeks.push(days.slice(i, i + 7));
        }
        const lastWeek = weeks[weeks.length - 1];
        if (lastWeek.every(d => !d.currentMonth)) {
            weeks = weeks.slice(0, -1);
        }

        return (
            <div class="year-month">
                <div class="year-month-header" role="button" tabIndex={0} onClick={() => onMonthClick(month)} onKeyDown={(ev: KeyboardEvent) => { if (ev.key === 'Enter' || ev.key === ' ') { ev.preventDefault(); onMonthClick(month); } }}>
                    {monthName}
                </div>
                <div class="year-month-grid">
                    <div class="year-weekday-row">
                        <div></div>
                        {weekdays.map(d => <div class="year-weekday">{d.charAt(0)}</div>)}
                    </div>
                    {weeks.map(week => (
                        <div class="year-week-row">
                            <div class="year-week-number" role="button" tabIndex={0} onClick={(ev: MouseEvent) => { ev.stopPropagation(); onWeekClick(week[0].date); }} onKeyDown={(ev: KeyboardEvent) => { if (ev.key === 'Enter' || ev.key === ' ') { ev.preventDefault(); ev.stopPropagation(); onWeekClick(week[0].date); } }}>
                                week {getISOWeekNumber(week[0].date)}
                            </div>
                            {week.map(({ date, currentMonth }) => {
                                const dayEvents = currentMonth ? (eventsByDate.get(toDateStr(date.getFullYear(), date.getMonth(), date.getDate())) ?? []) : [];
                                const hasEvents = dayEvents.length > 0;
                                const isHighlighted = highlightEventId && dayEvents.some(e => (e.id + '|' + eventStartStr(e)) === highlightEventId);
                                const classes = ['year-day',
                                    !currentMonth && 'year-day-other',
                                    currentMonth && isToday(date) && 'year-day-today',
                                    hasEvents && 'year-day-has-events',
                                    isHighlighted && 'highlight-event'
                                ].filter(Boolean).join(' ');
                                return <div class={classes}
                                    style={currentMonth ? 'cursor: pointer' : ''}
                                    onClick={currentMonth ? (ev: MouseEvent) => { ev.stopPropagation(); onDayClick(date); } : undefined}>{date.getDate()}</div>;
                            })}
                        </div>
                    ))}
                </div>
            </div>
        );
    }

    return (
        <div class="year-view">
            {Array.from({ length: 12 }, (_, i) => renderMonth(i))}
        </div>
    );
}
