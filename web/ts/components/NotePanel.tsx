import type { VNode } from 'preact';
import { useEffect, useRef, useState } from 'preact/hooks';
import { getNote, noteUrl, renderHostUrl } from '../util/mynotes.js';
import type { Note } from '../util/mynotes.js';

// Contract of the MyNotes render kit's host page (its web/ts/render/host.ts).
// The kit is loaded in an iframe from the MyNotes deployment and driven through
// this API, so the MyNotes Markdown dialect — wikilinks, callouts, AsciiMath,
// Mermaid, Lucide icons — is rendered by MyNotes' own pipeline and MyCal never
// parses Markdown itself.
interface RenderHost {
    render(markdown: string): Promise<void>;
    setTheme(theme: 'light' | 'dark', vars?: Record<string, string>): void;
}

// Schemes an anchor in a rendered note may keep. Everything else is downgraded
// to plain text, mirroring the rest of the app's link handling.
const SAFE_SCHEMES = ['http:', 'https:', 'mailto:'];

interface NotePanelProps {
    mynotesUrl: string;
    slug: string;
    darkMode: boolean;
}

function getRenderHost(frame: HTMLIFrameElement): RenderHost | null {
    try {
        const win = frame.contentWindow as (Window & { MyNotesRender?: RenderHost }) | null;
        return win?.MyNotesRender ?? null;
    } catch {
        return null; // MyNotes on another origin: the kit cannot be driven
    }
}

// The render kit is content-free and carries no <base>, so wikilinks and
// embedded images come out root-relative (/notes/…, /api/v1/icons/…) — its host
// is expected to resolve them. MyCal does that here, against the MyNotes base
// URL, and sends every link to a new tab so nothing navigates inside the frame.
function resolveLinks(doc: Document, mynotesUrl: string): void {
    for (const a of Array.from(doc.querySelectorAll('a[href]'))) {
        const raw = a.getAttribute('href') ?? '';
        if (raw.startsWith('/')) {
            a.setAttribute('href', mynotesUrl + raw);
        }
        let scheme: string;
        try {
            scheme = new URL((a as HTMLAnchorElement).href).protocol;
        } catch {
            scheme = '';
        }
        if (!SAFE_SCHEMES.includes(scheme)) {
            a.removeAttribute('href');
            continue;
        }
        a.setAttribute('target', '_blank');
        a.setAttribute('rel', 'noopener noreferrer');
    }
    for (const img of Array.from(doc.querySelectorAll('img[src]'))) {
        const raw = img.getAttribute('src') ?? '';
        if (raw.startsWith('/')) {
            img.setAttribute('src', mynotesUrl + raw);
        }
    }
}

// The kit accepts overrides for note.css's own custom properties so an embedded
// note blends into the surrounding chrome. Feed it MyCal's dialog colours,
// re-read on every theme switch since the kit keeps overrides until replaced.
function chromeVars(): Record<string, string> {
    const style = getComputedStyle(document.documentElement);
    const vars: Record<string, string> = {};
    const bg = style.getPropertyValue('--surface').trim();
    const fg = style.getPropertyValue('--text').trim();
    if (bg) vars['--bg'] = bg;
    if (fg) vars['--fg'] = fg;
    return vars;
}

export function NotePanel({ mynotesUrl, slug, darkMode }: NotePanelProps): VNode {
    const frameRef = useRef<HTMLIFrameElement | null>(null);
    const [note, setNote] = useState<Note | null>(null);
    const [error, setError] = useState('');
    const [hostReady, setHostReady] = useState(false);

    useEffect(() => {
        const controller = new AbortController();
        setNote(null);
        setError('');
        getNote(mynotesUrl, slug, controller.signal)
            .then(setNote)
            .catch((err: Error) => {
                if (err.name !== 'AbortError') setError(err.message || 'Could not load note');
            });
        return () => controller.abort();
    }, [mynotesUrl, slug]);

    // Push the note into the render kit once both the note and the host page are
    // available. The frame is sized by the dialog's flex column (see
    // .note-panel-frame), so nothing here has to follow the content height —
    // notes taller than the frame scroll inside it.
    useEffect(() => {
        const frame = frameRef.current;
        if (!frame || !hostReady || !note) return;
        const host = getRenderHost(frame);
        if (!host) {
            setError('Could not reach the MyNotes renderer');
            return;
        }

        let cancelled = false;

        host.setTheme(darkMode ? 'dark' : 'light', chromeVars());
        void host.render(note.content).then(() => {
            if (cancelled) return;
            const doc = frame.contentDocument;
            if (!doc) return;
            resolveLinks(doc, mynotesUrl);
        });

        return () => {
            cancelled = true;
        };
    }, [hostReady, note, darkMode, mynotesUrl]);

    return (
        <div class="note-panel">
            <div class="note-panel-header">
                <span class="detail-label">Note:</span>
                <a href={noteUrl(mynotesUrl, slug)} target="_blank" rel="noopener noreferrer" class="url-link">
                    {note ? note.title : slug} &#x2197;
                </a>
            </div>
            {error ? (
                <div class="note-panel-error">{error}</div>
            ) : (
                <iframe ref={frameRef} class="note-panel-frame" title="Note content"
                        src={renderHostUrl(mynotesUrl)}
                        onLoad={() => setHostReady(true)} />
            )}
        </div>
    );
}
