import { html } from 'htm/preact';
import { formatMonthYear } from '../lib/date-utils.js';

export function Nav({ currentDate, onPrev, onNext, onToday }) {
    return html`
        <nav class="nav">
            <button onClick=${onToday}>Today</button>
            <button onClick=${onPrev}>\u25C0</button>
            <button onClick=${onNext}>\u25B6</button>
            <h1>${formatMonthYear(currentDate)}</h1>
        </nav>
    `;
}
