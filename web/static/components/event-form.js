import { html } from 'htm/preact';
import { useState, useEffect, useRef } from 'preact/hooks';
import { toLocalDatetimeValue, fromLocalDatetimeValue, formatDate, formatTime, toLocalDateValue, formatDateOnly, exclusiveToInclusiveDate, inclusiveToExclusiveDate } from '../lib/date-utils.js';
import { MapPicker } from './map-picker.js';
import { RichEditor } from './rich-editor.js';

const COLORS = ['#4285f4', '#ea4335', '#fbbc04', '#34a853', '#ff6d01', '#46bdc6', '#7baaf7', '#e67c73'];
const WEEKDAYS = [
    { key: 'MO', label: 'Mon' },
    { key: 'TU', label: 'Tue' },
    { key: 'WE', label: 'Wed' },
    { key: 'TH', label: 'Thu' },
    { key: 'FR', label: 'Fri' },
    { key: 'SA', label: 'Sat' },
    { key: 'SU', label: 'Sun' },
];

function formatDatetime(isoStr, config) {
    const d = new Date(isoStr);
    return `${formatDate(d, config.dateFormat)} ${formatTime(isoStr, config.clockFormat)}`;
}

function getWeekdayAbbr(date) {
    const days = ['SU', 'MO', 'TU', 'WE', 'TH', 'FR', 'SA'];
    return days[date.getDay()];
}

function getNthWeekdayOfMonth(date) {
    return Math.ceil(date.getDate() / 7);
}

export function EventForm({ event, defaultDate, defaultAllDay, onSave, onDelete, onClose, config }) {
    const dialogRef = useRef(null);
    const titleRef = useRef(null);
    const isInstanceEdit = event && event._editInstance;
    const [editing, setEditing] = useState(!event);
    const [title, setTitle] = useState('');
    const [description, setDescription] = useState('');
    const [startTime, setStartTime] = useState('');
    const [endTime, setEndTime] = useState('');
    const [allDay, setAllDay] = useState(false);
    const [color, setColor] = useState('');
    const [recurrenceFreq, setRecurrenceFreq] = useState('');
    const [recurrenceCount, setRecurrenceCount] = useState(0);
    const [recurrenceUntil, setRecurrenceUntil] = useState('');
    const [recurrenceInterval, setRecurrenceInterval] = useState(1);
    const [recurrenceByDay, setRecurrenceByDay] = useState('');
    const [recurrenceByMonthDay, setRecurrenceByMonthDay] = useState('');
    const [recurrenceByMonth, setRecurrenceByMonth] = useState('');
    const [exdates, setExdates] = useState('');
    const [rdates, setRdates] = useState('');
    const [newRdate, setNewRdate] = useState('');
    const [reminderMinutes, setReminderMinutes] = useState(0);
    const [location, setLocation] = useState('');
    const [latitude, setLatitude] = useState('');
    const [longitude, setLongitude] = useState('');
    const [error, setError] = useState('');
    const [monthlyMode, setMonthlyMode] = useState('bymonthday');
    const [useDuration, setUseDuration] = useState(false);
    const [durationHours, setDurationHours] = useState(1);
    const [durationMinutes, setDurationMinutes] = useState(0);
    const [categories, setCategories] = useState('');
    const [eventURL, setEventURL] = useState('');

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
            setRecurrenceUntil(event.recurrence_until ? event.recurrence_until.substring(0, 10) : '');
            setRecurrenceInterval(event.recurrence_interval > 0 ? event.recurrence_interval : 1);
            setRecurrenceByDay(event.recurrence_by_day || '');
            setRecurrenceByMonthDay(event.recurrence_by_monthday || '');
            setRecurrenceByMonth(event.recurrence_by_month || '');
            setExdates(event.exdates || '');
            setRdates(event.rdates || '');
            setReminderMinutes(event.reminder_minutes || 0);
            setLocation(event.location || '');
            setLatitude(event.latitude != null ? String(event.latitude) : '');
            setLongitude(event.longitude != null ? String(event.longitude) : '');
            setCategories(event.categories || '');
            setEventURL(event.url || '');
            // Duration
            if (event.duration) {
                setUseDuration(true);
                const parsed = parseDurationString(event.duration);
                setDurationHours(parsed.hours);
                setDurationMinutes(parsed.minutes);
            } else {
                setUseDuration(false);
                setDurationHours(1);
                setDurationMinutes(0);
            }
            // Set monthly mode based on existing BY* params
            if (event.recurrence_by_day) {
                setMonthlyMode('byday');
            } else {
                setMonthlyMode('bymonthday');
            }
            setEditing(isInstanceEdit ? true : false);
        } else if (defaultDate) {
            const start = new Date(defaultDate);
            if (defaultAllDay) {
                const pad = n => String(n).padStart(2, '0');
                const dateStr = `${start.getFullYear()}-${pad(start.getMonth()+1)}-${pad(start.getDate())}`;
                setStartTime(dateStr);
                setEndTime(dateStr);
                setAllDay(true);
            } else {
                const hour = start.getHours() || 9;
                start.setHours(hour, 0, 0, 0);
                const end = new Date(defaultDate);
                end.setHours(hour + 1, 0, 0, 0);
                setStartTime(toLocalDatetimeValue(start.toISOString()));
                setEndTime(toLocalDatetimeValue(end.toISOString()));
                setAllDay(false);
            }
            setTitle('');
            setDescription('');
            setColor('');
            setRecurrenceFreq('');
            setRecurrenceCount(0);
            setRecurrenceUntil('');
            setRecurrenceInterval(1);
            setRecurrenceByDay('');
            setRecurrenceByMonthDay('');
            setRecurrenceByMonth('');
            setExdates('');
            setRdates('');
            setReminderMinutes(0);
            setLocation('');
            setLatitude('');
            setLongitude('');
            setMonthlyMode('bymonthday');
            setUseDuration(false);
            setDurationHours(1);
            setDurationMinutes(0);
            setCategories('');
            setEventURL('');
            setEditing(true);
        }
        setError('');
    }, [event, defaultDate]);

    useEffect(() => {
        const dialog = dialogRef.current;
        if (dialog && !dialog.open) {
            dialog.showModal();
            if (!event && titleRef.current) {
                titleRef.current.focus();
            }
        }
    });

    function handleAllDayToggle(checked) {
        setAllDay(checked);
        if (checked && startTime) {
            const dateStr = startTime.substring(0, 10);
            setStartTime(dateStr);
            setEndTime(dateStr);
        } else if (!checked && startTime) {
            setStartTime(startTime + 'T09:00');
            setEndTime(endTime ? endTime + 'T10:00' : '');
        }
    }

    function toggleByDay(dayKey) {
        const current = recurrenceByDay ? recurrenceByDay.split(',') : [];
        const idx = current.indexOf(dayKey);
        if (idx >= 0) {
            current.splice(idx, 1);
        } else {
            current.push(dayKey);
        }
        setRecurrenceByDay(current.filter(Boolean).join(','));
    }

    function getStartDate() {
        const dateStr = startTime ? startTime.substring(0, 10) : '';
        if (!dateStr) return null;
        return new Date(dateStr + 'T12:00:00');
    }

    function buildDurationString() {
        const h = parseInt(durationHours) || 0;
        const m = parseInt(durationMinutes) || 0;
        if (h === 0 && m === 0) return '';
        let s = 'PT';
        if (h > 0) s += h + 'H';
        if (m > 0) s += m + 'M';
        return s;
    }

    function handleSubmit(e) {
        e.preventDefault();
        if (!title.trim()) { setError('Title is required'); return; }

        const locationFields = {
            location,
            latitude: latitude !== '' ? parseFloat(latitude) : null,
            longitude: longitude !== '' ? parseFloat(longitude) : null,
        };

        const extraFields = {};
        if (categories) extraFields.categories = categories;
        if (eventURL) extraFields.url = eventURL;

        // For instance edits, don't include recurrence fields
        const recurrenceFields = isInstanceEdit ? {} : {
            recurrence_freq: recurrenceFreq,
            recurrence_count: recurrenceCount,
            recurrence_until: recurrenceUntil ? recurrenceUntil + 'T00:00:00Z' : '',
            recurrence_interval: recurrenceFreq ? recurrenceInterval : 0,
            recurrence_by_day: recurrenceFreq ? recurrenceByDay : '',
            recurrence_by_monthday: recurrenceFreq ? recurrenceByMonthDay : '',
            recurrence_by_month: recurrenceFreq ? recurrenceByMonth : '',
            exdates: exdates,
            rdates: rdates,
        };

        const instanceStart = isInstanceEdit ? event._instanceStart : undefined;

        if (allDay) {
            if (!startTime) { setError('Start date is required'); return; }
            const data = {
                title: title.trim(),
                description,
                all_day: true,
                start_time: startTime,
                end_time: useDuration ? undefined : inclusiveToExclusiveDate(endTime || startTime),
                color,
                ...recurrenceFields,
                reminder_minutes: 0,
                ...locationFields,
                ...extraFields,
            };
            if (useDuration) {
                data.duration = buildDurationString();
            }
            onSave(event?.id, data, instanceStart).catch(err => setError(err.message));
        } else {
            if (!startTime || (!useDuration && !endTime)) { setError('Start and end times are required'); return; }
            const data = {
                title: title.trim(),
                description,
                all_day: false,
                start_time: fromLocalDatetimeValue(startTime),
                end_time: useDuration ? undefined : fromLocalDatetimeValue(endTime),
                color,
                ...recurrenceFields,
                reminder_minutes: reminderMinutes,
                ...locationFields,
                ...extraFields,
            };
            if (useDuration) {
                data.duration = buildDurationString();
            }
            onSave(event?.id, data, instanceStart).catch(err => setError(err.message));
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

    function handleRestoreExdate(exdate) {
        const remaining = exdates.split(',').filter(d => d.trim() !== exdate).join(',');
        setExdates(remaining);
    }

    function handleAddRdate() {
        if (!newRdate) return;
        const rfc3339 = newRdate + 'T00:00:00Z';
        const current = rdates ? rdates.split(',') : [];
        if (!current.includes(rfc3339)) {
            current.push(rfc3339);
        }
        setRdates(current.filter(Boolean).join(','));
        setNewRdate('');
    }

    function handleRemoveRdate(rdate) {
        const remaining = rdates.split(',').filter(d => d.trim() !== rdate).join(',');
        setRdates(remaining);
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
        let label = freqLabels[recurrenceFreq] || recurrenceFreq;
        if (recurrenceInterval > 1) {
            const units = { DAILY: 'days', WEEKLY: 'weeks', MONTHLY: 'months', YEARLY: 'years' };
            label = `Every ${recurrenceInterval} ${units[recurrenceFreq] || recurrenceFreq}`;
        }
        if (recurrenceByDay) {
            label += ` on ${recurrenceByDay}`;
        }
        if (recurrenceByMonthDay) {
            label += ` on day ${recurrenceByMonthDay}`;
        }
        if (recurrenceByMonth) {
            label += ` in month ${recurrenceByMonth}`;
        }
        if (recurrenceUntil) return `${label}, until ${recurrenceUntil}`;
        return recurrenceCount > 0 ? `${label}, ${recurrenceCount} times` : `${label}, forever`;
    }

    function displayExdates() {
        if (!exdates) return [];
        return exdates.split(',').map(d => d.trim()).filter(Boolean);
    }

    function displayRdates() {
        if (!rdates) return [];
        return rdates.split(',').map(d => d.trim()).filter(Boolean);
    }

    function displayCategories() {
        if (!categories) return [];
        return categories.split(',').map(c => c.trim()).filter(Boolean);
    }

    const startDate = getStartDate();

    return html`
        <dialog ref=${dialogRef} class="event-dialog" onClose=${onClose}>
            <form onSubmit=${handleSubmit}>
                <div class="dialog-header">
                    <h2>${event ? (editing ? (isInstanceEdit ? 'Edit Instance' : 'Edit Event') : 'Event') : 'New Event'}</h2>
                    <button type="button" class="close-btn" onClick=${handleClose}>\u00D7</button>
                </div>

                ${error && html`<div class="error">${error}</div>`}

                <label>
                    Title
                    <input type="text" ref=${titleRef} value=${title} disabled=${!editing}
                           onInput=${e => setTitle(e.target.value)} />
                </label>

                ${editing ? html`
                    <label>
                        Description
                        <${RichEditor} value=${description} onChange=${v => setDescription(v)} />
                    </label>
                ` : description && html`
                    <label>
                        Description
                        <div class="description-display" dangerouslySetInnerHTML=${{ __html: description }} />
                    </label>
                `}

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

                ${editing && html`
                    <label class="checkbox-label">
                        <input type="checkbox" checked=${useDuration}
                               onChange=${e => setUseDuration(e.target.checked)} />
                        Use duration instead of end time
                    </label>
                `}

                ${useDuration && editing ? html`
                    <label>
                        Duration
                        <div class="duration-row">
                            <input type="number" min="0" max="999" value=${durationHours}
                                   style="width: 60px"
                                   onInput=${e => setDurationHours(parseInt(e.target.value) || 0)} />
                            <span>h</span>
                            <input type="number" min="0" max="59" value=${durationMinutes}
                                   style="width: 60px"
                                   onInput=${e => setDurationMinutes(parseInt(e.target.value) || 0)} />
                            <span>m</span>
                        </div>
                    </label>
                ` : html`
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
                `}

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
                    <div class="form-row">
                        <label>
                            Categories
                            <input type="text" value=${categories}
                                   onInput=${e => setCategories(e.target.value)}
                                   placeholder="e.g. Work, Meeting" />
                        </label>
                        <label>
                            URL
                            <input type="url" value=${eventURL}
                                   onInput=${e => setEventURL(e.target.value)}
                                   placeholder="https://example.com" />
                        </label>
                    </div>
                ` : html`
                    ${categories && html`
                        <label>
                            Categories
                            <div class="categories-display">
                                ${displayCategories().map(cat => html`
                                    <span class="category-tag" key=${cat}>${cat}</span>
                                `)}
                            </div>
                        </label>
                    `}
                    ${eventURL && html`
                        <label>
                            URL
                            <a href=${eventURL} target="_blank" rel="noopener noreferrer"
                               style="display: block; color: #4285f4; word-break: break-all;">
                                ${eventURL} \u2197
                            </a>
                        </label>
                    `}
                `}

                ${!isInstanceEdit && editing ? html`
                    <div class="form-row">
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
                        ${!allDay && html`
                            <label>
                                Reminder
                                <select value=${reminderMinutes}
                                        onChange=${e => setReminderMinutes(parseInt(e.target.value) || 0)}>
                                    <option value="0">None</option>
                                    <option value="5">5 min before</option>
                                    <option value="10">10 min before</option>
                                    <option value="15">15 min before</option>
                                    <option value="30">30 min before</option>
                                    <option value="60">1 hour before</option>
                                </select>
                            </label>
                        `}
                    </div>
                    ${recurrenceFreq && html`
                        <label>
                            Every
                            <div class="interval-row">
                                <input type="number" min="1" max="99" value=${recurrenceInterval}
                                       style="width: 60px"
                                       onInput=${e => setRecurrenceInterval(parseInt(e.target.value) || 1)} />
                                <span>${{DAILY:'day(s)',WEEKLY:'week(s)',MONTHLY:'month(s)',YEARLY:'year(s)'}[recurrenceFreq]}</span>
                            </div>
                        </label>
                        ${recurrenceFreq === 'WEEKLY' && html`
                            <div class="byday-picker">
                                <span>On days</span>
                                <div class="byday-buttons">
                                    ${WEEKDAYS.map(wd => html`
                                        <button type="button"
                                                class="byday-btn ${recurrenceByDay.split(',').includes(wd.key) ? 'active' : ''}"
                                                onClick=${() => toggleByDay(wd.key)}>
                                            ${wd.label}
                                        </button>
                                    `)}
                                </div>
                            </div>
                        `}
                        ${recurrenceFreq === 'MONTHLY' && html`
                            <div class="monthly-options">
                                <label class="radio-label">
                                    <input type="radio" name="monthly-mode" value="bymonthday"
                                           checked=${monthlyMode === 'bymonthday'}
                                           onChange=${() => {
                                               setMonthlyMode('bymonthday');
                                               setRecurrenceByDay('');
                                               if (startDate) {
                                                   setRecurrenceByMonthDay(String(startDate.getDate()));
                                               }
                                           }} />
                                    On day ${startDate ? startDate.getDate() : '...'}
                                </label>
                                <label class="radio-label">
                                    <input type="radio" name="monthly-mode" value="byday"
                                           checked=${monthlyMode === 'byday'}
                                           onChange=${() => {
                                               setMonthlyMode('byday');
                                               setRecurrenceByMonthDay('');
                                               if (startDate) {
                                                   const nth = getNthWeekdayOfMonth(startDate);
                                                   const dayAbbr = getWeekdayAbbr(startDate);
                                                   setRecurrenceByDay(`${nth}${dayAbbr}`);
                                               }
                                           }} />
                                    On the ${startDate ? ordinalLabel(getNthWeekdayOfMonth(startDate)) : '...'} ${startDate ? WEEKDAYS.find(w => w.key === getWeekdayAbbr(startDate))?.label : '...'}
                                </label>
                            </div>
                        `}
                        <label>
                            Occurrences (0 = unlimited)
                            <input type="number" min="0" value=${recurrenceCount}
                                   onInput=${e => setRecurrenceCount(parseInt(e.target.value) || 0)} />
                        </label>
                        <label>
                            Until date (optional)
                            <input type="date" value=${recurrenceUntil}
                                   onInput=${e => setRecurrenceUntil(e.target.value)} />
                        </label>
                        ${displayExdates().length > 0 && html`
                            <div class="exdates-section">
                                <span>Excluded dates</span>
                                <div class="exdates-list">
                                    ${displayExdates().map(exd => html`
                                        <div class="exdate-item" key=${exd}>
                                            <span>${new Date(exd).toLocaleDateString()}</span>
                                            <button type="button" class="small-btn" onClick=${() => handleRestoreExdate(exd)}>Restore</button>
                                        </div>
                                    `)}
                                </div>
                            </div>
                        `}
                        ${html`
                            <div class="rdates-section">
                                <span>Additional dates</span>
                                <div class="rdate-add-row">
                                    <input type="date" value=${newRdate}
                                           onInput=${e => setNewRdate(e.target.value)} />
                                    <button type="button" class="small-btn" onClick=${handleAddRdate}>Add</button>
                                </div>
                                ${displayRdates().length > 0 && html`
                                    <div class="rdates-list">
                                        ${displayRdates().map(rd => html`
                                            <div class="rdate-item" key=${rd}>
                                                <span>${new Date(rd).toLocaleDateString()}</span>
                                                <button type="button" class="small-btn danger" onClick=${() => handleRemoveRdate(rd)}>Remove</button>
                                            </div>
                                        `)}
                                    </div>
                                `}
                            </div>
                        `}
                    `}
                ` : !isInstanceEdit && recurrenceFreq && !editing ? html`
                    <label>
                        Repeat
                        <input type="text" disabled value=${displayRecurrence()} />
                    </label>
                    ${displayExdates().length > 0 && html`
                        <label>
                            Excluded dates
                            <input type="text" disabled value=${displayExdates().map(d => new Date(d).toLocaleDateString()).join(', ')} />
                        </label>
                    `}
                    ${displayRdates().length > 0 && html`
                        <label>
                            Additional dates
                            <input type="text" disabled value=${displayRdates().map(d => new Date(d).toLocaleDateString()).join(', ')} />
                        </label>
                    `}
                ` : null}

                ${!allDay && reminderMinutes > 0 && !editing ? html`
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

function parseDurationString(s) {
    let hours = 0, minutes = 0;
    if (!s) return { hours, minutes };
    s = s.toUpperCase();
    const hMatch = s.match(/(\d+)H/);
    const mMatch = s.match(/(\d+)M/);
    const dMatch = s.match(/(\d+)D/);
    if (hMatch) hours = parseInt(hMatch[1]);
    if (mMatch) minutes = parseInt(mMatch[1]);
    if (dMatch) hours += parseInt(dMatch[1]) * 24;
    return { hours, minutes };
}

function ordinalLabel(n) {
    if (n === 1) return '1st';
    if (n === 2) return '2nd';
    if (n === 3) return '3rd';
    return n + 'th';
}
