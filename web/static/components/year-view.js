import { html } from 'htm/preact';
import { getCalendarDays, getWeekdays, isToday, getISOWeekNumber } from '../lib/date-utils.js';

export function YearView({ currentDate, events, onMonthClick, onWeekClick, onDayClick, config }) {
    const year = currentDate.getFullYear();
    const weekStartDay = config.weekStartDay;
    const weekdays = getWeekdays(weekStartDay);

    function eventsForDay(date) {
        const dayStart = new Date(date.getFullYear(), date.getMonth(), date.getDate());
        const dayEnd = new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1);
        return events.filter(e => {
            if (e.all_day) {
                const startDate = e.start_time.substring(0, 10);
                const endDate = e.end_time.substring(0, 10);
                const pad = n => String(n).padStart(2, '0');
                const dayStr = `${date.getFullYear()}-${pad(date.getMonth()+1)}-${pad(date.getDate())}`;
                return dayStr >= startDate && dayStr < endDate;
            }
            const start = new Date(e.start_time);
            const end = new Date(e.end_time);
            return start < dayEnd && end > dayStart;
        });
    }

    function renderMonth(month) {
        const monthDate = new Date(year, month, 1);
        const monthName = monthDate.toLocaleDateString(undefined, { month: 'long' });
        const days = getCalendarDays(year, month, weekStartDay);

        let weeks = [];
        for (let i = 0; i < days.length; i += 7) {
            weeks.push(days.slice(i, i + 7));
        }
        // Drop the last row if it's entirely other-month padding
        const lastWeek = weeks[weeks.length - 1];
        if (lastWeek.every(d => !d.currentMonth)) {
            weeks = weeks.slice(0, -1);
        }

        return html`
            <div class="year-month">
                <div class="year-month-header" onClick=${() => onMonthClick(month)}>
                    ${monthName}
                </div>
                <div class="year-month-grid">
                    <div class="year-weekday-row">
                        ${weekdays.map(d => html`<div class="year-weekday">${d.charAt(0)}</div>`)}
                    </div>
                    ${weeks.map(week => html`
                        <div class="year-week-row">
                            <div class="year-week-number" onClick=${(ev) => { ev.stopPropagation(); onWeekClick(week[0].date); }}>
                                ${getISOWeekNumber(week[0].date)}
                            </div>
                            ${week.map(({ date, currentMonth }) => {
                                const hasEvents = currentMonth && eventsForDay(date).length > 0;
                                const classes = ['year-day',
                                    !currentMonth && 'year-day-other',
                                    currentMonth && isToday(date) && 'year-day-today',
                                    hasEvents && 'year-day-has-events'
                                ].filter(Boolean).join(' ');
                                return html`<div class=${classes}
                                    style=${currentMonth ? 'cursor: pointer' : ''}
                                    onClick=${currentMonth ? (ev) => { ev.stopPropagation(); onDayClick(date); } : undefined}>${date.getDate()}</div>`;
                            })}
                        </div>
                    `)}
                </div>
            </div>
        `;
    }

    return html`
        <div class="year-view">
            ${Array.from({ length: 12 }, (_, i) => renderMonth(i))}
        </div>
    `;
}
