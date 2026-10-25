import { showToast } from './toast.js';

// The helper opens MyCal with a normal GET, so browser authentication works
// normally. Only a matching loopback opener can deliver the file afterwards.
export function receiveDesktopImport(onFile: (content: string) => void): () => void {
    const params = new URLSearchParams(location.hash.slice(1));
    const raw = params.get('desktop-import');
    if (!raw) return () => {};
    history.replaceState(history.state, '', location.pathname + location.search);
    let helper: URL;
    try {
        helper = new URL(raw);
    } catch {
        return () => {};
    }
    const nonce = helper.pathname.slice(1);
    const opener = window.opener;
    if (helper.protocol !== 'http:' || helper.hostname !== '127.0.0.1' || !helper.port ||
        helper.username || helper.password || helper.search || helper.hash ||
        !/^[A-Za-z0-9_-]{20,128}$/.test(nonce) || !opener) return () => {};

    const cleanup = () => {
        window.removeEventListener('message', onMessage);
        clearTimeout(timeout);
    };
    const onMessage = (event: MessageEvent) => {
        if (event.origin !== helper.origin || event.source !== opener ||
            event.data?.type !== 'mycal-desktop-file' || event.data.nonce !== nonce) return;
        const content = event.data.content;
        if (typeof content !== 'string' || new TextEncoder().encode(content).length > 10 * 1024 * 1024 ||
            !content.trimStart().startsWith('BEGIN:VCALENDAR')) return;
        cleanup();
        onFile(content);
    };
    const timeout = window.setTimeout(() => {
        cleanup();
        showToast('Could not receive the calendar file. Open it again from your desktop.', { error: true });
    }, 60000);
    window.addEventListener('message', onMessage);
    opener.postMessage({ type: 'mycal-desktop-ready', nonce }, helper.origin);
    return cleanup;
}
