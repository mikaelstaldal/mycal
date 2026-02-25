import { html } from 'htm/preact';
import { useState, useEffect, useRef } from 'preact/hooks';
import { getWeekDays, isToday, formatHour, formatTime, getISOWeekNumber, isPastEvent } from '../lib/date-utils.js';
import { startDrag } from '../lib/drag.js';

const HOURS = Array.from({ length: 24 }, (_, i) => i);

export function WeekView({ currentDate, events, onDayClick, onEventClick, onAllDayClick, onEventDrag, config }) {
    const weekStartDay = config.weekStartDay;
    const days = getWeekDays(currentDate, weekStartDay);

    function eventsForDay(date) {
        const dayStart = new Date(date.getFullYear(), date.getMonth(), date.getDate());
        const dayEnd = new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1);
        return events.filter(e => {
            if (e.all_day) {
                // Compare dates only (UTC) to avoid timezone shift issues
                const startDate = e.start_time.substring(0, 10);
                const endDate = e.end_time.substring(0, 10); // exclusive
                const pad = n => String(n).padStart(2, '0');
                const dayStr = `${date.getFullYear()}-${pad(date.getMonth()+1)}-${pad(date.getDate())}`;
                return dayStr >= startDate && dayStr < endDate;
            }
            const start = new Date(e.start_time);
            const end = new Date(e.end_time);
            return start < dayEnd && end > dayStart;
        });
    }

    function timedEventsForDay(date) {
        return eventsForDay(date).filter(e => !e.all_day);
    }

    function allDayEventsForDay(date) {
        return eventsForDay(date).filter(e => e.all_day);
    }

    function eventStyle(event, date) {
        const start = new Date(event.start_time);
        const end = new Date(event.end_time);
        const dayStart = new Date(date.getFullYear(), date.getMonth(), date.getDate());
        const dayEnd = new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1);

        const effectiveStart = start < dayStart ? dayStart : start;
        const effectiveEnd = end > dayEnd ? dayEnd : end;

        const startMinutes = (effectiveStart - dayStart) / 60000;
        const endMinutes = (effectiveEnd - dayStart) / 60000;
        const duration = endMinutes - startMinutes;

        const top = (startMinutes / 60) * 48;
        const height = Math.max((duration / 60) * 48, 18);

        return {
            top: `${top}px`,
            height: `${height}px`,
            backgroundColor: event.color || '#4285f4'
        };
    }

    const dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

    const hasAnyAllDay = days.some(date => allDayEventsForDay(date).length > 0);
    const maxAllDay = 2;
    const maxAllDayCount = Math.max(...days.map(date => allDayEventsForDay(date).length));
    const hasOverflow = maxAllDayCount > maxAllDay;
    const [allDayExpanded, setAllDayExpanded] = useState(false);

    const overlayRef = useRef(null);
    const alldayRowRef = useRef(null);

    const weekBodyRef = useRef(null);
    useEffect(() => {
        if (weekBodyRef.current) {
            const hour = config.dayStartHour || 0;
            weekBodyRef.current.scrollTop = hour * 48;
        }
    }, []);

    return html`
        <div class="week-view">
            <div class="week-header">
                <div class="time-gutter-header">W${getISOWeekNumber(days[0])}</div>
                ${days.map(date => {
                    const classes = ['week-day-header', isToday(date) && 'today'].filter(Boolean).join(' ');
                    return html`
                        <div class=${classes}>
                            <span class="week-day-name">${dayNames[date.getDay()]}</span>
                            <span class="week-day-number">${date.getDate()}</span>
                        </div>
                    `;
                })}
            </div>
            <div class="week-allday-row" ref=${alldayRowRef}>
                <div class="allday-label">
                    all-day
                    ${hasOverflow && html`
                        <div class="allday-toggle" onClick=${() => setAllDayExpanded(!allDayExpanded)}>
                            ${allDayExpanded ? '\u25B2' : '\u25BC'}
                        </div>
                    `}
                </div>
                ${days.map(date => {
                    const adEvents = allDayEventsForDay(date);
                    const visible = allDayExpanded ? adEvents : adEvents.slice(0, maxAllDay);
                    const hidden = adEvents.length - visible.length;
                    return html`
                        <div class="allday-cell" onClick=${() => onAllDayClick(date)}>
                            ${visible.map(e => {
                                const canDrag = !e.recurrence_index;
                                return html`
                                    <div class=${`allday-event${isPastEvent(e) ? ' past-event' : ''}`}
                                         key=${`${e.id}-${e.recurrence_index || 0}`}
                                         title=${e.title}
                                         style=${e.color ? `background-color: ${e.color}` : ''}
                                         onClick=${(ev) => { ev.stopPropagation(); onEventClick(e); }}
                                         onMouseDown=${canDrag ? (ev) => {
                                             if (ev.button !== 0) return;
                                             startDrag(e, ev.currentTarget, ev, {
                                                 mode: 'move-horizontal',
                                                 dayColumns: days,
                                                 columnsContainer: alldayRowRef.current,
                                                 columnSelector: '.allday-cell',
                                                 onDragEnd: (s, end) => onEventDrag(e.id, s, end)
                                             });
                                         } : undefined}
                                         onTouchStart=${canDrag ? (ev) => {
                                             startDrag(e, ev.currentTarget, ev, {
                                                 mode: 'move-horizontal',
                                                 dayColumns: days,
                                                 columnsContainer: alldayRowRef.current,
                                                 columnSelector: '.allday-cell',
                                                 onDragEnd: (s, end) => onEventDrag(e.id, s, end)
                                             });
                                         } : undefined}>
                                        ${e.title}
                                    </div>
                                `;
                            })}
                            ${hidden > 0 && html`
                                <div class="allday-more" onClick=${(ev) => { ev.stopPropagation(); setAllDayExpanded(true); }}>
                                    +${hidden} more
                                </div>
                            `}
                        </div>
                    `;
                })}
            </div>
            <div class="week-body" ref=${weekBodyRef}>
                <div class="week-grid">
                    ${HOURS.map(hour => html`
                        <div class="time-gutter">${formatHour(hour, config.clockFormat)}</div>
                        ${days.map((date, colIndex) => html`
                            <div class="hour-cell"
                                 onClick=${() => {
                                     const d = new Date(date.getFullYear(), date.getMonth(), date.getDate(), hour);
                                     onDayClick(d);
                                 }}>
                            </div>
                        `)}
                    `)}
                </div>
                <div class="week-events-overlay" ref=${overlayRef}>
                    <div class="week-events-gutter-spacer"></div>
                    ${days.map((date, colIndex) => {
                        const dayEvents = timedEventsForDay(date);
                        return html`
                            <div class="week-day-events">
                                ${dayEvents.map(e => {
                                    const durationMin = (new Date(e.end_time) - new Date(e.start_time)) / 60000;
                                    const isShort = durationMin <= 30;
                                    const classes = ['week-event', isShort && 'short-event', isPastEvent(e) && 'past-event'].filter(Boolean).join(' ');
                                    const canDrag = !e.recurrence_index;
                                    return html`
                                        <div class=${classes}
                                             key=${`${e.id}-${e.recurrence_index || 0}`}
                                             title=${e.title}
                                             style=${eventStyle(e, date)}
                                             onClick=${(ev) => { ev.stopPropagation(); onEventClick(e); }}
                                             onMouseDown=${canDrag ? (ev) => {
                                                 if (ev.button !== 0) return;
                                                 startDrag(e, ev.currentTarget, ev, {
                                                     mode: 'move',
                                                     dayColumns: days,
                                                     columnsContainer: overlayRef.current,
                                                     onDragEnd: (s, end) => onEventDrag(e.id, s, end)
                                                 });
                                             } : undefined}
                                             onTouchStart=${canDrag ? (ev) => {
                                                 startDrag(e, ev.currentTarget, ev, {
                                                     mode: 'move',
                                                     dayColumns: days,
                                                     columnsContainer: overlayRef.current,
                                                     onDragEnd: (s, end) => onEventDrag(e.id, s, end)
                                                 });
                                             } : undefined}>
                                            ${isShort ? html`
                                                <span class="week-event-time">${formatTime(e.start_time, config.clockFormat)}</span>
                                                <span class="week-event-title">${e.title}</span>
                                            ` : html`
                                                <span class="week-event-title">${e.title}</span>
                                                <span class="week-event-time">${formatTime(e.start_time, config.clockFormat)}</span>
                                            `}
                                            ${canDrag && html`<div class="resize-handle"
                                                onMouseDown=${(ev) => {
                                                    ev.stopPropagation();
                                                    startDrag(e, ev.currentTarget.parentElement, ev, {
                                                        mode: 'resize',
                                                        onDragEnd: (s, end) => onEventDrag(e.id, s, end)
                                                    });
                                                }}
                                                onTouchStart=${(ev) => {
                                                    ev.stopPropagation();
                                                    startDrag(e, ev.currentTarget.parentElement, ev, {
                                                        mode: 'resize',
                                                        onDragEnd: (s, end) => onEventDrag(e.id, s, end)
                                                    });
                                                }} />`}
                                        </div>
                                    `;
                                })}
                            </div>
                        `;
                    })}
                </div>
            </div>
        </div>
    `;
}
