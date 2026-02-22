import { html } from 'htm/preact';
import { useState, useEffect, useRef } from 'preact/hooks';
import { toLocalDatetimeValue, fromLocalDatetimeValue, formatDate, formatTime, toLocalDateValue, formatDateOnly, exclusiveToInclusiveDate, inclusiveToExclusiveDate } from '../lib/date-utils.js';
import { MapPicker } from './map-picker.js';

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
    const [recurrenceFreq, setRecurrenceFreq] = useState('');
    const [recurrenceCount, setRecurrenceCount] = useState(0);
    const [reminderMinutes, setReminderMinutes] = useState(0);
    const [location, setLocation] = useState('');
    const [latitude, setLatitude] = useState('');
    const [longitude, setLongitude] = useState('');
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
            setRecurrenceFreq(event.recurrence_freq || '');
            setRecurrenceCount(event.recurrence_count || 0);
            setReminderMinutes(event.reminder_minutes || 0);
            setLocation(event.location || '');
            setLatitude(event.latitude != null ? String(event.latitude) : '');
            setLongitude(event.longitude != null ? String(event.longitude) : '');
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
            setRecurrenceFreq('');
            setRecurrenceCount(0);
            setReminderMinutes(0);
            setLocation('');
            setLatitude('');
            setLongitude('');
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

        const locationFields = {
            location,
            latitude: latitude !== '' ? parseFloat(latitude) : null,
            longitude: longitude !== '' ? parseFloat(longitude) : null,
        };

        if (allDay) {
            if (!startTime) { setError('Start date is required'); return; }
            const data = {
                title: title.trim(),
                description,
                all_day: true,
                start_time: startTime,
                end_time: inclusiveToExclusiveDate(endTime || startTime),
                color,
                recurrence_freq: recurrenceFreq,
                recurrence_count: recurrenceCount,
                reminder_minutes: 0,
                ...locationFields,
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
                recurrence_freq: recurrenceFreq,
                recurrence_count: recurrenceCount,
                reminder_minutes: reminderMinutes,
                ...locationFields,
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

    function displayReminder() {
        const labels = { 0: 'None', 5: '5 minutes', 10: '10 minutes', 15: '15 minutes', 30: '30 minutes', 60: '1 hour' };
        return labels[reminderMinutes] || `${reminderMinutes} minutes`;
    }

    function displayRecurrence() {
        if (!recurrenceFreq) return 'None';
        const freqLabels = { DAILY: 'Daily', WEEKLY: 'Weekly', MONTHLY: 'Monthly', YEARLY: 'Yearly' };
        const label = freqLabels[recurrenceFreq] || recurrenceFreq;
        return recurrenceCount > 0 ? `${label}, ${recurrenceCount} times` : `${label}, forever`;
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

                ${editing ? html`
                    ${!(config.mapProvider === 'openstreetmap' || (config.mapProvider === 'google' && /^AIza[A-Za-z0-9_-]{35}$/.test(config.googleMapsApiKey))) && html`
                        <label>
                            Location
                            <input type="text" value=${location}
                                   onInput=${e => setLocation(e.target.value)}
                                   placeholder="e.g. Conference Room A" />
                        </label>
                    `}
                    ${!(config.mapProvider === 'openstreetmap' || (config.mapProvider === 'google' && /^AIza[A-Za-z0-9_-]{35}$/.test(config.googleMapsApiKey))) && html`
                        <div class="coord-row">
                            <label>
                                Latitude
                                <input type="number" step="any" min="-90" max="90"
                                       value=${latitude}
                                       onInput=${e => setLatitude(e.target.value)}
                                       placeholder="e.g. 59.3293" />
                            </label>
                            <label>
                                Longitude
                                <input type="number" step="any" min="-180" max="180"
                                       value=${longitude}
                                       onInput=${e => setLongitude(e.target.value)}
                                       placeholder="e.g. 18.0686" />
                            </label>
                        </div>
                    `}
                    <${MapPicker}
                        mapProvider=${config.mapProvider}
                        apiKey=${config.googleMapsApiKey}
                        latitude=${latitude}
                        longitude=${longitude}
                        editing=${true}
                        onCoordinateChange=${(lat, lng) => { setLatitude(lat); setLongitude(lng); }}
                    />
                ` : html`
                    ${location && html`
                        <label>
                            Location
                            <input type="text" disabled value=${location} />
                        </label>
                    `}
                    ${(latitude !== '' && longitude !== '') ? html`
                        <${MapPicker}
                            mapProvider=${config.mapProvider}
                            apiKey=${config.googleMapsApiKey}
                            latitude=${latitude}
                            longitude=${longitude}
                            editing=${false}
                        />
                        ${config.mapProvider !== 'google' && html`
                            <a href=${config.mapProvider === 'openstreetmap'
                                    ? `https://www.openstreetmap.org/?mlat=${latitude}&mlon=${longitude}#map=15/${latitude}/${longitude}`
                                    : `https://www.google.com/maps?q=${latitude},${longitude}`}
                               target="_blank" rel="noopener noreferrer"
                               style="display: inline-block; margin: 4px 0 8px; color: #4285f4;">
                                ${config.mapProvider === 'openstreetmap' ? 'View on OpenStreetMap' : 'View on Google Maps'} \u2197
                            </a>
                        `}
                    ` : location && html`
                        <a href=${config.mapProvider === 'openstreetmap'
                                ? `https://www.openstreetmap.org/search?query=${encodeURIComponent(location)}`
                                : `https://www.google.com/maps/search/${encodeURIComponent(location)}`}
                           target="_blank" rel="noopener noreferrer"
                           style="display: inline-block; margin: 4px 0 8px; color: #4285f4;">
                            ${config.mapProvider === 'openstreetmap' ? 'Search on OpenStreetMap' : 'Search on Google Maps'} \u2197
                        </a>
                    `}
                `}

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

                ${editing ? html`
                    <label>
                        Repeat
                        <select value=${recurrenceFreq}
                                onChange=${e => setRecurrenceFreq(e.target.value)}>
                            <option value="">None</option>
                            <option value="DAILY">Daily</option>
                            <option value="WEEKLY">Weekly</option>
                            <option value="MONTHLY">Monthly</option>
                            <option value="YEARLY">Yearly</option>
                        </select>
                    </label>
                    ${recurrenceFreq && html`
                        <label>
                            Occurrences (0 = unlimited)
                            <input type="number" min="0" value=${recurrenceCount}
                                   onInput=${e => setRecurrenceCount(parseInt(e.target.value) || 0)} />
                        </label>
                    `}
                ` : recurrenceFreq && html`
                    <label>
                        Repeat
                        <input type="text" disabled value=${displayRecurrence()} />
                    </label>
                `}

                ${!allDay && editing ? html`
                    <label>
                        Reminder
                        <select value=${reminderMinutes}
                                onChange=${e => setReminderMinutes(parseInt(e.target.value) || 0)}>
                            <option value="0">None</option>
                            <option value="5">5 minutes before</option>
                            <option value="10">10 minutes before</option>
                            <option value="15">15 minutes before</option>
                            <option value="30">30 minutes before</option>
                            <option value="60">1 hour before</option>
                        </select>
                    </label>
                ` : !allDay && reminderMinutes > 0 && !editing ? html`
                    <label>
                        Reminder
                        <input type="text" disabled value=${displayReminder()} />
                    </label>
                ` : null}

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
