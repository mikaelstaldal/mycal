import type { VNode } from 'preact';
import { html } from 'htm/preact';
import { getCalendarDays, getWeekdays, isToday, formatTime, getISOWeekNumber, isPastEvent } from '../lib/date-utils.js';
import { eventColor } from '../lib/event-utils.js';
import type { CalendarEvent, AppConfig } from '../types/models.js';

interface CalendarProps {
    currentDate: Date;
    events: CalendarEvent[];
    onDayClick: (date: Date) => void;
    onEventClick: (event: CalendarEvent) => void;
    onWeekClick: (date: Date) => void;
    config: AppConfig;
    highlightEventId?: string | null;
}

export function Calendar({ currentDate, events, onDayClick, onEventClick, onWeekClick, config, highlightEventId }: CalendarProps): VNode | null {
    const weekStartDay = config.weekStartDay;
    const days = getCalendarDays(currentDate.getFullYear(), currentDate.getMonth(), weekStartDay);
    const weekdays = getWeekdays(weekStartDay);

    function eventsForDay(date: Date): CalendarEvent[] {
        const dayStart = new Date(date.getFullYear(), date.getMonth(), date.getDate());
        const dayEnd = new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1);
        return events.filter(e => {
            if (e.all_day) {
                // Compare dates only (UTC) to avoid timezone shift issues
                const startDate = e.start_time.substring(0, 10);
                const endDate = e.end_time.substring(0, 10); // exclusive
                const pad = (n: number) => String(n).padStart(2, '0');
                const dayStr = `${date.getFullYear()}-${pad(date.getMonth()+1)}-${pad(date.getDate())}`;
                return dayStr >= startDate && dayStr < endDate;
            }
            const start = new Date(e.start_time);
            const end = new Date(e.end_time);
            return start < dayEnd && end > dayStart;
        });
    }

    // Group days into weeks of 7
    const weeks = [];
    for (let i = 0; i < days.length; i += 7) {
        weeks.push(days.slice(i, i + 7));
    }

    return html`
        <div class="calendar">
            <div class="calendar-header">
                <div class="week-number-header"></div>
                ${weekdays.map(d => html`<div class="weekday">${d}</div>`)}
            </div>
            <div class="calendar-grid">
                ${weeks.map(week => html`
                    <div class="week-number" onClick=${() => onWeekClick(week[0].date)}>week ${getISOWeekNumber(week[0].date)}</div>
                    ${week.map(({ date, currentMonth }) => {
                        const dayEvents = eventsForDay(date);
                        const classes = ['day',
                            !currentMonth && 'other-month',
                            isToday(date) && 'today'
                        ].filter(Boolean).join(' ');

                        return html`
                            <div class=${classes} onClick=${() => onDayClick(date)}>
                                <span class="day-number">${date.getDate()}</span>
                                <div class="day-events">
                                    ${dayEvents.map(e => html`
                                        <div class=${`event-chip${isPastEvent(e) ? ' past-event' : ''}${highlightEventId === e.id + '|' + e.start_time ? ' highlight-event' : ''}`}
                                             key=${e.id}
                                             title=${e.title}
                                             style=${`background-color: ${eventColor(e, config)}`}
                                             onClick=${(ev: MouseEvent) => { ev.stopPropagation(); onEventClick(e); }}>
                                            ${e.all_day ? '' : formatTime(e.start_time) + ' '}${e.title}
                                        </div>
                                    `)}
                                </div>
                            </div>
                        `;
                    })}
                `)}
            </div>
        </div>
    ` as VNode;
}
