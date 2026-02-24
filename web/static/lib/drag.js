const PIXELS_PER_HOUR = 48;
const SNAP_MINUTES = 15;
const MIN_DURATION_MINUTES = 15;
const CLICK_THRESHOLD = 4; // pixels moved before it counts as a drag

function snapMinutes(minutes) {
    return Math.round(minutes / SNAP_MINUTES) * SNAP_MINUTES;
}

function addMinutes(dateStr, minutes) {
    const d = new Date(dateStr);
    d.setUTCMinutes(d.getUTCMinutes() + minutes);
    return d.toISOString();
}

function shiftDate(dateStr, daysDelta) {
    const d = new Date(dateStr);
    d.setUTCDate(d.getUTCDate() + daysDelta);
    return d.toISOString();
}

function shiftDateOnly(dateStr, daysDelta) {
    // For all-day events: parse YYYY-MM-DD or ISO, shift days, return YYYY-MM-DD
    const d = new Date(dateStr);
    d.setUTCDate(d.getUTCDate() + daysDelta);
    const pad = n => String(n).padStart(2, '0');
    return `${d.getUTCFullYear()}-${pad(d.getUTCMonth()+1)}-${pad(d.getUTCDate())}`;
}

/**
 * Initialize drag behavior on an event element.
 *
 * @param {Object} event - The calendar event object
 * @param {HTMLElement} el - The event DOM element
 * @param {Object} options
 * @param {'move'|'resize'|'move-horizontal'} options.mode
 * @param {Function} options.onDragEnd(newStartTime, newEndTime)
 * @param {Date[]} [options.dayColumns] - Array of day Dates for horizontal movement (week view)
 * @param {HTMLElement} [options.columnsContainer] - Parent element containing day columns
 * @param {string} [options.columnSelector] - CSS selector for day columns (default: '.week-day-events')
 */
export function startDrag(event, el, startEvent, options) {
    // Skip recurring instances
    if (event.recurrence_index > 0) return;

    const { mode, onDragEnd, dayColumns, columnsContainer } = options;
    const columnSelector = options.columnSelector || '.week-day-events';

    // Prevent text selection on mousedown
    startEvent.preventDefault();

    const isTouch = startEvent.type === 'touchstart';
    const startX = isTouch ? startEvent.touches[0].clientX : startEvent.clientX;
    const startY = isTouch ? startEvent.touches[0].clientY : startEvent.clientY;

    const origTop = parseFloat(el.style.top) || 0;
    const origHeight = parseFloat(el.style.height) || 0;
    let isDragging = false;
    let totalDeltaMinutes = 0;
    let dayDelta = 0;

    // Find the starting column index and cache column positions for horizontal movement
    let startColIndex = -1;
    let colRects = [];
    if (dayColumns && columnsContainer) {
        const cols = columnsContainer.querySelectorAll(columnSelector);
        for (let i = 0; i < cols.length; i++) {
            colRects.push(cols[i].getBoundingClientRect());
            if (cols[i].contains(el)) {
                startColIndex = i;
            }
        }
    }

    function getPos(e) {
        if (e.touches) return { x: e.touches[0].clientX, y: e.touches[0].clientY };
        return { x: e.clientX, y: e.clientY };
    }

    function onMove(e) {
        const pos = getPos(e);
        const deltaX = pos.x - startX;
        const deltaY = pos.y - startY;

        if (!isDragging) {
            if (Math.abs(deltaX) < CLICK_THRESHOLD && Math.abs(deltaY) < CLICK_THRESHOLD) return;
            isDragging = true;
            el.classList.add('dragging');
            document.body.style.userSelect = 'none';
            if (isTouch) e.preventDefault();
        }

        if (isDragging && isTouch) e.preventDefault();

        if (mode !== 'move-horizontal') {
            totalDeltaMinutes = snapMinutes((deltaY / PIXELS_PER_HOUR) * 60);
        }

        function detectColumn(posX) {
            if (colRects.length === 0 || startColIndex < 0) return;
            // Refresh column rects (they may shift if the page scrolls)
            const cols = columnsContainer.querySelectorAll(columnSelector);
            for (let i = 0; i < cols.length; i++) {
                colRects[i] = cols[i].getBoundingClientRect();
            }
            let newColIndex = startColIndex;
            for (let i = 0; i < colRects.length; i++) {
                if (posX >= colRects[i].left && posX < colRects[i].right) {
                    newColIndex = i;
                    break;
                }
            }
            dayDelta = newColIndex - startColIndex;
            if (dayDelta !== 0) {
                const offsetX = colRects[newColIndex].left - colRects[startColIndex].left;
                el.style.transform = `translateX(${offsetX}px)`;
            } else {
                el.style.transform = '';
            }
        }

        if (mode === 'move') {
            el.style.top = `${origTop + (totalDeltaMinutes / 60) * PIXELS_PER_HOUR}px`;
            detectColumn(pos.x);
        } else if (mode === 'move-horizontal') {
            detectColumn(pos.x);
        } else if (mode === 'resize') {
            const newHeight = origHeight + (totalDeltaMinutes / 60) * PIXELS_PER_HOUR;
            const minHeight = (MIN_DURATION_MINUTES / 60) * PIXELS_PER_HOUR;
            el.style.height = `${Math.max(newHeight, minHeight)}px`;
            // Clamp totalDeltaMinutes for resize
            const origDuration = (new Date(event.end_time) - new Date(event.start_time)) / 60000;
            if (origDuration + totalDeltaMinutes < MIN_DURATION_MINUTES) {
                totalDeltaMinutes = MIN_DURATION_MINUTES - origDuration;
            }
        }
    }

    function onEnd(e) {
        document.removeEventListener('mousemove', onMove);
        document.removeEventListener('mouseup', onEnd);
        document.removeEventListener('touchmove', onMove);
        document.removeEventListener('touchend', onEnd);

        el.classList.remove('dragging');
        document.body.style.userSelect = '';

        if (!isDragging) return; // was just a click

        // Suppress the click event that follows mouseup
        function suppressClick(e) {
            e.stopPropagation();
            e.preventDefault();
        }
        el.addEventListener('click', suppressClick, { capture: true, once: true });

        let newStart, newEnd;
        if (mode === 'move') {
            newStart = addMinutes(event.start_time, totalDeltaMinutes);
            newEnd = addMinutes(event.end_time, totalDeltaMinutes);
            if (dayDelta !== 0) {
                newStart = shiftDate(newStart, dayDelta);
                newEnd = shiftDate(newEnd, dayDelta);
            }
        } else if (mode === 'move-horizontal') {
            newStart = shiftDateOnly(event.start_time, dayDelta);
            newEnd = shiftDateOnly(event.end_time, dayDelta);
        } else {
            newStart = event.start_time;
            newEnd = addMinutes(event.end_time, totalDeltaMinutes);
        }

        // Reset inline styles so re-render takes over
        el.style.top = `${origTop}px`;
        el.style.height = `${origHeight}px`;
        el.style.transform = '';

        if (totalDeltaMinutes !== 0 || dayDelta !== 0) {
            onDragEnd(newStart, newEnd);
        }
    }

    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onEnd);
    document.addEventListener('touchmove', onMove, { passive: false });
    document.addEventListener('touchend', onEnd);
}
