import { html } from 'htm/preact';
import { useState, useRef, useEffect } from 'preact/hooks';
import { importEvents } from '../lib/api.js';

export function ImportForm({ onImported, onClose }) {
    const [mode, setMode] = useState('file');
    const [url, setUrl] = useState('');
    const [error, setError] = useState('');
    const [result, setResult] = useState(null);
    const [loading, setLoading] = useState(false);
    const dialogRef = useRef(null);
    const fileRef = useRef(null);

    useEffect(() => {
        if (dialogRef.current && !dialogRef.current.open) {
            dialogRef.current.showModal();
        }
    }, []);

    async function handleImport() {
        setError('');
        setResult(null);
        setLoading(true);
        try {
            let data;
            if (mode === 'file') {
                const file = fileRef.current?.files?.[0];
                if (!file) { setError('Please select a file'); setLoading(false); return; }
                const text = await file.text();
                data = { ics_content: text };
            } else {
                if (!url.trim()) { setError('Please enter a URL'); setLoading(false); return; }
                data = { url: url.trim() };
            }
            const res = await importEvents(data);
            setResult(res.imported);
            onImported();
        } catch (err) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    }

    return html`
        <dialog ref=${dialogRef} class="event-dialog import-dialog" onClose=${onClose}>
            <div class="dialog-header">
                <h2>Import Events</h2>
                <button class="close-btn" onClick=${onClose}>\u00d7</button>
            </div>
            <div class="import-tabs">
                <button class=${`import-tab ${mode === 'file' ? 'active' : ''}`}
                        onClick=${() => setMode('file')}>File</button>
                <button class=${`import-tab ${mode === 'url' ? 'active' : ''}`}
                        onClick=${() => setMode('url')}>URL</button>
            </div>
            ${error && html`<div class="error">${error}</div>`}
            ${result != null && html`<div class="import-success">Imported ${result} event${result !== 1 ? 's' : ''}.</div>`}
            ${mode === 'file' && html`
                <label>
                    iCalendar file (.ics)
                    <input ref=${fileRef} type="file" accept=".ics,.ical" />
                </label>
            `}
            ${mode === 'url' && html`
                <label>
                    iCalendar URL
                    <input type="url" value=${url} onInput=${e => setUrl(e.target.value)}
                           placeholder="https://calendar.google.com/..." />
                </label>
            `}
            <div class="dialog-actions">
                <button onClick=${onClose}>Cancel</button>
                <button type="submit" onClick=${handleImport} disabled=${loading}>
                    ${loading ? 'Importing...' : 'Import'}
                </button>
            </div>
        </dialog>
    `;
}
