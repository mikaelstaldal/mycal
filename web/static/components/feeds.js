import { html } from 'htm/preact';
import { useState, useRef, useEffect } from 'preact/hooks';
import { listFeeds, createFeed, deleteFeed, refreshFeed } from '../lib/api.js';
import { COLORS } from '../lib/colors.js';

export function FeedsDialog({ onClose, onRefreshed }) {
    const [feeds, setFeeds] = useState([]);
    const [showAdd, setShowAdd] = useState(false);
    const [loading, setLoading] = useState(true);
    const [refreshingId, setRefreshingId] = useState(null);
    const [error, setError] = useState('');
    const dialogRef = useRef(null);

    useEffect(() => {
        if (dialogRef.current && !dialogRef.current.open) {
            dialogRef.current.showModal();
        }
        loadFeeds();
    }, []);

    async function loadFeeds() {
        try {
            setLoading(true);
            const data = await listFeeds();
            setFeeds(data);
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }

    async function handleDelete(id) {
        if (!confirm('Delete this feed subscription?')) return;
        try {
            await deleteFeed(id);
            await loadFeeds();
        } catch (err) {
            setError(err.message);
        }
    }

    async function handleRefresh(id) {
        setRefreshingId(id);
        try {
            await refreshFeed(id);
            await loadFeeds();
            if (onRefreshed) onRefreshed();
        } catch (err) {
            setError(err.message);
        } finally {
            setRefreshingId(null);
        }
    }

    async function handleAdd(data) {
        try {
            await createFeed(data);
            setShowAdd(false);
            await loadFeeds();
        } catch (err) {
            throw err;
        }
    }

    function formatDate(dateStr) {
        if (!dateStr) return 'Never';
        const d = new Date(dateStr);
        return d.toLocaleString(undefined, { dateStyle: 'short', timeStyle: 'short' });
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
                        ${feeds.map(feed => html`
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
    `;
}

function AddFeedForm({ onAdd, onCancel }) {
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
            const data = {
                url: url.trim(),
                calendar_name: calendarName.trim(),
                refresh_interval_minutes: Number(interval),
            };
            if (calendarName.trim()) data.calendar_color = calendarColor;
            await onAdd(data);
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }

    return html`
        <div class="feed-add-form">
            <label>
                Feed URL
                <input type="url" value=${url} onInput=${e => setUrl(e.target.value)}
                       placeholder="https://calendar.google.com/..." />
            </label>
            <label>
                Calendar name (optional)
                <input type="text" value=${calendarName} onInput=${e => setCalendarName(e.target.value)}
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
                <input type="number" value=${interval} onInput=${e => setInterval(e.target.value)}
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
    `;
}
