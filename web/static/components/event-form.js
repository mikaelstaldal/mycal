import { html } from 'htm/preact';
import { useState, useEffect, useRef } from 'preact/hooks';
import { toLocalDatetimeValue, fromLocalDatetimeValue, formatDate, formatTime, toLocalDateValue, formatDateOnly, exclusiveToInclusiveDate, inclusiveToExclusiveDate } from '../lib/date-utils.js';

const COLORS = ['#4285f4', '#ea4335', '#fbbc04', '#34a853', '#ff6d01', '#46bdc6', '#7baaf7', '#e67c73'];

function formatDatetime(isoStr, config) {
    const d = new Date(isoStr);
    return `${formatDate(d, config.dateFormat)} ${formatTime(isoStr, config.clockFormat)}`;
}

export function EventForm({ event, defaultDate, onSave, onDelete, onClose, config }) {
    const dialogRef = useRef(null);
    const [editing, setEditing] = useState(!event);
    const [title, setTitle] = useState('');
    const [description, setDescription] = useState('');
    const [startTime, setStartTime] = useState('');
    const [endTime, setEndTime] = useState('');
    const [allDay, setAllDay] = useState(false);
    const [color, setColor] = useState('');
    const [error, setError] = useState('');

    useEffect(() => {
        if (event) {
            setTitle(event.title);
            setDescription(event.description);
            setAllDay(event.all_day || false);
            if (event.all_day) {
                setStartTime(toLocalDateValue(event.start_time));
                setEndTime(exclusiveToInclusiveDate(event.end_time));
            } else {
                setStartTime(toLocalDatetimeValue(event.start_time));
                setEndTime(toLocalDatetimeValue(event.end_time));
            }
            setColor(event.color);
            setEditing(false);
        } else if (defaultDate) {
            const start = new Date(defaultDate);
            start.setHours(9, 0, 0, 0);
            const end = new Date(defaultDate);
            end.setHours(10, 0, 0, 0);
            setStartTime(toLocalDatetimeValue(start.toISOString()));
            setEndTime(toLocalDatetimeValue(end.toISOString()));
            setTitle('');
            setDescription('');
            setAllDay(false);
            setColor('');
            setEditing(true);
        }
        setError('');
    }, [event, defaultDate]);

    useEffect(() => {
        const dialog = dialogRef.current;
        if (dialog && !dialog.open) {
            dialog.showModal();
        }
    });

    function handleAllDayToggle(checked) {
        setAllDay(checked);
        if (checked && startTime) {
            // Convert datetime-local to date (inclusive end = same day)
            const dateStr = startTime.substring(0, 10);
            setStartTime(dateStr);
            setEndTime(dateStr);
        } else if (!checked && startTime) {
            // Convert date to datetime-local
            setStartTime(startTime + 'T09:00');
            setEndTime(endTime ? endTime + 'T10:00' : '');
        }
    }

    function handleSubmit(e) {
        e.preventDefault();
        if (!title.trim()) { setError('Title is required'); return; }

        if (allDay) {
            if (!startTime) { setError('Start date is required'); return; }
            const data = {
                title: title.trim(),
                description,
                all_day: true,
                start_time: startTime,
                end_time: inclusiveToExclusiveDate(endTime || startTime),
                color,
            };
            onSave(event?.id, data).catch(err => setError(err.message));
        } else {
            if (!startTime || !endTime) { setError('Start and end times are required'); return; }
            const data = {
                title: title.trim(),
                description,
                all_day: false,
                start_time: fromLocalDatetimeValue(startTime),
                end_time: fromLocalDatetimeValue(endTime),
                color,
            };
            onSave(event?.id, data).catch(err => setError(err.message));
        }
    }

    function handleDelete() {
        if (confirm('Delete this event?')) {
            onDelete(event.id).catch(err => setError(err.message));
        }
    }

    function handleClose(e) {
        e.preventDefault();
        onClose();
    }

    function displayStart() {
        if (!event) return '';
        if (event.all_day) return formatDateOnly(event.start_time, config.dateFormat);
        return formatDatetime(event.start_time, config);
    }

    function displayEnd() {
        if (!event) return '';
        if (event.all_day) return formatDateOnly(exclusiveToInclusiveDate(event.end_time), config.dateFormat);
        return formatDatetime(event.end_time, config);
    }

    return html`
        <dialog ref=${dialogRef} class="event-dialog" onClose=${onClose}>
            <form onSubmit=${handleSubmit}>
                <div class="dialog-header">
                    <h2>${event ? (editing ? 'Edit Event' : 'Event') : 'New Event'}</h2>
                    <button type="button" class="close-btn" onClick=${handleClose}>\u00D7</button>
                </div>

                ${error && html`<div class="error">${error}</div>`}

                <label>
                    Title
                    <input type="text" value=${title} disabled=${!editing}
                           onInput=${e => setTitle(e.target.value)} />
                </label>

                <label>
                    Description
                    <textarea value=${description} disabled=${!editing}
                              onInput=${e => setDescription(e.target.value)} rows="3" />
                </label>

                ${editing && html`
                    <label class="checkbox-label">
                        <input type="checkbox" checked=${allDay}
                               onChange=${e => handleAllDayToggle(e.target.checked)} />
                        All day
                    </label>
                `}

                <label>
                    Start
                    ${editing
                        ? allDay
                            ? html`<input type="date" value=${startTime}
                                          onInput=${e => setStartTime(e.target.value)} />`
                            : html`<input type="datetime-local" value=${startTime}
                                          onInput=${e => setStartTime(e.target.value)} />`
                        : html`<input type="text" disabled value=${displayStart()} />`
                    }
                </label>

                <label>
                    End
                    ${editing
                        ? allDay
                            ? html`<input type="date" value=${endTime}
                                          onInput=${e => setEndTime(e.target.value)} />`
                            : html`<input type="datetime-local" value=${endTime}
                                          onInput=${e => setEndTime(e.target.value)} />`
                        : html`<input type="text" disabled value=${displayEnd()} />`
                    }
                </label>

                ${editing && html`
                    <div class="color-picker">
                        <span>Color</span>
                        <div class="color-options">
                            <div class="color-swatch ${!color ? 'selected' : ''}"
                                 style="background-color: #9e9e9e"
                                 onClick=${() => setColor('')} />
                            ${COLORS.map(c => html`
                                <div class="color-swatch ${color === c ? 'selected' : ''}"
                                     style="background-color: ${c}"
                                     onClick=${() => setColor(c)} />
                            `)}
                        </div>
                    </div>
                `}

                <div class="dialog-actions">
                    ${event && !editing && html`
                        <button type="button" onClick=${() => setEditing(true)}>Edit</button>
                        <button type="button" class="danger" onClick=${handleDelete}>Delete</button>
                    `}
                    ${editing && html`
                        <button type="submit">Save</button>
                    `}
                    <button type="button" onClick=${handleClose}>Cancel</button>
                </div>
            </form>
        </dialog>
    `;
}
