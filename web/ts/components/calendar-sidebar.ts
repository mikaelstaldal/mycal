import type { VNode } from 'preact';
import { html } from 'htm/preact';
import { useState } from 'preact/hooks';
import { COLORS } from '../lib/colors.js';
import type { CalendarMeta } from '../types/models.js';

interface CalendarSidebarProps {
    calendars: CalendarMeta[];
    selectedCalendarIds: number[] | null;
    onToggleCalendar: (id: number) => void;
    onToggleAll: () => void;
    onEditCalendar: (id: number, data: { name: string; color: string }) => void;
}

export function CalendarSidebar({ calendars, selectedCalendarIds, onToggleCalendar, onToggleAll, onEditCalendar }: CalendarSidebarProps): VNode | null {
    const allSelected = selectedCalendarIds === null;
    const [editingId, setEditingId] = useState<number | null>(null);
    const [editName, setEditName] = useState('');
    const [editColor, setEditColor] = useState('');

    function startEdit(cal: CalendarMeta, e: MouseEvent) {
        e.stopPropagation();
        setEditingId(cal.id);
        setEditName(cal.name);
        setEditColor(cal.color);
    }

    function handleSave(e: MouseEvent) {
        e.stopPropagation();
        if (editName.trim()) {
            onEditCalendar(editingId!, { name: editName.trim(), color: editColor });
        }
        setEditingId(null);
    }

    function handleCancel(e: MouseEvent) {
        e.stopPropagation();
        setEditingId(null);
    }

    function handleKeyDown(e: KeyboardEvent) {
        if (e.key === 'Enter') handleSave(e as any);
        if (e.key === 'Escape') handleCancel(e as any);
    }

    return html`
        <div class="calendar-sidebar">
            <div class="calendar-sidebar-header">Calendars</div>
            <label class="calendar-sidebar-toggle-all" onClick=${(e: MouseEvent) => { e.preventDefault(); onToggleAll(); }}>
                <input type="checkbox" checked=${allSelected} readOnly />
                <span class="calendar-sidebar-name">${allSelected ? 'Deselect all' : 'Select all'}</span>
            </label>
            ${calendars.map(cal => {
                if (editingId === cal.id) {
                    return html`
                        <div key=${cal.id} class="calendar-sidebar-edit" onClick=${(e: MouseEvent) => e.stopPropagation()}>
                            <input type="text" class="calendar-edit-name" value=${editName}
                                   onInput=${(e: Event) => setEditName((e.target as HTMLInputElement).value)}
                                   onKeyDown=${handleKeyDown}
                                   ref=${(el: HTMLInputElement | null) => el && setTimeout(() => el.focus(), 0)} />
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
                           onClick=${(e: MouseEvent) => { e.preventDefault(); onToggleCalendar(cal.id); }}>
                        <input type="checkbox" checked=${isChecked} readOnly />
                        <span class="calendar-dot" style="background: ${cal.color}" />
                        <span class="calendar-sidebar-name" title="${cal.name}">${cal.name}</span>
                        <button class="calendar-edit-trigger" onClick=${(e: MouseEvent) => startEdit(cal, e)} title="Edit calendar">\u270E</button>
                    </label>
                `;
            })}
        </div>
    ` as VNode;
}
