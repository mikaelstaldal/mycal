import { html } from 'htm/preact';
import { useState, useRef, useEffect } from 'preact/hooks';
import { importEvents, importSingleEvent } from '../lib/api.js';

export function ImportForm({ onImported, onClose }) {
    const [importMode, setImportMode] = useState('single');
    const [sourceMode, setSourceMode] = useState('file');
    const [url, setUrl] = useState('');
    const [error, setError] = useState('');
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
        setLoading(true);
        try {
            let data;
            if (sourceMode === 'file') {
                const file = fileRef.current?.files?.[0];
                if (!file) { setError('Please select a file'); setLoading(false); return; }
                const text = await file.text();
                data = { ics_content: text };
            } else {
                if (!url.trim()) { setError('Please enter a URL'); setLoading(false); return; }
                data = { url: url.trim() };
            }
            let message;
            if (importMode === 'single') {
                await importSingleEvent(data);
                message = 'Event imported successfully.';
            } else {
                const res = await importEvents(data);
                message = `Imported ${res.imported} event${res.imported !== 1 ? 's' : ''}.`;
            }
            onImported(message);
        } catch (err) {
            setError(err.message);
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
                <button class=${`import-tab ${importMode === 'single' ? 'active' : ''}`}
                        onClick=${() => { setImportMode('single'); setError(''); }}>Single Event</button>
                <button class=${`import-tab ${importMode === 'bulk' ? 'active' : ''}`}
                        onClick=${() => { setImportMode('bulk'); setError(''); }}>Bulk Import</button>
            </div>
            <div class="import-tabs">
                <button class=${`import-tab ${sourceMode === 'file' ? 'active' : ''}`}
                        onClick=${() => { setSourceMode('file'); setError(''); }}>File</button>
                <button class=${`import-tab ${sourceMode === 'url' ? 'active' : ''}`}
                        onClick=${() => { setSourceMode('url'); setError(''); }}>URL</button>
            </div>
            ${error && html`<div class="error">${error}</div>`}
            ${sourceMode === 'file' && html`
                <label>
                    iCalendar file (.ics)
                    <input ref=${fileRef} type="file" accept=".ics,.ical" />
                </label>
            `}
            ${sourceMode === 'url' && html`
                <label>
                    iCalendar URL
                    <input type="url" value=${url} onInput=${e => setUrl(e.target.value)}
                           placeholder="https://calendar.google.com/..." />
                </label>
            `}
            ${importMode === 'single' && html`
                <div class="import-hint">The file or URL must contain exactly one event.</div>
            `}
            <div class="dialog-actions">
                <button onClick=${onClose}>Cancel</button>
                <button type="submit" onClick=${handleImport} disabled=${loading}>
                    ${loading ? 'Importing...' : importMode === 'single' ? 'Import Event' : 'Import All'}
                </button>
            </div>
        </dialog>
    `;
}
