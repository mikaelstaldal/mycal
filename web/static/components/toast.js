import { html } from 'htm/preact';
import { useEffect } from 'preact/hooks';

export function Toast({ message, onDone }) {
    useEffect(() => {
        const id = setTimeout(onDone, 3000);
        return () => clearTimeout(id);
    }, []);

    return html`
        <div class="toast" onClick=${onDone}>${message}</div>
    `;
}
