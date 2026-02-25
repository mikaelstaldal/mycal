import { html, render } from 'htm/preact';
import { useState, useEffect, useCallback, useRef } from 'preact/hooks';
import { Nav } from './components/nav.js';
import { Calendar } from './components/calendar.js';
import { WeekView } from './components/week-view.js';
import { DayView } from './components/day-view.js';
import { YearView } from './components/year-view.js';
import { EventForm } from './components/event-form.js';
import { ImportForm } from './components/import-form.js';
import { Toast } from './components/toast.js';
import { Settings } from './components/settings.js';
import { listEvents, searchEvents, createEvent, updateEvent, deleteEvent, getEvent } from './lib/api.js';
import { addMonths, addWeeks, startOfWeek, toRFC3339 } from './lib/date-utils.js';
import { getConfig } from './lib/config.js';
import { checkAndNotify, requestPermission } from './lib/notifications.js';

function App() {
    const [currentDate, setCurrentDate] = useState(new Date());
    const [events, setEvents] = useState([]);
    const [showForm, setShowForm] = useState(false);
    const [selectedEvent, setSelectedEvent] = useState(null);
    const [defaultDate, setDefaultDate] = useState(null);
    const [defaultAllDay, setDefaultAllDay] = useState(false);
    const [config, setConfig] = useState(getConfig);
    const [showImport, setShowImport] = useState(false);
    const [toast, setToast] = useState(null);
    const [toastError, setToastError] = useState(false);
    const [viewMode, setViewMode] = useState(() => getConfig().defaultView || 'month');
    const [searchQuery, setSearchQuery] = useState('');
    const [searchResults, setSearchResults] = useState(null);
    const searchTimer = useRef(null);

    const loadEvents = useCallback(async () => {
        let from, to;
        if (viewMode === 'day') {
            from = new Date(currentDate.getFullYear(), currentDate.getMonth(), currentDate.getDate() - 1);
            to = new Date(currentDate.getFullYear(), currentDate.getMonth(), currentDate.getDate() + 2);
        } else if (viewMode === 'week') {
            const weekStart = startOfWeek(currentDate, config.weekStartDay);
            from = new Date(weekStart.getFullYear(), weekStart.getMonth(), weekStart.getDate() - 1);
            to = new Date(weekStart.getFullYear(), weekStart.getMonth(), weekStart.getDate() + 8);
        } else if (viewMode === 'year') {
            const year = currentDate.getFullYear();
            from = new Date(year, 0, -6);
            to = new Date(year + 1, 0, 7);
        } else {
            const year = currentDate.getFullYear();
            const month = currentDate.getMonth();
            from = new Date(year, month, -6);
            to = new Date(year, month + 1, 7);
        }
        try {
            const data = await listEvents(toRFC3339(from), toRFC3339(to));
            setEvents(data);
        } catch (err) {
            console.error('Failed to load events:', err);
        }
    }, [currentDate, viewMode, config.weekStartDay]);

    useEffect(() => { loadEvents(); }, [loadEvents]);

    useEffect(() => {
        checkAndNotify(events);
        const id = setInterval(() => checkAndNotify(events), 30000);
        return () => clearInterval(id);
    }, [events]);

    function handlePrev() {
        if (viewMode === 'day') {
            setCurrentDate(new Date(currentDate.getFullYear(), currentDate.getMonth(), currentDate.getDate() - 1));
        } else if (viewMode === 'week') {
            setCurrentDate(addWeeks(currentDate, -1));
        } else if (viewMode === 'year') {
            setCurrentDate(new Date(currentDate.getFullYear() - 1, currentDate.getMonth(), 1));
        } else {
            setCurrentDate(addMonths(currentDate, -1));
        }
    }

    function handleNext() {
        if (viewMode === 'day') {
            setCurrentDate(new Date(currentDate.getFullYear(), currentDate.getMonth(), currentDate.getDate() + 1));
        } else if (viewMode === 'week') {
            setCurrentDate(addWeeks(currentDate, 1));
        } else if (viewMode === 'year') {
            setCurrentDate(new Date(currentDate.getFullYear() + 1, currentDate.getMonth(), 1));
        } else {
            setCurrentDate(addMonths(currentDate, 1));
        }
    }

    function handleToday() { setCurrentDate(new Date()); }

    function handleViewChange(mode) { setViewMode(mode); }

    function handleDayClick(date) {
        setSelectedEvent(null);
        setDefaultDate(date);
        setDefaultAllDay(false);
        setShowForm(true);
    }

    function handleAllDayClick(date) {
        setSelectedEvent(null);
        setDefaultDate(date);
        setDefaultAllDay(true);
        setShowForm(true);
    }

    async function handleEventClick(event) {
        if (event.recurrence_parent_id) {
            // This is an override instance - edit it directly
            setSelectedEvent(event);
            setDefaultDate(null);
            setShowForm(true);
            return;
        }
        if (event.recurrence_freq && event.recurrence_index > 0) {
            // Recurring instance (not the first one) - ask user what to edit
            const choice = confirm('Edit this instance only?\n\nOK = Edit this instance\nCancel = Edit all instances');
            if (choice) {
                // Edit single instance - pass instance start for override
                try {
                    const parent = await getEvent(event.id);
                    parent._instanceStart = event.start_time;
                    parent._editInstance = true;
                    setSelectedEvent(parent);
                } catch (err) {
                    console.error('Failed to fetch parent event:', err);
                    setSelectedEvent(event);
                }
            } else {
                // Edit all - fetch parent
                try {
                    const parent = await getEvent(event.id);
                    parent._instanceStart = event.start_time;
                    setSelectedEvent(parent);
                } catch (err) {
                    console.error('Failed to fetch parent event:', err);
                    setSelectedEvent(event);
                }
            }
        } else if (event.recurrence_freq && event.recurrence_index === 0) {
            setSelectedEvent(event);
        } else {
            setSelectedEvent(event);
        }
        setDefaultDate(null);
        setShowForm(true);
    }

    async function handleSave(id, data, instanceStart) {
        if (data.reminder_minutes > 0) {
            requestPermission();
        }
        if (id) {
            await updateEvent(id, data, instanceStart);
        } else {
            await createEvent(data);
        }
        setShowForm(false);
        setSelectedEvent(null);
        await loadEvents();
    }

    async function handleDelete(id, instanceStart) {
        await deleteEvent(id, instanceStart);
        setShowForm(false);
        setSelectedEvent(null);
        await loadEvents();
    }

    function handleClose() {
        setShowForm(false);
        setSelectedEvent(null);
    }

    async function handleEventDrag(eventId, startTime, endTime) {
        try {
            await updateEvent(eventId, { start_time: startTime, end_time: endTime });
            await loadEvents();
        } catch (err) {
            console.error('Failed to update event via drag:', err);
        }
    }

    function handleYearMonthClick(month) {
        setCurrentDate(new Date(currentDate.getFullYear(), month, 1));
        setViewMode('month');
    }

    function handleYearWeekClick(date) {
        setCurrentDate(date);
        setViewMode('week');
    }

    function handleYearDayClick(date) {
        setCurrentDate(date);
        setViewMode('day');
    }

    function handleSearchInput(e) {
        const value = e.target.value;
        setSearchQuery(value);
        if (searchTimer.current) clearTimeout(searchTimer.current);
        if (!value.trim()) {
            setSearchResults(null);
            return;
        }
        searchTimer.current = setTimeout(async () => {
            try {
                const results = await searchEvents(value.trim());
                setSearchResults(results);
            } catch (err) {
                console.error('Search failed:', err);
            }
        }, 300);
    }

    function handleClearSearch() {
        setSearchQuery('');
        setSearchResults(null);
    }

    function formatSearchDate(startTime) {
        const d = new Date(startTime);
        return d.toLocaleDateString(undefined, { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric' });
    }

    function formatSearchTime(startTime, endTime) {
        const s = new Date(startTime);
        const e = new Date(endTime);
        return s.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' }) + ' - ' +
               e.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
    }

    return html`
        <div class="app">
            <div class="top-bar">
                <${Nav} currentDate=${currentDate}
                        onPrev=${handlePrev} onNext=${handleNext} onToday=${handleToday}
                        viewMode=${viewMode} onViewChange=${handleViewChange}
                        weekStartDay=${config.weekStartDay} />
                <div class="top-bar-actions">
                    <input type="search" class="search-input" placeholder="Search events..."
                           value=${searchQuery} onInput=${handleSearchInput} />
                    <button class="settings-btn" onClick=${() => setShowImport(true)} title="Import">
                        \u2B07
                    </button>
                    <${Settings} config=${config} onConfigChange=${setConfig} />
                </div>
            </div>
            ${searchResults !== null ? html`
                <div class="search-results">
                    ${searchResults.length === 0 ? html`
                        <div class="search-empty">No events found for "${searchQuery}"</div>
                    ` : searchResults.map(event => html`
                        <div class="search-result-item" key=${event.id}
                             onClick=${() => handleEventClick(event)}>
                            <div class="search-result-title">${event.title}</div>
                            <div class="search-result-date">${formatSearchDate(event.start_time)}</div>
                            <div class="search-result-time">${formatSearchTime(event.start_time, event.end_time)}</div>
                            ${event.description && html`<div class="search-result-desc" dangerouslySetInnerHTML=${{ __html: event.description }} />`}
                        </div>
                    `)}
                </div>
            ` : viewMode === 'year' ? html`
                <${YearView} currentDate=${currentDate} events=${events}
                             onMonthClick=${handleYearMonthClick} onWeekClick=${handleYearWeekClick}
                             onDayClick=${handleYearDayClick} config=${config} />
            ` : viewMode === 'day' ? html`
                <${DayView} currentDate=${currentDate} events=${events}
                            onDayClick=${handleDayClick} onEventClick=${handleEventClick}
                            onAllDayClick=${handleAllDayClick} onEventDrag=${handleEventDrag} config=${config} />
            ` : viewMode === 'week' ? html`
                <${WeekView} currentDate=${currentDate} events=${events}
                             onDayClick=${handleDayClick} onEventClick=${handleEventClick}
                             onAllDayClick=${handleAllDayClick} onEventDrag=${handleEventDrag} config=${config} />
            ` : html`
                <${Calendar} currentDate=${currentDate} events=${events}
                             onDayClick=${handleDayClick} onEventClick=${handleEventClick}
                             onWeekClick=${handleYearWeekClick}
                             config=${config} />
            `}
            ${showForm && html`
                <${EventForm} event=${selectedEvent} defaultDate=${defaultDate}
                              defaultAllDay=${defaultAllDay}
                              onSave=${handleSave} onDelete=${handleDelete} onClose=${handleClose}
                              config=${config} />
            `}
            ${showImport && html`
                <${ImportForm} onImported=${(message, isError) => { setShowImport(false); if (!isError) loadEvents(); setToastError(!!isError); setToast(message); }}
                               onClose=${() => setShowImport(false)} />
            `}
            ${toast && html`<${Toast} message=${toast} isError=${toastError} onDone=${() => setToast(null)} />`}
        </div>
    `;
}

render(html`<${App} />`, document.getElementById('app'));
