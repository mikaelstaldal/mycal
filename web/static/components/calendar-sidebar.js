import { html } from 'htm/preact';

export function CalendarSidebar({ calendars, selectedCalendarIds, onToggleCalendar, onToggleAll }) {
    const allSelected = selectedCalendarIds === null;

    return html`
        <div class="calendar-sidebar">
            <div class="calendar-sidebar-header">Calendars</div>
            <label class="calendar-sidebar-item" onClick=${(e) => { e.preventDefault(); onToggleAll(); }}>
                <input type="checkbox" checked=${allSelected} readOnly />
                <span class="calendar-dot" style="background: #888" />
                <span class="calendar-sidebar-name">All</span>
            </label>
            ${calendars.map(cal => {
                const isChecked = allSelected || (selectedCalendarIds && selectedCalendarIds.includes(cal.id));
                return html`
                    <label key=${cal.id} class="calendar-sidebar-item"
                           onClick=${(e) => { e.preventDefault(); onToggleCalendar(cal.id); }}>
                        <input type="checkbox" checked=${isChecked} readOnly />
                        <span class="calendar-dot" style="background: ${cal.color}" />
                        <span class="calendar-sidebar-name">${cal.name}</span>
                    </label>
                `;
            })}
        </div>
    `;
}
