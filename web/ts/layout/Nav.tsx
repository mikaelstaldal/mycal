import type { VNode } from 'preact';
import { formatMonthYear, formatWeekRange, formatDayHeading } from '../util/date-utils.js';
import { Icon } from '../components/Icon.js';

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
            <button data-shortcut="t" aria-keyshortcuts="t" onClick={onToday}><span><u>T</u>oday</span></button>
            <button onClick={onPrev} aria-label="Previous" title="Previous"><Icon name="chevron-left" /></button>
            <button onClick={onNext} aria-label="Next" title="Next"><Icon name="chevron-right" /></button>
            <div class="view-toggle">
                <button data-shortcut="y" aria-keyshortcuts="y" class={viewMode === 'year' ? 'active' : ''} onClick={() => onViewChange('year')}><span><u>Y</u>ear</span></button>
                <button data-shortcut="m" aria-keyshortcuts="m" class={viewMode === 'month' ? 'active' : ''} onClick={() => onViewChange('month')}><span><u>M</u>onth</span></button>
                <button data-shortcut="w" aria-keyshortcuts="w" class={viewMode === 'week' ? 'active' : ''} onClick={() => onViewChange('week')}><span><u>W</u>eek</span></button>
                <button data-shortcut="d" aria-keyshortcuts="d" class={viewMode === 'day' ? 'active' : ''} onClick={() => onViewChange('day')}><span><u>D</u>ay</span></button>
                <button data-shortcut="s" aria-keyshortcuts="s" class={viewMode === 'schedule' ? 'active' : ''} onClick={() => onViewChange('schedule')}><span><u>S</u>chedule</span></button>
            </div>
            <h1>{heading}</h1>
        </nav>
    );
}
