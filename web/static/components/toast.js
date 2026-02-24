import { html } from 'htm/preact';
import { useEffect } from 'preact/hooks';

export function Toast({ message, isError, onDone }) {
    useEffect(() => {
        const id = setTimeout(onDone, isError ? 5000 : 3000);
        return () => clearTimeout(id);
    }, []);

    return html`
        <div class=${`toast${isError ? ' toast-error' : ''}`} onClick=${onDone}>${message}</div>
    `;
}
