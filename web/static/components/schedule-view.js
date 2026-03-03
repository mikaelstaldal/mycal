import { html } from 'htm/preact';
import { useEffect, useRef } from 'preact/hooks';
import { formatTime, isPastEvent } from '../lib/date-utils.js';

export function ScheduleView({ currentDate, events, onEventClick, onDayClick, config }) {
    const containerRef = useRef(null);
    const todayRef = useRef(null);

    const from = new Date(currentDate.getFullYear(), currentDate.getMonth(), currentDate.getDate() - 30);
    const to = new Date(currentDate.getFullYear(), currentDate.getMonth(), currentDate.getDate() + 30);

    // Group events by date
    const dayMap = new Map();

    events.forEach(event => {
        if (event.all_day) {
            const startDate = event.start_time.substring(0, 10);
            const endDate = event.end_time.substring(0, 10);
            const cur = new Date(startDate + 'T12:00:00');
            const end = new Date(endDate + 'T12:00:00');
            while (cur < end) {
                const key = toDateKey(cur);
                if (!dayMap.has(key)) dayMap.set(key, []);
                dayMap.get(key).push(event);
                cur.setDate(cur.getDate() + 1);
            }
        } else {
            const key = toDateKey(new Date(event.start_time));
            if (!dayMap.has(key)) dayMap.set(key, []);
            dayMap.get(key).push(event);
        }
    });

    const fromKey = toDateKey(from);
    const toKey = toDateKey(to);
    const todayKey = toDateKey(new Date());
    const sortedDays = Array.from(dayMap.keys())
        .filter(k => k >= fromKey && k < toKey)
        .sort();

    // Find the first day >= today to scroll to
    const todayIndex = sortedDays.findIndex(k => k >= todayKey);

    useEffect(() => {
        if (todayRef.current && containerRef.current) {
            containerRef.current.scrollTop = todayRef.current.offsetTop - containerRef.current.offsetTop;
        }
    }, [events, currentDate]);

    function toDateKey(d) {
        const pad = n => String(n).padStart(2, '0');
        return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
    }

    function sortEvents(evts) {
        return [...evts].sort((a, b) => {
            if (a.all_day && !b.all_day) return -1;
            if (!a.all_day && b.all_day) return 1;
            return a.start_time.localeCompare(b.start_time);
        });
    }

    function formatDateHeader(dateKey) {
        const d = new Date(dateKey + 'T12:00:00');
        return d.toLocaleDateString(undefined, { weekday: 'long', month: 'short', day: 'numeric' });
    }

    function formatEventTime(event) {
        if (event.all_day) return null;
        return `${formatTime(event.start_time, config.clockFormat)} – ${formatTime(event.end_time, config.clockFormat)}`;
    }

    function dedup(evts) {
        const seen = new Set();
        return evts.filter(e => {
            const key = e.id + ':' + e.start_time;
            if (seen.has(key)) return false;
            seen.add(key);
            return true;
        });
    }

    return html`
        <div class="schedule-view" ref=${containerRef}>
            ${sortedDays.length === 0 && html`
                <div class="schedule-empty">No upcoming events</div>
            `}
            ${sortedDays.map((dateKey, i) => {
                const dayDate = new Date(dateKey + 'T12:00:00');
                const dayEvents = dedup(sortEvents(dayMap.get(dateKey)));
                const isScrollTarget = i === todayIndex;
                return html`
                    <div class="schedule-date-group" key=${dateKey}
                         ref=${isScrollTarget ? todayRef : null}>
                        <div class="schedule-date-header"
                             onClick=${() => onDayClick(dayDate)}>
                            ${formatDateHeader(dateKey)}
                        </div>
                        ${dayEvents.map(event => html`
                            <div class="schedule-event${isPastEvent(event) ? ' past-event' : ''}"
                                 key=${event.id + ':' + event.start_time}
                                 style=${'background:' + (event.color || '#4285f4')}
                                 onClick=${(e) => { e.stopPropagation(); onEventClick(event); }}>
                                <div class="schedule-event-title">${event.title}</div>
                                ${formatEventTime(event) && html`
                                    <div class="schedule-event-time">${formatEventTime(event)}</div>
                                `}
                                ${event.location && html`
                                    <div class="schedule-event-location">${event.location}</div>
                                `}
                            </div>
                        `)}
                    </div>
                `;
            })}
        </div>
    `;
}
