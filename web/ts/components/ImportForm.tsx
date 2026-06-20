import type { VNode } from 'preact';
import { useState, useRef, useEffect } from 'preact/hooks';
import { api } from '../api/client.js';
import { showToast } from '../util/toast.js';
import { eventStartStr } from '../util/date-utils.js';

interface ImportSingleFormProps {
    onImported: () => void;
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
            if (!file) { showToast('Please select a file.', { error: true }); return; }
            input = await file.text();
        } else {
            if (!url.trim()) { showToast('Please enter a URL.', { error: true }); return; }
            input = url.trim();
        }
        setLoading(true);
        try {
            const event = await api.import.single(input, calendarName.trim());
            const startStr = eventStartStr(event);
            const date = startStr ? new Date(startStr).toLocaleDateString() : '';
            showToast(`Event imported successfully.${date ? ' Start: ' + date : ''}`);
            onImported();
        } catch (err: any) {
            showToast(err.message, { error: true });
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
                       placeholder="e.g. work, personal" maxlength={100} />
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
    onImported: () => void;
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
            if (!file) { showToast('Please select a file.', { error: true }); return; }
            input = await file.text();
        } else {
            if (!url.trim()) { showToast('Please enter a URL.', { error: true }); return; }
            input = url.trim();
        }
        setLoading(true);
        try {
            const res = await api.import.bulk(input, calendarName.trim());
            showToast(`Imported ${res.imported} event${res.imported !== 1 ? 's' : ''}.`);
            onImported();
        } catch (err: any) {
            showToast(err.message, { error: true });
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
                       placeholder="e.g. work, personal" maxlength={100} />
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
