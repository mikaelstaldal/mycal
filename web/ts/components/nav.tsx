import { h } from 'preact';
import type { VNode } from 'preact';
import { formatMonthYear, formatWeekRange, formatDayHeading } from '../lib/date-utils.js';

interface NavProps {
    currentDate: Date;
    onPrev: () => void;
    onNext: () => void;
    onToday: () => void;
    viewMode: string;
    onViewChange: (mode: string) => void;
    weekStartDay: number;
}

export function Nav({ currentDate, onPrev, onNext, onToday, viewMode, onViewChange, weekStartDay }: NavProps): VNode | null {
    const heading = viewMode === 'schedule'
        ? formatDayHeading(currentDate)
        : viewMode === 'day'
        ? formatDayHeading(currentDate)
        : viewMode === 'week'
        ? formatWeekRange(currentDate, weekStartDay)
        : viewMode === 'year'
        ? String(currentDate.getFullYear())
        : formatMonthYear(currentDate);

    return (
        <nav class="nav">
            <button onClick={onToday}>Today</button>
            <button onClick={onPrev}>&#x25C0;</button>
            <button onClick={onNext}>&#x25B6;</button>
            <div class="view-toggle">
                <button class={viewMode === 'year' ? 'active' : ''} onClick={() => onViewChange('year')}>Year</button>
                <button class={viewMode === 'month' ? 'active' : ''} onClick={() => onViewChange('month')}>Month</button>
                <button class={viewMode === 'week' ? 'active' : ''} onClick={() => onViewChange('week')}>Week</button>
                <button class={viewMode === 'day' ? 'active' : ''} onClick={() => onViewChange('day')}>Day</button>
                <button class={viewMode === 'schedule' ? 'active' : ''} onClick={() => onViewChange('schedule')}>Schedule</button>
            </div>
            <h1>{heading}</h1>
        </nav>
    );
}
