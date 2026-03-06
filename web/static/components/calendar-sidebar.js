import { html } from 'htm/preact';
import { useState } from 'preact/hooks';
import { COLORS } from '../lib/colors.js';

export function CalendarSidebar({ calendars, selectedCalendarIds, onToggleCalendar, onToggleAll, onEditCalendar }) {
    const allSelected = selectedCalendarIds === null;
    const [editingId, setEditingId] = useState(null);
    const [editName, setEditName] = useState('');
    const [editColor, setEditColor] = useState('');

    function startEdit(cal, e) {
        e.stopPropagation();
        setEditingId(cal.id);
        setEditName(cal.name);
        setEditColor(cal.color);
    }

    function handleSave(e) {
        e.stopPropagation();
        if (editName.trim()) {
            onEditCalendar(editingId, { name: editName.trim(), color: editColor });
        }
        setEditingId(null);
    }

    function handleCancel(e) {
        e.stopPropagation();
        setEditingId(null);
    }

    function handleKeyDown(e) {
        if (e.key === 'Enter') handleSave(e);
        if (e.key === 'Escape') handleCancel(e);
    }

    return html`
        <div class="calendar-sidebar">
            <div class="calendar-sidebar-header">Calendars</div>
            <label class="calendar-sidebar-toggle-all" onClick=${(e) => { e.preventDefault(); onToggleAll(); }}>
                <input type="checkbox" checked=${allSelected} readOnly />
                <span class="calendar-sidebar-name">${allSelected ? 'Deselect all' : 'Select all'}</span>
            </label>
            ${calendars.map(cal => {
                if (editingId === cal.id) {
                    return html`
                        <div key=${cal.id} class="calendar-sidebar-edit" onClick=${(e) => e.stopPropagation()}>
                            <input type="text" class="calendar-edit-name" value=${editName}
                                   onInput=${(e) => setEditName(e.target.value)}
                                   onKeyDown=${handleKeyDown}
                                   ref=${(el) => el && setTimeout(() => el.focus(), 0)} />
                            <div class="calendar-edit-actions">
                                <button class="calendar-edit-btn" onClick=${handleSave} title="Save">\u2713</button>
                                <button class="calendar-edit-btn" onClick=${handleCancel} title="Cancel">\u2717</button>
                            </div>
                            <div class="calendar-edit-colors">
                                ${COLORS.map(c => html`
                                    <div class="calendar-color-swatch ${editColor === c.name ? 'selected' : ''}"
                                         style="background-color: ${c.name}"
                                         title=${c.name}
                                         onClick=${() => setEditColor(c.name)} />
                                `)}
                            </div>
                        </div>
                    `;
                }
                const isChecked = allSelected || (selectedCalendarIds && selectedCalendarIds.includes(cal.id));
                return html`
                    <label key=${cal.id} class="calendar-sidebar-item"
                           onClick=${(e) => { e.preventDefault(); onToggleCalendar(cal.id); }}>
                        <input type="checkbox" checked=${isChecked} readOnly />
                        <span class="calendar-dot" style="background: ${cal.color}" />
                        <span class="calendar-sidebar-name" title="${cal.name}">${cal.name}</span>
                        <button class="calendar-edit-trigger" onClick=${(e) => startEdit(cal, e)} title="Edit calendar">\u270E</button>
                    </label>
                `;
            })}
        </div>
    `;
}
