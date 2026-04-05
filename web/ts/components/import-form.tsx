import { h } from 'preact';
import type { VNode } from 'preact';
import { useState, useRef, useEffect } from 'preact/hooks';
import { importEvents, importSingleEvent } from '../lib/api.js';

interface ImportSingleFormProps {
    onImported: (message: string, isError?: boolean) => void;
    onClose: () => void;
}

export function ImportSingleForm({ onImported, onClose }: ImportSingleFormProps): VNode | null {
    const [sourceMode, setSourceMode] = useState('file');
    const [url, setUrl] = useState('');
    const [calendarName, setCalendarName] = useState('');
    const [loading, setLoading] = useState(false);
    const dialogRef = useRef<HTMLDialogElement | null>(null);
    const fileRef = useRef<HTMLInputElement | null>(null);

    useEffect(() => {
        if (dialogRef.current && !dialogRef.current.open) {
            dialogRef.current.showModal();
        }
    }, []);

    async function handleImport() {
        let input: string;
        if (sourceMode === 'file') {
            const file = fileRef.current?.files?.[0];
            if (!file) { onImported('Please select a file.', true); return; }
            input = await file.text();
        } else {
            if (!url.trim()) { onImported('Please enter a URL.', true); return; }
            input = url.trim();
        }
        setLoading(true);
        try {
            const event = await importSingleEvent(input, calendarName.trim());
            const date = event.start_time ? new Date(event.start_time).toLocaleDateString() : '';
            onImported(`Event imported successfully.${date ? ' Start: ' + date : ''}`);
        } catch (err: any) {
            onImported(err.message, true);
        } finally {
            setLoading(false);
        }
    }

    return (
        <dialog ref={dialogRef} class="event-dialog import-dialog" onClose={onClose}>
            <div class="dialog-header">
                <h2>Import Event</h2>
                <button class="close-btn" onClick={onClose}>&#xd7;</button>
            </div>
            <div class="import-tabs">
                <button class={`import-tab ${sourceMode === 'file' ? 'active' : ''}`}
                        onClick={() => setSourceMode('file')}>File</button>
                <button class={`import-tab ${sourceMode === 'url' ? 'active' : ''}`}
                        onClick={() => setSourceMode('url')}>URL</button>
            </div>
            {sourceMode === 'file' && (
                <label>
                    iCalendar file (.ics)
                    <input ref={fileRef} type="file" accept=".ics,.ical" />
                </label>
            )}
            {sourceMode === 'url' && (
                <label>
                    iCalendar URL
                    <input type="url" value={url} onInput={(e: Event) => setUrl((e.target as HTMLInputElement).value)}
                           placeholder="https://calendar.google.com/..." />
                </label>
            )}
            <label>
                Calendar name (optional)
                <input type="text" value={calendarName} onInput={(e: Event) => setCalendarName((e.target as HTMLInputElement).value)}
                       placeholder="e.g. work, personal" maxlength="100" />
            </label>
            <div class="import-hint">The file or URL must contain exactly one event.</div>
            <div class="dialog-actions">
                <button onClick={onClose}>Cancel</button>
                <button type="submit" onClick={handleImport} disabled={loading}>
                    {loading ? <span class="spinner"></span> : null}{loading ? ' Importing...' : 'Import Event'}
                </button>
            </div>
        </dialog>
    );
}

interface ImportBulkFormProps {
    onImported: (message: string, isError?: boolean) => void;
    onClose: () => void;
}

export function ImportBulkForm({ onImported, onClose }: ImportBulkFormProps): VNode | null {
    const [sourceMode, setSourceMode] = useState('file');
    const [url, setUrl] = useState('');
    const [calendarName, setCalendarName] = useState('');
    const [loading, setLoading] = useState(false);
    const dialogRef = useRef<HTMLDialogElement | null>(null);
    const fileRef = useRef<HTMLInputElement | null>(null);

    useEffect(() => {
        if (dialogRef.current && !dialogRef.current.open) {
            dialogRef.current.showModal();
        }
    }, []);

    async function handleImport() {
        let input: string;
        if (sourceMode === 'file') {
            const file = fileRef.current?.files?.[0];
            if (!file) { onImported('Please select a file.', true); return; }
            input = await file.text();
        } else {
            if (!url.trim()) { onImported('Please enter a URL.', true); return; }
            input = url.trim();
        }
        setLoading(true);
        try {
            const res = await importEvents(input, calendarName.trim());
            onImported(`Imported ${res.imported} event${res.imported !== 1 ? 's' : ''}.`);
        } catch (err: any) {
            onImported(err.message, true);
        } finally {
            setLoading(false);
        }
    }

    return (
        <dialog ref={dialogRef} class="event-dialog import-dialog" onClose={onClose}>
            <div class="dialog-header">
                <h2>Bulk Import</h2>
                <button class="close-btn" onClick={onClose}>&#xd7;</button>
            </div>
            <div class="import-tabs">
                <button class={`import-tab ${sourceMode === 'file' ? 'active' : ''}`}
                        onClick={() => setSourceMode('file')}>File</button>
                <button class={`import-tab ${sourceMode === 'url' ? 'active' : ''}`}
                        onClick={() => setSourceMode('url')}>URL</button>
            </div>
            {sourceMode === 'file' && (
                <label>
                    iCalendar file (.ics)
                    <input ref={fileRef} type="file" accept=".ics,.ical" />
                </label>
            )}
            {sourceMode === 'url' && (
                <label>
                    iCalendar URL
                    <input type="url" value={url} onInput={(e: Event) => setUrl((e.target as HTMLInputElement).value)}
                           placeholder="https://calendar.google.com/..." />
                </label>
            )}
            <label>
                Calendar name (optional)
                <input type="text" value={calendarName} onInput={(e: Event) => setCalendarName((e.target as HTMLInputElement).value)}
                       placeholder="e.g. work, personal" maxlength="100" />
            </label>
            <div class="dialog-actions">
                <button onClick={onClose}>Cancel</button>
                <button type="submit" onClick={handleImport} disabled={loading}>
                    {loading ? <span class="spinner"></span> : null}{loading ? ' Importing...' : 'Import All'}
                </button>
            </div>
        </dialog>
    );
}
