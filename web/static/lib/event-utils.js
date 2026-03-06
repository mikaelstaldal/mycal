/**
 * Resolve the display color for an event, falling back to its calendar's color,
 * then the global default.
 */
export function eventColor(event, config) {
    return event.color
        || (config.calendarColors && config.calendarColors[event.calendar_id])
        || config.defaultEventColor
        || 'dodgerblue';
}
