import { render } from 'preact';
import { useState, useEffect, useCallback, useRef } from 'preact/hooks';
import { Nav } from './layout/Nav.js';
import { Calendar } from './views/Calendar.js';
import { WeekView } from './views/WeekView.js';
import { DayView } from './views/DayView.js';
import { ScheduleView } from './views/ScheduleView.js';
import { YearView } from './views/YearView.js';
import { EventForm } from './components/EventForm.js';
import { ImportSingleForm, ImportBulkForm } from './components/ImportForm.js';
import { FeedsDialog } from './components/FeedsDialog.js';
import { Toast } from './components/Toast.js';
import { Settings } from './components/Settings.js';
import { CalendarSidebar } from './layout/CalendarSidebar.js';
import { MiniMonth } from './layout/MiniMonth.js';
import { api } from './api/client.js';
import { showToast } from './util/toast.js';
import { addMonths, addWeeks, startOfWeek, toRFC3339, eventStartStr } from './util/date-utils.js';
import { getConfig, hasUserDefaultView } from './util/config.js';
import { checkAndNotify, requestPermission } from './util/notifications.js';
import { showChoice } from './util/confirm.js';
import type { components } from './api/types.js';
import type { AppConfig } from './util/config.js';
type CalendarEvent = components['schemas']['Event'];
type CalendarMeta = components['schemas']['Calendar'];

declare global {
    interface Window {
        __serverConfig?: { mymailUrl?: string };
    }
}

function App() {
    const [darkMode, setDarkMode] = useState(() => localStorage.getItem('darkMode') === 'true');
    const [currentDate, setCurrentDate] = useState(new Date());
    const [events, setEvents] = useState<CalendarEvent[]>([]);
    const [showForm, setShowForm] = useState(false);
    const [selectedEvent, setSelectedEvent] = useState<(CalendarEvent & { _editInstance?: boolean }) | null>(null);
    const [copiedEvent, setCopiedEvent] = useState<CalendarEvent | null>(null);
    const [defaultDate, setDefaultDate] = useState<Date | null>(null);
    const [defaultAllDay, setDefaultAllDay] = useState(false);
    const [config, setConfig] = useState<AppConfig>(getConfig);
    const [showImportSingle, setShowImportSingle] = useState(false);
    const [showImportBulk, setShowImportBulk] = useState(false);
    const [showFeeds, setShowFeeds] = useState(false);
    const [viewMode, setViewMode] = useState<string>(() => {
        if (hasUserDefaultView()) return getConfig().defaultView;
        return window.innerWidth <= 600 ? 'schedule' : 'week';
    });
    const [searchQuery, setSearchQuery] = useState('');
    const [searchResults, setSearchResults] = useState<CalendarEvent[] | null>(null);
    const searchTimer = useRef<number | null>(null);
    const searchGeneration = useRef(0);
    const preSearchViewMode = useRef<string | null>(null);
    const [highlightEventId, setHighlightEventId] = useState<string | null>(null);
    const [isDragging, setIsDragging] = useState(false);
    const dragCounter = useRef(0);
    const [calendars, setCalendars] = useState<CalendarMeta[]>([]);
    const [selectedCalendarIds, setSelectedCalendarIds] = useState<number[] | null>(null);
    const [scheduleDaysLoaded, setScheduleDaysLoaded] = useState(30);
    const [loadingMoreSchedule, setLoadingMoreSchedule] = useState(false);

    useEffect(() => {
        document.documentElement.setAttribute('data-theme', darkMode ? 'dark' : 'light');
        localStorage.setItem('darkMode', String(darkMode));
    }, [darkMode]);

    const loadCalendars = useCallback(async () => {
        try {
            const cals = await api.calendars.list();
            setCalendars(cals);
            const defaultCal = cals.find(c => c.id === 0);
            const calColors: Record<number, string> = {};
            for (const c of cals) { calColors[c.id] = c.color; }
            setConfig(prev => ({
                ...prev,
                defaultEventColor: defaultCal ? defaultCal.color : 'dodgerblue',
                calendarColors: calColors
            }));
        } catch (err) {
            showToast('Failed to load calendars', { error: true });
        }
    }, []);

    useEffect(() => { loadCalendars(); }, [loadCalendars]);

    const loadEvents = useCallback(async () => {
        let from: Date, to: Date;
        if (viewMode === 'schedule') {
            const today = new Date();
            from = new Date(today.getFullYear(), today.getMonth(), today.getDate());
            to = new Date(today.getFullYear(), today.getMonth(), today.getDate() + scheduleDaysLoaded);
        } else if (viewMode === 'day') {
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
            const data = await api.events.list(toRFC3339(from), toRFC3339(to));
            setEvents(data);
        } catch (err) {
            showToast('Failed to load events', { error: true });
        }
    }, [currentDate, viewMode, config.weekStartDay, scheduleDaysLoaded]);

    useEffect(() => { loadEvents(); }, [loadEvents]);

    useEffect(() => {
        if (viewMode === 'schedule') setScheduleDaysLoaded(30);
    }, [viewMode]);

    useEffect(() => {
        if (!highlightEventId) return;
        const timer = setTimeout(() => setHighlightEventId(null), 2000);
        return () => clearTimeout(timer);
    }, [highlightEventId]);

    const loadMoreScheduleEvents = useCallback(async () => {
        if (loadingMoreSchedule) return;
        setLoadingMoreSchedule(true);
        const today = new Date();
        const from = new Date(today.getFullYear(), today.getMonth(), today.getDate() + scheduleDaysLoaded);
        const newDays = scheduleDaysLoaded + 30;
        const to = new Date(today.getFullYear(), today.getMonth(), today.getDate() + newDays);
        try {
            const data = await api.events.list(toRFC3339(from), toRFC3339(to));
            setEvents(prev => [...prev, ...data]);
            setScheduleDaysLoaded(newDays);
        } catch (err) {
            showToast('Failed to load more events', { error: true });
        } finally {
            setLoadingMoreSchedule(false);
        }
    }, [scheduleDaysLoaded, loadingMoreSchedule]);

    useEffect(() => {
        checkAndNotify(events);
        const id = setInterval(() => checkAndNotify(events), 30000);
        return () => clearInterval(id);
    }, [events]);

    function handleToggleCalendar(calId: number) {
        setSelectedCalendarIds(prev => {
            if (prev === null) {
                const allIds = calendars.map(c => c.id);
                return allIds.filter(id => id !== calId);
            }
            if (prev.includes(calId)) {
                const next = prev.filter(id => id !== calId);
                return next.length === 0 ? [] : next;
            }
            const next = [...prev, calId];
            if (next.length === calendars.length) return null;
            return next;
        });
    }

    function handleToggleAll() {
        setSelectedCalendarIds(prev => prev === null ? [] : null);
    }

    async function handleEditCalendar(id: number, data: { name: string; color: string }) {
        try {
            await api.calendars.update(id, data);
            await loadCalendars();
            await loadEvents();
        } catch (err) {
            console.error('Failed to update calendar:', err);
        }
    }

    function handlePrev() {
        if (viewMode === 'schedule') {
            setCurrentDate(new Date(currentDate.getFullYear(), currentDate.getMonth(), currentDate.getDate() - 7));
        } else if (viewMode === 'day') {
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
        if (viewMode === 'schedule') {
            setCurrentDate(new Date(currentDate.getFullYear(), currentDate.getMonth(), currentDate.getDate() + 7));
        } else if (viewMode === 'day') {
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

    function handleViewChange(mode: string) { setViewMode(mode); }

    function handleDayClick(date: Date) {
        setSelectedEvent(null);
        setDefaultDate(date);
        setDefaultAllDay(false);
        setShowForm(true);
    }

    function handleAllDayClick(date: Date) {
        setSelectedEvent(null);
        setDefaultDate(date);
        setDefaultAllDay(true);
        setShowForm(true);
    }

    async function handleEventClick(event: CalendarEvent) {
        if (event.recurrence_parent_id) {
            setSelectedEvent(event);
            setDefaultDate(null);
            setShowForm(true);
            return;
        }
        if (event.recurrence_freq && event.parent_id) {
            const parentId = event.parent_id;
            const choice = await showChoice('What would you like to edit?', {
                title: 'Edit Recurring Event',
                choices: [
                    { label: 'All instances', value: 'all' },
                    { label: 'This instance', value: 'instance', primary: true },
                ]
            });
            if (choice === null) return;
            if (choice === 'instance') {
                (event as any)._editInstance = true;
                setSelectedEvent(event);
            } else {
                try {
                    const parent = await api.events.get(parentId);
                    setSelectedEvent(parent);
                } catch (err) {
                    console.error('Failed to fetch parent event:', err);
                    setSelectedEvent(event);
                }
            }
        } else {
            setSelectedEvent(event);
        }
        setDefaultDate(null);
        setShowForm(true);
    }

    async function handleSave(id: string | null | undefined, data: any) {
        if (data.reminder_minutes > 0) {
            requestPermission();
        }
        if (id) {
            await api.events.update(id, data);
        } else {
            await api.events.create(data);
        }
        setShowForm(false);
        setSelectedEvent(null);
        setCopiedEvent(null);
        await loadEvents();
    }

    async function handleDelete(id: string) {
        await api.events.delete(id);
        setShowForm(false);
        setSelectedEvent(null);
        await loadEvents();
    }

    function handleCopy() {
        const src = selectedEvent;
        setShowForm(false);
        setSelectedEvent(null);
        setCopiedEvent(src);
        setDefaultDate(null);
        setShowForm(true);
    }

    function handleClose() {
        setShowForm(false);
        setSelectedEvent(null);
        setCopiedEvent(null);
    }

    async function handleEventDrag(eventId: string, startTime: string, endTime: string) {
        try {
            // YYYY-MM-DD strings (no 'T') are all-day dates; ISO strings are datetimes
            const isDateOnly = !startTime.includes('T');
            const update = isDateOnly
                ? { start_date: startTime, end_date: endTime }
                : { start_time: startTime, end_time: endTime };
            await api.events.update(eventId, update);
            await loadEvents();
        } catch (err) {
            showToast('Failed to update event', { error: true });
        }
    }

    function handleYearMonthClick(month: number) {
        setCurrentDate(new Date(currentDate.getFullYear(), month, 1));
        setViewMode('month');
    }

    function handleYearWeekClick(date: Date) {
        setCurrentDate(date);
        setViewMode('week');
    }

    function handleYearDayClick(date: Date) {
        setCurrentDate(date);
        setViewMode('day');
    }

    function clearSearch() {
        setSearchQuery('');
        setSearchResults(null);
        if (searchTimer.current) clearTimeout(searchTimer.current);
    }

    function handleSearchResultClick(event: CalendarEvent) {
        setCurrentDate(new Date(eventStartStr(event)));
        setViewMode(preSearchViewMode.current || viewMode);
        setHighlightEventId(event.id + '|' + eventStartStr(event));
        clearSearch();
        setSelectedEvent(event);
        setDefaultDate(null);
        setShowForm(true);
        setTimeout(() => {
            document.querySelector('.highlight-event')?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
        }, 100);
    }

    function handleSearchInput(e: Event) {
        const value = (e.target as HTMLInputElement).value;
        setSearchQuery(value);
        if (searchTimer.current) clearTimeout(searchTimer.current);
        if (!value.trim()) {
            setSearchResults(null);
            return;
        }
        if (!searchQuery.trim()) {
            preSearchViewMode.current = viewMode;
        }
        const generation = ++searchGeneration.current;
        searchTimer.current = setTimeout(async () => {
            try {
                const results = await api.events.search(value.trim());
                if (generation === searchGeneration.current) {
                    setSearchResults(results);
                }
            } catch (err) {
                if (generation === searchGeneration.current) {
                    console.error('Search failed:', err);
                }
            }
        }, 300) as unknown as number;
    }

    function handleDragOver(e: DragEvent) {
        e.preventDefault();
        e.dataTransfer!.dropEffect = 'copy';
    }

    function handleDragEnter(e: DragEvent) {
        e.preventDefault();
        dragCounter.current++;
        if (dragCounter.current === 1) setIsDragging(true);
    }

    function handleDragLeave(e: DragEvent) {
        e.preventDefault();
        dragCounter.current--;
        if (dragCounter.current === 0) setIsDragging(false);
    }

    async function handleDrop(e: DragEvent) {
        e.preventDefault();
        dragCounter.current = 0;
        setIsDragging(false);
        const file = e.dataTransfer!.files[0];
        if (!file || !file.name.endsWith('.ics')) return;
        try {
            const text = await file.text();
            await api.import.single(text);
            showToast('Event imported successfully');
            await loadEvents();
        } catch (err: any) {
            showToast(err.message || 'Import failed', { error: true });
        }
    }

    function formatSearchDate(startTime: string) {
        const d = new Date(startTime);
        return d.toLocaleDateString(undefined, { weekday: 'short', year: 'numeric', month: 'short', day: 'numeric' });
    }

    function formatSearchTime(startTime: string, endTime: string) {
        const s = new Date(startTime);
        const e = new Date(endTime);
        return s.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' }) + ' - ' +
               e.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
    }

    const visibleEvents = selectedCalendarIds === null
        ? events
        : events.filter(e => selectedCalendarIds.includes(e.calendar_id ?? 0));

    return (
        <div class={`app${isDragging ? ' drag-over' : ''}`}
             onDragOver={handleDragOver} onDragEnter={handleDragEnter}
             onDragLeave={handleDragLeave} onDrop={handleDrop}>
            <header class="top-bar">
                <Nav currentDate={currentDate}
                     onPrev={handlePrev} onNext={handleNext} onToday={handleToday}
                     viewMode={viewMode} onViewChange={handleViewChange}
                     weekStartDay={config.weekStartDay} />
                <div class="top-bar-actions">
                    <input type="search" class="search-input" placeholder="Search events..."
                           value={searchQuery} onInput={handleSearchInput} />
                    <button class="dark-mode-btn" onClick={() => setDarkMode(d => !d)} title={darkMode ? 'Switch to light mode' : 'Switch to dark mode'} aria-label={darkMode ? 'Switch to light mode' : 'Switch to dark mode'}>
                        {darkMode ? '☀︎' : '☾︎'}
                    </button>
                    <button class="settings-btn" onClick={() => { loadEvents(); loadCalendars(); }} title="Refresh" aria-label="Refresh">
                        ↻
                    </button>
                    <button class="settings-btn" onClick={() => setShowImportSingle(true)} title="Import Event" aria-label="Import Event">
                        ⬇︎
                    </button>
                    <button class="settings-btn" onClick={() => setShowImportBulk(true)} title="Bulk Import" aria-label="Bulk Import">
                        ⇊︎
                    </button>
                    <button class="settings-btn" onClick={() => setShowFeeds(true)} title="Feed Subscriptions" aria-label="Feed Subscriptions">
                        🔗︎
                    </button>
                    <Settings config={config} onConfigChange={setConfig} />
                </div>
            </header>
            <div class="app-layout">
                <div class="left-sidebar">
                    <MiniMonth currentDate={currentDate}
                               onDayClick={handleYearDayClick}
                               onMonthClick={handleYearMonthClick}
                               config={config} />
                    {calendars.length > 1 ? (
                        <CalendarSidebar calendars={calendars}
                                         selectedCalendarIds={selectedCalendarIds}
                                         onToggleCalendar={handleToggleCalendar}
                                         onToggleAll={handleToggleAll}
                                         onEditCalendar={handleEditCalendar} />
                    ) : null}
                </div>
                <main class="app-main">
                    {searchResults !== null ? (
                        <div class="search-results">
                            <div class="search-results-header">
                                <span>Search results for "{searchQuery}"</span>
                                <button class="search-clear-btn" onClick={clearSearch} title="Clear search">&#x2715;</button>
                            </div>
                            {searchResults.length === 0 ? (
                                <div class="search-empty">No events found</div>
                            ) : searchResults.map(event => (
                                <div class={`search-result-item${new Date(eventStartStr(event)) < new Date() ? ' search-result-past' : ''}`} key={event.id}
                                     onClick={() => handleSearchResultClick(event)}>
                                    <div class="search-result-title">{event.title}</div>
                                    <div class="search-result-date">{formatSearchDate(eventStartStr(event))}</div>
                                    <div class="search-result-time">{event.all_day ? '' : formatSearchTime(event.start_time!, event.end_time!)}</div>
                                    {event.description && <div class="search-result-desc" dangerouslySetInnerHTML={{ __html: event.description }} />}
                                </div>
                            ))}
                        </div>
                    ) : viewMode === 'year' ? (
                        <YearView currentDate={currentDate} events={visibleEvents}
                                  onMonthClick={handleYearMonthClick} onWeekClick={handleYearWeekClick}
                                  onDayClick={handleYearDayClick} config={config}
                                  highlightEventId={highlightEventId} />
                    ) : viewMode === 'schedule' ? (
                        <ScheduleView currentDate={currentDate} events={visibleEvents}
                                      onEventClick={handleEventClick} onDayClick={handleDayClick} config={config}
                                      onLoadMore={loadMoreScheduleEvents} daysLoaded={scheduleDaysLoaded}
                                      highlightEventId={highlightEventId} />
                    ) : viewMode === 'day' ? (
                        <DayView currentDate={currentDate} events={visibleEvents}
                                 onDayClick={handleDayClick} onEventClick={handleEventClick}
                                 onAllDayClick={handleAllDayClick} onEventDrag={handleEventDrag} config={config}
                                 highlightEventId={highlightEventId} />
                    ) : viewMode === 'week' ? (
                        <WeekView currentDate={currentDate} events={visibleEvents}
                                  onDayClick={handleDayClick} onEventClick={handleEventClick}
                                  onAllDayClick={handleAllDayClick} onEventDrag={handleEventDrag} config={config}
                                  highlightEventId={highlightEventId} />
                    ) : (
                        <Calendar currentDate={currentDate} events={visibleEvents}
                                  onDayClick={handleDayClick} onEventClick={handleEventClick}
                                  onWeekClick={handleYearWeekClick}
                                  config={config}
                                  highlightEventId={highlightEventId} />
                    )}
                </main>
            </div>
            {showForm && (
                <EventForm event={selectedEvent} defaultDate={defaultDate}
                           defaultAllDay={defaultAllDay}
                           copiedEvent={copiedEvent}
                           onSave={handleSave} onDelete={handleDelete} onClose={handleClose}
                           onCopy={selectedEvent ? handleCopy : undefined}
                           config={config} mymailUrl={config.mymailUrl || window.__serverConfig?.mymailUrl || ''} />
            )}
            {showImportSingle && (
                <ImportSingleForm onImported={() => { setShowImportSingle(false); loadEvents(); }}
                                  onClose={() => setShowImportSingle(false)} />
            )}
            {showImportBulk && (
                <ImportBulkForm onImported={() => { setShowImportBulk(false); loadEvents(); }}
                                onClose={() => setShowImportBulk(false)} />
            )}
            {showFeeds && (
                <FeedsDialog onClose={() => setShowFeeds(false)}
                             onRefreshed={() => { loadEvents(); loadCalendars(); }} />
            )}
            <Toast />
        </div>
    );
}

render(<App />, document.getElementById('app')!);
