import { html } from 'htm/preact';
import { formatMonthYear, formatWeekRange } from '../lib/date-utils.js';

export function Nav({ currentDate, onPrev, onNext, onToday, viewMode, onViewChange, weekStartDay }) {
    const heading = viewMode === 'week'
        ? formatWeekRange(currentDate, weekStartDay)
        : viewMode === 'year'
        ? String(currentDate.getFullYear())
        : formatMonthYear(currentDate);

    return html`
        <nav class="nav">
            <button onClick=${onToday}>Today</button>
            <button onClick=${onPrev}>\u25C0</button>
            <button onClick=${onNext}>\u25B6</button>
            <div class="view-toggle">
                <button class=${viewMode === 'year' ? 'active' : ''} onClick=${() => onViewChange('year')}>Year</button>
                <button class=${viewMode === 'month' ? 'active' : ''} onClick=${() => onViewChange('month')}>Month</button>
                <button class=${viewMode === 'week' ? 'active' : ''} onClick=${() => onViewChange('week')}>Week</button>
            </div>
            <h1>${heading}</h1>
        </nav>
    `;
}
