import type { CalendarEvent, AppConfig } from '../types/models.js';

/**
 * Resolve the display color for an event, falling back to its calendar's color,
 * then the global default.
 */
export function eventColor(event: CalendarEvent, config: AppConfig): string {
    return event.color
        || (config.calendarColors && config.calendarColors[event.calendar_id])
        || config.defaultEventColor
        || 'dodgerblue';
}

/**
 * Compute side-by-side column layout for overlapping timed events.
 * Returns an array parallel to `events` with { col, total } for each event,
 * where col is the 0-based column index and total is the number of columns
 * in that event's overlap cluster.
 */
export function computeOverlapLayout(events: CalendarEvent[]): { col: number; total: number }[] {
    const n = events.length;
    if (n === 0) return [];

    const starts = events.map(e => new Date(e.start_time).getTime());
    const ends = events.map(e => new Date(e.end_time).getTime());

    // Build overlap adjacency list
    const overlaps: number[][] = Array.from({ length: n }, (_, i) => {
        const result: number[] = [];
        for (let j = 0; j < n; j++) {
            if (i !== j && starts[i] < ends[j] && ends[i] > starts[j]) result.push(j);
        }
        return result;
    });

    // Assign columns greedily in start-time order
    const order = Array.from({ length: n }, (_, i) => i)
        .sort((a, b) => starts[a] - starts[b] || ends[b] - ends[a]);
    const cols = new Array(n).fill(-1);
    for (const i of order) {
        const used = new Set(overlaps[i].filter(j => cols[j] >= 0).map(j => cols[j]));
        let col = 0;
        while (used.has(col)) col++;
        cols[i] = col;
    }

    // Find connected clusters; total columns = max col in cluster + 1
    const visited = new Array(n).fill(false);
    const totalCols = new Array(n).fill(1);
    for (let i = 0; i < n; i++) {
        if (!visited[i]) {
            const cluster: number[] = [];
            const queue = [i];
            while (queue.length > 0) {
                const j = queue.shift()!;
                if (visited[j]) continue;
                visited[j] = true;
                cluster.push(j);
                overlaps[j].forEach(k => { if (!visited[k]) queue.push(k); });
            }
            const maxCol = Math.max(...cluster.map(k => cols[k]));
            cluster.forEach(k => { totalCols[k] = maxCol + 1; });
        }
    }

    return events.map((_, i) => ({ col: cols[i], total: totalCols[i] }));
}
