import type { VNode } from 'preact';
import { html } from 'htm/preact';
import { useRef, useEffect } from 'preact/hooks';
import { formatTime, isPastEvent } from '../lib/date-utils.js';
import { eventColor } from '../lib/event-utils.js';
import type { CalendarEvent, AppConfig } from '../types/models.js';

interface ScheduleViewProps {
    currentDate: Date;
    events: CalendarEvent[];
    onEventClick: (event: CalendarEvent) => void;
    onDayClick: (date: Date) => void;
    config: AppConfig;
    onLoadMore?: () => void;
    daysLoaded?: number;
    highlightEventId?: string | null;
}

export function ScheduleView({ currentDate, events, onEventClick, onDayClick, config, onLoadMore, daysLoaded, highlightEventId }: ScheduleViewProps): VNode | null {
    const containerRef = useRef<HTMLDivElement>(null);
    const sentinelRef = useRef<HTMLDivElement>(null);

    const today = new Date();
    const from = new Date(today.getFullYear(), today.getMonth(), today.getDate());
    const to = new Date(today.getFullYear(), today.getMonth(), today.getDate() + (daysLoaded || 90));

    // Infinite scroll: observe sentinel element
    useEffect(() => {
        if (!onLoadMore || !sentinelRef.current) return;
        const observer = new IntersectionObserver((entries) => {
            if (entries[0].isIntersecting) {
                onLoadMore();
            }
        }, { rootMargin: '200px' });
        observer.observe(sentinelRef.current);
        return () => observer.disconnect();
    }, [onLoadMore]);

    // Group events by date
    const dayMap = new Map<string, CalendarEvent[]>();

    events.forEach(event => {
        if (event.all_day) {
            const startDate = event.start_time.substring(0, 10);
            const endDate = event.end_time.substring(0, 10);
            const cur = new Date(startDate + 'T12:00:00');
            const end = new Date(endDate + 'T12:00:00');
            while (cur < end) {
                const key = toDateKey(cur);
                if (!dayMap.has(key)) dayMap.set(key, []);
                dayMap.get(key)!.push(event);
                cur.setDate(cur.getDate() + 1);
            }
        } else {
            const key = toDateKey(new Date(event.start_time));
            if (!dayMap.has(key)) dayMap.set(key, []);
            dayMap.get(key)!.push(event);
        }
    });

    const fromKey = toDateKey(from);
    const toKey = toDateKey(to);
    const sortedDays = Array.from(dayMap.keys())
        .filter(k => k >= fromKey && k < toKey)
        .sort();

    function toDateKey(d: Date): string {
        const pad = (n: number) => String(n).padStart(2, '0');
        return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
    }

    function sortEvents(evts: CalendarEvent[]): CalendarEvent[] {
        return [...evts].sort((a, b) => {
            if (a.all_day && !b.all_day) return -1;
            if (!a.all_day && b.all_day) return 1;
            return a.start_time.localeCompare(b.start_time);
        });
    }

    function formatDateHeader(dateKey: string): string {
        const d = new Date(dateKey + 'T12:00:00');
        return d.toLocaleDateString(undefined, { weekday: 'long', month: 'short', day: 'numeric' });
    }

    function formatEventTime(event: CalendarEvent): string | null {
        if (event.all_day) return null;
        return `${formatTime(event.start_time)} \u2013 ${formatTime(event.end_time)}`;
    }

    function dedup(evts: CalendarEvent[]): CalendarEvent[] {
        const seen = new Set<string>();
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
            ${sortedDays.map((dateKey) => {
                const dayDate = new Date(dateKey + 'T12:00:00');
                const dayEvents = dedup(sortEvents(dayMap.get(dateKey)!));
                return html`
                    <div class="schedule-date-group" key=${dateKey}>
                        <div class="schedule-date-header"
                             onClick=${() => onDayClick(dayDate)}>
                            ${formatDateHeader(dateKey)}
                        </div>
                        ${dayEvents.map(event => html`
                            <div class="schedule-event${isPastEvent(event) ? ' past-event' : ''}${highlightEventId === event.id + '|' + event.start_time ? ' highlight-event' : ''}"
                                 key=${event.id + ':' + event.start_time}
                                 style=${'background:' + (eventColor(event, config))}
                                 onClick=${(e: MouseEvent) => { e.stopPropagation(); onEventClick(event); }}>
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
            <div ref=${sentinelRef} class="schedule-sentinel"></div>
        </div>
    ` as VNode;
}
