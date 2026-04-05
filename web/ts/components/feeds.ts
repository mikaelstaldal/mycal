import type { VNode } from 'preact';
import { html } from 'htm/preact';
import { useState, useRef, useEffect } from 'preact/hooks';
import { listFeeds, createFeed, deleteFeed, refreshFeed } from '../lib/api.js';
import { COLORS } from '../lib/colors.js';
import { showConfirm } from '../lib/confirm.js';
import type { Feed } from '../types/models.js';

interface FeedsDialogProps {
    onClose: () => void;
    onRefreshed?: () => void;
}

export function FeedsDialog({ onClose, onRefreshed }: FeedsDialogProps): VNode | null {
    const [feeds, setFeeds] = useState<Feed[]>([]);
    const [showAdd, setShowAdd] = useState(false);
    const [loading, setLoading] = useState(true);
    const [refreshingId, setRefreshingId] = useState<number | null>(null);
    const [error, setError] = useState('');
    const dialogRef = useRef<HTMLDialogElement | null>(null);

    useEffect(() => {
        if (dialogRef.current && !dialogRef.current.open) {
            dialogRef.current.showModal();
        }
        loadFeedsData();
    }, []);

    async function loadFeedsData() {
        try {
            setLoading(true);
            const data = await listFeeds();
            setFeeds(data);
        } catch (err: any) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }

    async function handleDelete(id: number) {
        const confirmed = await showConfirm('Delete this feed subscription?', {
            title: 'Delete Feed',
            okText: 'Delete',
            danger: true
        });
        if (!confirmed) return;
        try {
            await deleteFeed(id);
            await loadFeedsData();
        } catch (err: any) {
            setError(err.message);
        }
    }

    async function handleRefresh(id: number) {
        setRefreshingId(id);
        try {
            await refreshFeed(id);
            await loadFeedsData();
            if (onRefreshed) onRefreshed();
        } catch (err: any) {
            setError(err.message);
        } finally {
            setRefreshingId(null);
        }
    }

    async function handleAdd(data: any) {
        try {
            await createFeed(data);
            setShowAdd(false);
            await loadFeedsData();
        } catch (err) {
            throw err;
        }
    }

    function formatDate(dateStr?: string) {
        if (!dateStr) return 'Never';
        const d = new Date(dateStr);
        return d.toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' } as any);
    }

    return html`
        <dialog ref=${dialogRef} class="event-dialog feeds-dialog" onClose=${onClose}>
            <div class="dialog-header">
                <h2>Feed Subscriptions</h2>
                <button class="close-btn" onClick=${onClose}>\u00d7</button>
            </div>
            ${error && html`<div class="feed-error">${error}</div>`}
            ${loading ? html`<div class="feed-loading">Loading...</div>` : html`
                ${feeds.length === 0 && !showAdd ? html`
                    <div class="feed-empty">No feed subscriptions yet.</div>
                ` : html`
                    <div class="feed-list">
                        ${(feeds as any[]).map(feed => html`
                            <div class="feed-item" key=${feed.id}>
                                <div class="feed-item-info">
                                    <div class="feed-item-url" title=${feed.url}>${feed.url}</div>
                                    <div class="feed-item-meta">
                                        ${feed.calendar_name && html`<span class="feed-calendar">${feed.calendar_name}</span>`}
                                        <span>Every ${feed.refresh_interval_minutes} min</span>
                                        <span>\u00b7 Last: ${formatDate(feed.last_refreshed_at)}</span>
                                        ${!feed.enabled && html`<span class="feed-disabled">Disabled</span>`}
                                    </div>
                                    ${feed.last_error && html`<div class="feed-item-error">${feed.last_error}</div>`}
                                </div>
                                <div class="feed-item-actions">
                                    <button class="feed-action-btn" onClick=${() => handleRefresh(feed.id)}
                                            disabled=${refreshingId === feed.id}
                                            title="Refresh now">
                                        ${refreshingId === feed.id ? '\u23F3' : '\u21BB'}
                                    </button>
                                    <button class="feed-action-btn feed-delete-btn" onClick=${() => handleDelete(feed.id)}
                                            title="Delete">
                                        \u2715
                                    </button>
                                </div>
                            </div>
                        `)}
                    </div>
                `}
            `}
            ${showAdd ? html`
                <${AddFeedForm} onAdd=${handleAdd} onCancel=${() => setShowAdd(false)} />
            ` : html`
                <div class="dialog-actions">
                    <button onClick=${onClose}>Close</button>
                    <button onClick=${() => setShowAdd(true)}>Add Feed</button>
                </div>
            `}
        </dialog>
    ` as VNode;
}

interface AddFeedFormProps {
    onAdd: (data: any) => Promise<void>;
    onCancel: () => void;
}

function AddFeedForm({ onAdd, onCancel }: AddFeedFormProps): VNode | null {
    const [url, setUrl] = useState('');
    const [calendarName, setCalendarName] = useState('');
    const [calendarColor, setCalendarColor] = useState('dodgerblue');
    const [interval, setInterval] = useState(60);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    async function handleSubmit() {
        if (!url.trim()) { setError('URL is required'); return; }
        setLoading(true);
        setError('');
        try {
            const data: any = {
                url: url.trim(),
                calendar_name: calendarName.trim(),
                refresh_interval_minutes: Number(interval),
            };
            if (calendarName.trim()) data.calendar_color = calendarColor;
            await onAdd(data);
        } catch (err: any) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }

    return html`
        <div class="feed-add-form">
            <label>
                Feed URL
                <input type="url" value=${url} onInput=${(e: Event) => setUrl((e.target as HTMLInputElement).value)}
                       placeholder="https://calendar.google.com/..." />
            </label>
            <label>
                Calendar name (optional)
                <input type="text" value=${calendarName} onInput=${(e: Event) => setCalendarName((e.target as HTMLInputElement).value)}
                       placeholder="e.g. work, personal" maxlength="100" />
            </label>
            ${calendarName.trim() && html`
                <div class="color-picker">
                    <span>Calendar color</span>
                    <div class="color-options">
                        ${COLORS.map(c => html`
                            <div class="color-swatch ${calendarColor === c.name ? 'selected' : ''}"
                                 style="background-color: ${c.name}"
                                 title=${c.name}
                                 onClick=${() => setCalendarColor(c.name)} />
                        `)}
                    </div>
                </div>
            `}
            <label>
                Refresh interval (minutes)
                <input type="number" value=${interval} onInput=${(e: Event) => setInterval((e.target as HTMLInputElement).value as any)}
                       min="5" max="10080" />
            </label>
            ${error && html`<div class="feed-error">${error}</div>`}
            <div class="dialog-actions">
                <button onClick=${onCancel}>Cancel</button>
                <button onClick=${handleSubmit} disabled=${loading}>
                    ${loading ? 'Adding...' : 'Add Feed'}
                </button>
            </div>
        </div>
    ` as VNode;
}
