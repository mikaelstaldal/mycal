import { html } from 'htm/preact';
import { useState, useEffect, useRef } from 'preact/hooks';
import { isToday, formatHour, formatTime, isPastEvent } from '../lib/date-utils.js';
import { startDrag } from '../lib/drag.js';

const HOURS = Array.from({ length: 24 }, (_, i) => i);

export function DayView({ currentDate, events, onDayClick, onEventClick, onAllDayClick, onEventDrag, config }) {
    const date = new Date(currentDate.getFullYear(), currentDate.getMonth(), currentDate.getDate());

    function eventsForDay() {
        const dayStart = new Date(date.getFullYear(), date.getMonth(), date.getDate());
        const dayEnd = new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1);
        return events.filter(e => {
            if (e.all_day) {
                const startDate = e.start_time.substring(0, 10);
                const endDate = e.end_time.substring(0, 10);
                const pad = n => String(n).padStart(2, '0');
                const dayStr = `${date.getFullYear()}-${pad(date.getMonth()+1)}-${pad(date.getDate())}`;
                return dayStr >= startDate && dayStr < endDate;
            }
            const start = new Date(e.start_time);
            const end = new Date(e.end_time);
            return start < dayEnd && end > dayStart;
        });
    }

    function timedEvents() {
        return eventsForDay().filter(e => !e.all_day);
    }

    function allDayEvents() {
        return eventsForDay().filter(e => e.all_day);
    }

    function eventStyle(event) {
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
    const adEvents = allDayEvents();
    const hasAllDay = adEvents.length > 0;

    const dayBodyRef = useRef(null);
    useEffect(() => {
        if (dayBodyRef.current) {
            const hour = config.dayStartHour || 0;
            dayBodyRef.current.scrollTop = hour * 48;
        }
    }, []);

    const todayClass = isToday(date) ? ' today' : '';

    return html`
        <div class="day-view">
            <div class="day-view-header">
                <div class=${'day-view-day-header' + todayClass}>
                    <span class="day-view-day-name">${dayNames[date.getDay()]}</span>
                    <span class="day-view-day-number">${date.getDate()}</span>
                </div>
            </div>
            <div class="day-view-allday-row">
                <div class="allday-label">all-day</div>
                <div class="day-view-allday-cell" onClick=${() => onAllDayClick(date)}>
                    ${adEvents.map(e => html`
                        <div class=${`allday-event${isPastEvent(e) ? ' past-event' : ''}`}
                             key=${`${e.id}-${e.recurrence_index || 0}`}
                             title=${e.title}
                             style=${e.color ? `background-color: ${e.color}` : ''}
                             onClick=${(ev) => { ev.stopPropagation(); onEventClick(e); }}>
                            ${e.title}
                        </div>
                    `)}
                </div>
            </div>
            <div class="day-view-body" ref=${dayBodyRef}>
                <div class="day-view-grid">
                    ${HOURS.map(hour => html`
                        <div class="time-gutter">${formatHour(hour, config.clockFormat)}</div>
                        <div class="hour-cell"
                             onClick=${() => {
                                 const d = new Date(date.getFullYear(), date.getMonth(), date.getDate(), hour);
                                 onDayClick(d);
                             }}>
                        </div>
                    `)}
                </div>
                <div class="day-view-events-overlay">
                    <div class="day-view-events-gutter-spacer"></div>
                    <div class="day-view-day-events">
                        ${timedEvents().map(e => {
                            const durationMin = (new Date(e.end_time) - new Date(e.start_time)) / 60000;
                            const isShort = durationMin <= 30;
                            const classes = ['week-event', isShort && 'short-event', isPastEvent(e) && 'past-event'].filter(Boolean).join(' ');
                            const canDrag = !e.recurrence_index;
                            return html`
                                <div class=${classes}
                                     key=${`${e.id}-${e.recurrence_index || 0}`}
                                     title=${e.title}
                                     style=${eventStyle(e)}
                                     onClick=${(ev) => { ev.stopPropagation(); onEventClick(e); }}
                                     onMouseDown=${canDrag ? (ev) => {
                                         if (ev.button !== 0) return;
                                         startDrag(e, ev.currentTarget, ev, {
                                             mode: 'move',
                                             onDragEnd: (s, end) => onEventDrag(e.id, s, end)
                                         });
                                     } : undefined}
                                     onTouchStart=${canDrag ? (ev) => {
                                         startDrag(e, ev.currentTarget, ev, {
                                             mode: 'move',
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
                </div>
            </div>
        </div>
    `;
}
