import { showToast } from './toast.js';

// Keep in sync with cmd/open's URL/file caps. Windows' shell has a lower URL cap.
const MAX_ENCODED_LENGTH = 16 * 1024;
const MAX_FILE_BYTES = 12 * 1024;

export function decodeDesktopImport(encoded: string): string {
    if (!encoded || encoded.length > MAX_ENCODED_LENGTH ||
        !/^[A-Za-z0-9_-]+$/.test(encoded) || encoded.length % 4 === 1) {
        throw new Error('Invalid or oversized desktop import. Use the Import Event file picker.');
    }
    const binary = atob(encoded.replace(/-/g, '+').replace(/_/g, '/'));
    if (binary.length > MAX_FILE_BYTES) throw new Error('Desktop import is too large. Use the Import Event file picker.');
    const content = new TextDecoder('utf-8', { fatal: true }).decode(Uint8Array.from(binary, char => char.charCodeAt(0)));
    if (!content.trimStart().startsWith('BEGIN:VCALENDAR')) throw new Error('The desktop import is not an iCalendar file.');
    return content;
}

// Consume before rendering or starting API requests. Never put file contents in
// query parameters, storage, links, or subsequent history entries.
export function takeDesktopImport(): string | null {
    const params = new URLSearchParams(location.hash.slice(1));
    if (!params.has('import')) return null;
    const encoded = params.get('import')!;
    history.replaceState(history.state, '', location.pathname + location.search);
    try {
        return decodeDesktopImport(encoded);
    } catch {
        showToast('Could not read the desktop import. Use the Import Event file picker.', { error: true });
        return null;
    }
}
