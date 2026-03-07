import { html } from 'htm/preact';
import { useState } from 'preact/hooks';
import { getCalendarDays, getWeekdays, isToday } from '../lib/date-utils.js';

export function MiniMonth({ currentDate, onDayClick, onMonthClick, config }) {
    const [offset, setOffset] = useState(0);
    const displayDate = new Date(currentDate.getFullYear(), currentDate.getMonth() + offset, 1);
    const year = displayDate.getFullYear();
    const month = displayDate.getMonth();
    const weekStartDay = config.weekStartDay;
    const weekdays = getWeekdays(weekStartDay);
    const monthName = displayDate.toLocaleDateString(undefined, { month: 'long', year: 'numeric' });
    const days = getCalendarDays(year, month, weekStartDay);

    let weeks = [];
    for (let i = 0; i < days.length; i += 7) {
        weeks.push(days.slice(i, i + 7));
    }
    const lastWeek = weeks[weeks.length - 1];
    if (lastWeek.every(d => !d.currentMonth)) {
        weeks = weeks.slice(0, -1);
    }

    return html`
        <div class="mini-month">
            <div class="mini-month-header">
                <button class="mini-month-nav" onClick=${() => setOffset(o => o - 1)} title="Previous month">\u25C0</button>
                <span class="mini-month-title" onClick=${() => onMonthClick && onMonthClick(month)}>
                    ${monthName}
                </span>
                <button class="mini-month-nav" onClick=${() => setOffset(o => o + 1)} title="Next month">\u25B6</button>
            </div>
            <div class="mini-month-grid">
                <div class="mini-month-weekday-row">
                    ${weekdays.map(d => html`<div class="mini-month-weekday">${d.charAt(0)}</div>`)}
                </div>
                ${weeks.map(week => html`
                    <div class="mini-month-week-row">
                        ${week.map(({ date, currentMonth }) => {
                            const classes = ['mini-month-day',
                                !currentMonth && 'mini-month-day-other',
                                currentMonth && isToday(date) && 'mini-month-day-today',
                            ].filter(Boolean).join(' ');
                            return html`<div class=${classes}
                                onClick=${currentMonth && onDayClick ? () => onDayClick(date) : undefined}
                                style=${currentMonth ? 'cursor: pointer' : ''}>${date.getDate()}</div>`;
                        })}
                    </div>
                `)}
            </div>
        </div>
    `;
}
