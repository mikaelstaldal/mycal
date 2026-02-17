import { html } from 'htm/preact';
import { getCalendarDays, getWeekdays, isToday, formatTime } from '../lib/date-utils.js';

export function Calendar({ currentDate, events, onDayClick, onEventClick, config }) {
    const weekStartDay = config.weekStartDay;
    const days = getCalendarDays(currentDate.getFullYear(), currentDate.getMonth(), weekStartDay);
    const weekdays = getWeekdays(weekStartDay);

    function eventsForDay(date) {
        return events.filter(e => {
            const start = new Date(e.start_time);
            const end = new Date(e.end_time);
            const dayStart = new Date(date.getFullYear(), date.getMonth(), date.getDate());
            const dayEnd = new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1);
            return start < dayEnd && end > dayStart;
        });
    }

    return html`
        <div class="calendar">
            <div class="calendar-header">
                ${weekdays.map(d => html`<div class="weekday">${d}</div>`)}
            </div>
            <div class="calendar-grid">
                ${days.map(({ date, currentMonth }) => {
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
                                    <div class="event-chip"
                                         style=${e.color ? `background-color: ${e.color}` : ''}
                                         onClick=${(ev) => { ev.stopPropagation(); onEventClick(e); }}>
                                        ${formatTime(e.start_time, config.clockFormat)} ${e.title}
                                    </div>
                                `)}
                            </div>
                        </div>
                    `;
                })}
            </div>
        </div>
    `;
}
