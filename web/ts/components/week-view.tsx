import { h, Fragment } from 'preact';
import type { VNode } from 'preact';
import { useState, useEffect, useRef, useMemo, useCallback } from 'preact/hooks';
import { getWeekDays, isToday, formatHour, formatTime, getISOWeekNumber, isPastEvent } from '../lib/date-utils.js';
import { startDrag } from '../lib/drag.js';
import { eventColor, computeOverlapLayout, buildDayIndex, dayKey } from '../lib/event-utils.js';
import type { CalendarEvent, AppConfig } from '../types/models.js';

const HOURS = Array.from({ length: 24 }, (_, i) => i);

const HOVER_CLASSES = ['hour-cell--hover-full', 'hour-cell--hover-top-half', 'hour-cell--hover-bottom-half'];
let _weekHoverCells: Element[] = [];

interface WeekViewProps {
    currentDate: Date;
    events: CalendarEvent[];
    onDayClick: (date: Date) => void;
    onEventClick: (event: CalendarEvent) => void;
    onAllDayClick: (date: Date) => void;
    onEventDrag: (eventId: string, startTime: string, endTime: string) => void;
    config: AppConfig;
    highlightEventId: string | null;
}

export function WeekView({ currentDate, events, onDayClick, onEventClick, onAllDayClick, onEventDrag, config, highlightEventId }: WeekViewProps): VNode | null {
    const weekStartDay = config.weekStartDay;
    const days = useMemo(
        () => getWeekDays(currentDate, weekStartDay),
        [currentDate.getFullYear(), currentDate.getMonth(), currentDate.getDate(), weekStartDay]
    );
    const dayIndex = useMemo(() => buildDayIndex(events, days), [events, days]);
    const overlapLayouts = useMemo(() => {
        const map = new Map<string, { col: number; total: number }[]>();
        for (const date of days) {
            const k = dayKey(date);
            map.set(k, computeOverlapLayout(dayIndex.get(k)?.timed ?? []));
        }
        return map;
    }, [dayIndex, days]);

    const eventStyle = useCallback(function(event: CalendarEvent, date: Date, col: number, total: number) {
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
    }, [config]);

    const dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];

    const hasAnyAllDay = useMemo(() => days.some(date => (dayIndex.get(dayKey(date))?.allDay.length ?? 0) > 0), [days, dayIndex]);
    const maxAllDay = 2;
    const maxAllDayCount = useMemo(() => Math.max(...days.map(date => dayIndex.get(dayKey(date))?.allDay.length ?? 0)), [days, dayIndex]);
    const hasOverflow = maxAllDayCount > maxAllDay;
    const [allDayExpanded, setAllDayExpanded] = useState(false);

    const overlayRef = useRef<HTMLDivElement | null>(null);
    const alldayRowRef = useRef<HTMLDivElement | null>(null);

    const weekBodyRef = useRef<HTMLDivElement | null>(null);
    useEffect(() => {
        if (weekBodyRef.current) {
            const hour = config.dayStartHour || 0;
            weekBodyRef.current.scrollTop = hour * 48;
        }
    }, []);

    return (
        <div class="week-view">
            <div class="week-header">
                <div class="time-gutter-header">week {getISOWeekNumber(days[0])}</div>
                {days.map(date => {
                    const classes = ['week-day-header', isToday(date) && 'today'].filter(Boolean).join(' ');
                    return (
                        <div class={classes} onClick={() => onAllDayClick(date)}>
                            <span class="week-day-name">{dayNames[date.getDay()]}</span>
                            <span class="week-day-number">{date.getDate()}</span>
                        </div>
                    );
                })}
            </div>
            <div class="week-allday-row" ref={alldayRowRef}>
                <div class="allday-label">
                    all-day
                    {hasOverflow && (
                        <div class="allday-toggle" onClick={() => setAllDayExpanded(!allDayExpanded)}
                             role="button" aria-label={allDayExpanded ? 'Collapse all-day events' : 'Expand all-day events'}>
                            {allDayExpanded ? '▲' : '▼'}
                        </div>
                    )}
                </div>
                {days.map(date => {
                    const adEvents = dayIndex.get(dayKey(date))?.allDay ?? [];
                    const visible = allDayExpanded ? adEvents : adEvents.slice(0, maxAllDay);
                    const hidden = adEvents.length - visible.length;
                    return (
                        <div class="allday-cell" onClick={() => onAllDayClick(date)}>
                            {visible.map(e => {
                                const canDrag = !e.parent_id;
                                return (
                                    <div class={`allday-event${isPastEvent(e) ? ' past-event' : ''}${highlightEventId === e.id + '|' + e.start_time ? ' highlight-event' : ''}`}
                                         key={e.id}
                                         title={e.title}
                                         style={`background-color: ${eventColor(e, config)}`}
                                         onClick={(ev: MouseEvent) => { ev.stopPropagation(); onEventClick(e); }}
                                         onMouseDown={canDrag ? (ev: MouseEvent) => {
                                             if (ev.button !== 0) return;
                                             startDrag(e, ev.currentTarget as HTMLElement, ev, {
                                                 mode: 'move-horizontal',
                                                 dayColumns: days,
                                                 columnsContainer: alldayRowRef.current!,
                                                 columnSelector: '.allday-cell',
                                                 onDragEnd: (s: string, end: string) => onEventDrag(e.id, s, end)
                                             });
                                         } : undefined}
                                         onTouchStart={canDrag ? (ev: TouchEvent) => {
                                             startDrag(e, ev.currentTarget as HTMLElement, ev, {
                                                 mode: 'move-horizontal',
                                                 dayColumns: days,
                                                 columnsContainer: alldayRowRef.current!,
                                                 columnSelector: '.allday-cell',
                                                 onDragEnd: (s: string, end: string) => onEventDrag(e.id, s, end)
                                             });
                                         } : undefined}>
                                        {e.title}
                                    </div>
                                );
                            })}
                            {hidden > 0 && (
                                <div class="allday-more" onClick={(ev: MouseEvent) => { ev.stopPropagation(); setAllDayExpanded(true); }}>
                                    +{hidden} more
                                </div>
                            )}
                        </div>
                    );
                })}
            </div>
            <div class="week-body" ref={weekBodyRef}>
                <div class="week-grid">
                    {HOURS.map(hour => (
                        <Fragment>
                            <div class="time-gutter">{formatHour(hour)}</div>
                            {days.map((date) => (
                                <div class="hour-cell"
                                     onMouseMove={(ev: MouseEvent) => {
                                         _weekHoverCells.forEach(c => c.classList.remove(...HOVER_CLASSES));
                                         const cell = ev.currentTarget as HTMLElement;
                                         if (ev.offsetY < 24) {
                                             cell.classList.add('hour-cell--hover-full');
                                             _weekHoverCells = [cell];
                                         } else {
                                             cell.classList.add('hour-cell--hover-bottom-half');
                                             const allCells = cell.closest('.week-grid')!.querySelectorAll('.hour-cell');
                                             const idx = Array.from(allCells).indexOf(cell);
                                             const nextCell = allCells[idx + 7];
                                             if (nextCell) {
                                                 nextCell.classList.add('hour-cell--hover-top-half');
                                                 _weekHoverCells = [cell, nextCell];
                                             } else {
                                                 _weekHoverCells = [cell];
                                             }
                                         }
                                     }}
                                     onMouseLeave={() => {
                                         _weekHoverCells.forEach(c => c.classList.remove(...HOVER_CLASSES));
                                         _weekHoverCells = [];
                                     }}
                                     onClick={(ev: MouseEvent) => {
                                         const minutes = ev.offsetY >= 24 ? 30 : 0;
                                         const d = new Date(date.getFullYear(), date.getMonth(), date.getDate(), hour, minutes);
                                         onDayClick(d);
                                     }}>
                                </div>
                            ))}
                        </Fragment>
                    ))}
                </div>
                <div class="week-events-overlay" ref={overlayRef}>
                    <div class="week-events-gutter-spacer"></div>
                    {days.map((date) => {
                        const k = dayKey(date);
                        const dayEvents = dayIndex.get(k)?.timed ?? [];
                        const layout = overlapLayouts.get(k)!;
                        return (
                            <div class="week-day-events">
                                {dayEvents.map((e, ei) => {
                                    const { col, total } = layout[ei];
                                    const durationMin = (new Date(e.end_time).getTime() - new Date(e.start_time).getTime()) / 60000;
                                    const isShort = durationMin <= 30;
                                    const isHighlighted = highlightEventId === e.id + '|' + e.start_time;
                                    const classes = ['week-event', isShort && 'short-event', isPastEvent(e) && 'past-event', isHighlighted && 'highlight-event'].filter(Boolean).join(' ');
                                    const canDrag = !e.parent_id;
                                    return (
                                        <div class={classes}
                                             key={e.id}
                                             title={e.title}
                                             style={eventStyle(e, date, col, total)}
                                             onClick={(ev: MouseEvent) => { ev.stopPropagation(); onEventClick(e); }}
                                             onMouseDown={canDrag ? (ev: MouseEvent) => {
                                                 if (ev.button !== 0) return;
                                                 startDrag(e, ev.currentTarget as HTMLElement, ev, {
                                                     mode: 'move',
                                                     dayColumns: days,
                                                     columnsContainer: overlayRef.current!,
                                                     onDragEnd: (s: string, end: string) => onEventDrag(e.id, s, end)
                                                 });
                                             } : undefined}
                                             onTouchStart={canDrag ? (ev: TouchEvent) => {
                                                 startDrag(e, ev.currentTarget as HTMLElement, ev, {
                                                     mode: 'move',
                                                     dayColumns: days,
                                                     columnsContainer: overlayRef.current!,
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
                                })}
                            </div>
                        );
                    })}
                </div>
            </div>
        </div>
    );
}
