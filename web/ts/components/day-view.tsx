import { h, Fragment } from 'preact';
import type { VNode } from 'preact';
import { useState, useEffect, useRef, useMemo, useCallback } from 'preact/hooks';
import { isToday, formatHour, formatTime, isPastEvent } from '../lib/date-utils.js';
import { startDrag } from '../lib/drag.js';
import { eventColor, computeOverlapLayout, buildDayIndex, dayKey } from '../lib/event-utils.js';
import type { CalendarEvent, AppConfig } from '../types/models.js';

const HOURS = Array.from({ length: 24 }, (_, i) => i);

const HOVER_CLASSES = ['hour-cell--hover-full', 'hour-cell--hover-top-half', 'hour-cell--hover-bottom-half'];
let _dayHoverCells: Element[] = [];

interface DayViewProps {
    currentDate: Date;
    events: CalendarEvent[];
    onDayClick: (date: Date) => void;
    onEventClick: (event: CalendarEvent) => void;
    onAllDayClick: (date: Date) => void;
    onEventDrag: (eventId: string, startTime: string, endTime: string) => void;
    config: AppConfig;
    highlightEventId: string | null;
}

export function DayView({ currentDate, events, onDayClick, onEventClick, onAllDayClick, onEventDrag, config, highlightEventId }: DayViewProps): VNode | null {
    const date = useMemo(
        () => new Date(currentDate.getFullYear(), currentDate.getMonth(), currentDate.getDate()),
        [currentDate.getFullYear(), currentDate.getMonth(), currentDate.getDate()]
    );
    const _dayEntry = useMemo(() => buildDayIndex(events, [date]).get(dayKey(date))!, [events, date]);
    const timedEvents = _dayEntry.timed;
    const allDayEvents = _dayEntry.allDay;
    const overlapLayout = useMemo(() => computeOverlapLayout(timedEvents), [timedEvents]);

    const eventStyle = useCallback(function(event: CalendarEvent, col: number, total: number) {
        const start = new Date(event.start_time);
        const end = new Date(event.end_time);
        const dayStart = new Date(date.getFullYear(), date.getMonth(), date.getDate());
        const dayEnd = new Date(date.getFullYear(), date.getMonth(), date.getDate() + 1);

        const effectiveStart = start < dayStart ? dayStart : start;
        const effectiveEnd = end > dayEnd ? dayEnd : end;

        const startMinutes = (effectiveStart.getTime() - dayStart.getTime()) / 60000;
        const endMinutes = (effectiveEnd.getTime() - dayStart.getTime()) / 60000;
        const duration = endMinutes - startMinutes;

        const top = (startMinutes / 60) * 48;
        const height = Math.max((duration / 60) * 48, 18);

        const colWidth = 100 / total;
        const left = col * colWidth;

        return {
            top: `${top}px`,
            height: `${height}px`,
            left: `calc(${left}% + 1px)`,
            width: `calc(${colWidth}% - 2px)`,
            backgroundColor: eventColor(event, config)
        };
    }, [date, config]);

    const dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    const adEvents = allDayEvents;

    const dayBodyRef = useRef<HTMLDivElement | null>(null);
    useEffect(() => {
        if (dayBodyRef.current) {
            const hour = config.dayStartHour || 0;
            dayBodyRef.current.scrollTop = hour * 48;
        }
    }, []);

    const todayClass = isToday(date) ? ' today' : '';

    return (
        <div class="day-view">
            <div class="day-view-header">
                <div class={'day-view-day-header' + todayClass} onClick={() => onAllDayClick(date)}>
                    <span class="day-view-day-name">{dayNames[date.getDay()]}</span>
                    <span class="day-view-day-number">{date.getDate()}</span>
                </div>
            </div>
            <div class="day-view-allday-row">
                <div class="allday-label">all-day</div>
                <div class="day-view-allday-cell" onClick={() => onAllDayClick(date)}>
                    {adEvents.map(e => (
                        <div class={`allday-event${isPastEvent(e) ? ' past-event' : ''}${highlightEventId === e.id + '|' + e.start_time ? ' highlight-event' : ''}`}
                             key={e.id}
                             title={e.title}
                             style={`background-color: ${eventColor(e, config)}`}
                             onClick={(ev: MouseEvent) => { ev.stopPropagation(); onEventClick(e); }}>
                            {e.title}
                        </div>
                    ))}
                </div>
            </div>
            <div class="day-view-body" ref={dayBodyRef}>
                <div class="day-view-grid">
                    {HOURS.map(hour => (
                        <Fragment>
                            <div class="time-gutter">{formatHour(hour)}</div>
                            <div class="hour-cell"
                                 onMouseMove={(ev: MouseEvent) => {
                                     _dayHoverCells.forEach(c => c.classList.remove(...HOVER_CLASSES));
                                     const cell = ev.currentTarget as HTMLElement;
                                     if (ev.offsetY < 24) {
                                         cell.classList.add('hour-cell--hover-full');
                                         _dayHoverCells = [cell];
                                     } else {
                                         cell.classList.add('hour-cell--hover-bottom-half');
                                         const allCells = cell.closest('.day-view-grid')!.querySelectorAll('.hour-cell');
                                         const idx = Array.from(allCells).indexOf(cell);
                                         const nextCell = allCells[idx + 1];
                                         if (nextCell) {
                                             nextCell.classList.add('hour-cell--hover-top-half');
                                             _dayHoverCells = [cell, nextCell];
                                         } else {
                                             _dayHoverCells = [cell];
                                         }
                                     }
                                 }}
                                 onMouseLeave={() => {
                                     _dayHoverCells.forEach(c => c.classList.remove(...HOVER_CLASSES));
                                     _dayHoverCells = [];
                                 }}
                                 onClick={(ev: MouseEvent) => {
                                     const minutes = ev.offsetY >= 24 ? 30 : 0;
                                     const d = new Date(date.getFullYear(), date.getMonth(), date.getDate(), hour, minutes);
                                     onDayClick(d);
                                 }}>
                            </div>
                        </Fragment>
                    ))}
                </div>
                <div class="day-view-events-overlay">
                    <div class="day-view-events-gutter-spacer"></div>
                    <div class="day-view-day-events">
                        {(() => {
                            return timedEvents.map((e, ei) => {
                            const { col, total } = overlapLayout[ei];
                            const durationMin = (new Date(e.end_time).getTime() - new Date(e.start_time).getTime()) / 60000;
                            const isShort = durationMin <= 30;
                            const isHighlighted = highlightEventId === e.id + '|' + e.start_time;
                            const classes = ['week-event', isShort && 'short-event', isPastEvent(e) && 'past-event', isHighlighted && 'highlight-event'].filter(Boolean).join(' ');
                            const canDrag = !e.parent_id;
                            return (
                                <div class={classes}
                                     key={e.id}
                                     title={e.title}
                                     style={eventStyle(e, col, total)}
                                     onClick={(ev: MouseEvent) => { ev.stopPropagation(); onEventClick(e); }}
                                     onMouseDown={canDrag ? (ev: MouseEvent) => {
                                         if (ev.button !== 0) return;
                                         startDrag(e, ev.currentTarget as HTMLElement, ev, {
                                             mode: 'move',
                                             onDragEnd: (s: string, end: string) => onEventDrag(e.id, s, end)
                                         });
                                     } : undefined}
                                     onTouchStart={canDrag ? (ev: TouchEvent) => {
                                         startDrag(e, ev.currentTarget as HTMLElement, ev, {
                                             mode: 'move',
                                             onDragEnd: (s: string, end: string) => onEventDrag(e.id, s, end)
                                         });
                                     } : undefined}>
                                    {isShort ? (
                                        <Fragment>
                                            <span class="week-event-time">{formatTime(e.start_time)}</span>
                                            <span class="week-event-title">{e.title}</span>
                                        </Fragment>
                                    ) : (
                                        <Fragment>
                                            <span class="week-event-title">{e.title}</span>
                                            <span class="week-event-time">{formatTime(e.start_time)}</span>
                                        </Fragment>
                                    )}
                                    {canDrag && <div class="resize-handle"
                                        onMouseDown={(ev: MouseEvent) => {
                                            ev.stopPropagation();
                                            startDrag(e, (ev.currentTarget as HTMLElement).parentElement!, ev, {
                                                mode: 'resize',
                                                onDragEnd: (s: string, end: string) => onEventDrag(e.id, s, end)
                                            });
                                        }}
                                        onTouchStart={(ev: TouchEvent) => {
                                            ev.stopPropagation();
                                            startDrag(e, (ev.currentTarget as HTMLElement).parentElement!, ev, {
                                                mode: 'resize',
                                                onDragEnd: (s: string, end: string) => onEventDrag(e.id, s, end)
                                            });
                                        }} />}
                                </div>
                            );
                        });
                        })()}
                    </div>
                </div>
            </div>
        </div>
    );
}
