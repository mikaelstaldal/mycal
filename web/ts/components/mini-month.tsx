import { h } from 'preact';
import type { VNode } from 'preact';
import { useState, useMemo } from 'preact/hooks';
import { getCalendarDays, getWeekdays, isToday } from '../lib/date-utils.js';
import type { AppConfig } from '../types/models.js';

interface MiniMonthProps {
    currentDate: Date;
    onDayClick?: (date: Date) => void;
    onMonthClick?: (month: number) => void;
    config: AppConfig;
}

export function MiniMonth({ currentDate, onDayClick, onMonthClick, config }: MiniMonthProps): VNode | null {
    const [offset, setOffset] = useState(0);
    const weekStartDay = config.weekStartDay;
    const displayDate = useMemo(
        () => new Date(currentDate.getFullYear(), currentDate.getMonth() + offset, 1),
        [currentDate.getFullYear(), currentDate.getMonth(), offset]
    );
    const year = displayDate.getFullYear();
    const month = displayDate.getMonth();
    const weekdays = useMemo(() => getWeekdays(weekStartDay), [weekStartDay]);
    const monthName = displayDate.toLocaleDateString(undefined, { month: 'long', year: 'numeric' });
    const days = useMemo(() => getCalendarDays(year, month, weekStartDay), [year, month, weekStartDay]);
    const weeks = useMemo(() => {
        let w = [];
        for (let i = 0; i < days.length; i += 7) {
            w.push(days.slice(i, i + 7));
        }
        const lastWeek = w[w.length - 1];
        if (lastWeek.every(d => !d.currentMonth)) {
            w = w.slice(0, -1);
        }
        return w;
    }, [days]);

    return (
        <div class="mini-month">
            <div class="mini-month-header">
                <button class="mini-month-nav" onClick={() => setOffset(o => o - 1)} title="Previous month">&#x25C0;</button>
                <span class="mini-month-title" role="button" tabIndex={0} onClick={() => onMonthClick && onMonthClick(month)} onKeyDown={(ev: KeyboardEvent) => { if (ev.key === 'Enter' || ev.key === ' ') { ev.preventDefault(); onMonthClick && onMonthClick(month); } }}>
                    {monthName}
                </span>
                <button class="mini-month-nav" onClick={() => setOffset(o => o + 1)} title="Next month">&#x25B6;</button>
            </div>
            <div class="mini-month-grid">
                <div class="mini-month-weekday-row">
                    {weekdays.map(d => <div class="mini-month-weekday">{d.charAt(0)}</div>)}
                </div>
                {weeks.map(week => (
                    <div class="mini-month-week-row">
                        {week.map(({ date, currentMonth }) => {
                            const classes = ['mini-month-day',
                                !currentMonth && 'mini-month-day-other',
                                currentMonth && isToday(date) && 'mini-month-day-today',
                            ].filter(Boolean).join(' ');
                            return <div class={classes}
                                onClick={currentMonth && onDayClick ? () => onDayClick(date) : undefined}
                                style={currentMonth ? 'cursor: pointer' : ''}>{date.getDate()}</div>;
                        })}
                    </div>
                ))}
            </div>
        </div>
    );
}
