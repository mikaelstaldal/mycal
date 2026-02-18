import { html } from 'htm/preact';
import { getWeekDays, isToday, formatHour, formatTime } from '../lib/date-utils.js';

const HOURS = Array.from({ length: 24 }, (_, i) => i);

export function WeekView({ currentDate, events, onDayClick, onEventClick, config }) {
    const weekStartDay = config.weekStartDay;
    const days = getWeekDays(currentDate, weekStartDay);

    function eventsForDay(date) {
        return events.filter(e => {
            const start = new Date(e.start_time);
            const end = new Date(e.end_time);
            const dayStart = new Date(date.getFullYear(), date.getMonth(), date.getDate());
            const dayEnd = new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1);
            return start < dayEnd && end > dayStart;
        });
    }

    function eventStyle(event, date) {
        const start = new Date(event.start_time);
        const end = new Date(event.end_time);
        const dayStart = new Date(date.getFullYear(), date.getMonth(), date.getDate());
        const dayEnd = new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1);

        const effectiveStart = start < dayStart ? dayStart : start;
        const effectiveEnd = end > dayEnd ? dayEnd : end;

        const startMinutes = (effectiveStart - dayStart) / 60000;
        const endMinutes = (effectiveEnd - dayStart) / 60000;
        const duration = endMinutes - startMinutes;

        const top = (startMinutes / 60) * 48;
        const height = Math.max((duration / 60) * 48, 18);

        return {
            top: `${top}px`,
            height: `${height}px`,
            backgroundColor: event.color || '#4285f4'
        };
    }

    const dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

    return html`
        <div class="week-view">
            <div class="week-header">
                <div class="time-gutter-header"></div>
                ${days.map(date => {
                    const classes = ['week-day-header', isToday(date) && 'today'].filter(Boolean).join(' ');
                    return html`
                        <div class=${classes}>
                            <span class="week-day-name">${dayNames[date.getDay()]}</span>
                            <span class="week-day-number">${date.getDate()}</span>
                        </div>
                    `;
                })}
            </div>
            <div class="week-body">
                <div class="week-grid">
                    ${HOURS.map(hour => html`
                        <div class="time-gutter">${formatHour(hour, config.clockFormat)}</div>
                        ${days.map((date, colIndex) => html`
                            <div class="hour-cell"
                                 onClick=${() => {
                                     const d = new Date(date.getFullYear(), date.getMonth(), date.getDate(), hour);
                                     onDayClick(d);
                                 }}>
                            </div>
                        `)}
                    `)}
                </div>
                <div class="week-events-overlay">
                    <div class="week-events-gutter-spacer"></div>
                    ${days.map((date, colIndex) => {
                        const dayEvents = eventsForDay(date);
                        return html`
                            <div class="week-day-events">
                                ${dayEvents.map(e => html`
                                    <div class="week-event"
                                         style=${eventStyle(e, date)}
                                         onClick=${(ev) => { ev.stopPropagation(); onEventClick(e); }}>
                                        <span class="week-event-title">${e.title}</span>
                                        <span class="week-event-time">${formatTime(e.start_time, config.clockFormat)}</span>
                                    </div>
                                `)}
                            </div>
                        `;
                    })}
                </div>
            </div>
        </div>
    `;
}
