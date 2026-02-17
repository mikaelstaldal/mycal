import { html, render } from 'htm/preact';
import { useState, useEffect, useCallback } from 'preact/hooks';
import { Nav } from './components/nav.js';
import { Calendar } from './components/calendar.js';
import { EventForm } from './components/event-form.js';
import { listEvents, createEvent, updateEvent, deleteEvent } from './lib/api.js';
import { addMonths, toRFC3339 } from './lib/date-utils.js';

function App() {
    const [currentDate, setCurrentDate] = useState(new Date());
    const [events, setEvents] = useState([]);
    const [showForm, setShowForm] = useState(false);
    const [selectedEvent, setSelectedEvent] = useState(null);
    const [defaultDate, setDefaultDate] = useState(null);

    const loadEvents = useCallback(async () => {
        const year = currentDate.getFullYear();
        const month = currentDate.getMonth();
        // Load from start of previous month's last week to end of next month's first week
        const from = new Date(year, month, -6);
        const to = new Date(year, month + 1, 7);
        try {
            const data = await listEvents(toRFC3339(from), toRFC3339(to));
            setEvents(data);
        } catch (err) {
            console.error('Failed to load events:', err);
        }
    }, [currentDate]);

    useEffect(() => { loadEvents(); }, [loadEvents]);

    function handlePrev() { setCurrentDate(addMonths(currentDate, -1)); }
    function handleNext() { setCurrentDate(addMonths(currentDate, 1)); }
    function handleToday() { setCurrentDate(new Date()); }

    function handleDayClick(date) {
        setSelectedEvent(null);
        setDefaultDate(date);
        setShowForm(true);
    }

    function handleEventClick(event) {
        setSelectedEvent(event);
        setDefaultDate(null);
        setShowForm(true);
    }

    async function handleSave(id, data) {
        if (id) {
            await updateEvent(id, data);
        } else {
            await createEvent(data);
        }
        setShowForm(false);
        setSelectedEvent(null);
        await loadEvents();
    }

    async function handleDelete(id) {
        await deleteEvent(id);
        setShowForm(false);
        setSelectedEvent(null);
        await loadEvents();
    }

    function handleClose() {
        setShowForm(false);
        setSelectedEvent(null);
    }

    return html`
        <div class="app">
            <${Nav} currentDate=${currentDate}
                    onPrev=${handlePrev} onNext=${handleNext} onToday=${handleToday} />
            <${Calendar} currentDate=${currentDate} events=${events}
                         onDayClick=${handleDayClick} onEventClick=${handleEventClick} />
            ${showForm && html`
                <${EventForm} event=${selectedEvent} defaultDate=${defaultDate}
                              onSave=${handleSave} onDelete=${handleDelete} onClose=${handleClose} />
            `}
        </div>
    `;
}

render(html`<${App} />`, document.getElementById('app'));
