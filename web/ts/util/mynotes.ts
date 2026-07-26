// Client for the MyNotes REST API, used to link a note to an event and to show
// that note in the event dialog.
//
// MyNotes is assumed to be deployed on the same origin as MyCal (e.g.
// https://example.com/mynotes next to https://example.com/cal) behind the same
// basic-auth realm, so requests carry the browser's existing credentials and
// need no CORS or token exchange. The base URL comes from the server-injected
// window.__serverConfig.mynotesUrl or the user's Settings override.
//
// These types mirror the MyNotes OpenAPI schemas (Note, NoteSummary); MyCal does
// not generate code from that spec, so only the fields used here are declared.

export interface NoteSummary {
    slug: string;
    title: string;
}

export interface Note extends NoteSummary {
    /** Verbatim Markdown, rendered through the MyNotes render kit — never parsed here. */
    content: string;
    updated_at: string;
}

interface NoteList {
    total: number;
    notes: NoteSummary[];
}

async function get<T>(url: string, signal?: AbortSignal): Promise<T> {
    const res = await fetch(url, { credentials: 'include', signal });
    if (!res.ok) {
        let msg = `MyNotes request failed (${res.status})`;
        try {
            const body = await res.json() as { error?: string };
            if (body.error) msg = body.error;
        } catch {
            // non-JSON error body; keep the status message
        }
        throw new Error(msg);
    }
    return await res.json() as T;
}

/** Search notes by title prefix, for the note picker's autocomplete. */
export async function searchNotes(baseUrl: string, query: string, signal?: AbortSignal): Promise<NoteSummary[]> {
    const url = `${baseUrl}/api/v1/notes?q=${encodeURIComponent(query)}&titlePrefix=true&limit=10`;
    const list = await get<NoteList>(url, signal);
    return list.notes ?? [];
}

/** Fetch a single note, including its Markdown content. */
export async function getNote(baseUrl: string, slug: string, signal?: AbortSignal): Promise<Note> {
    return await get<Note>(`${baseUrl}/api/v1/notes/${encodeURIComponent(slug)}`, signal);
}

/** URL of a note in the MyNotes web UI. */
export function noteUrl(baseUrl: string, slug: string): string {
    return `${baseUrl}/notes/${encodeURIComponent(slug)}`;
}

/** URL of the render kit's host page (see the MyNotes render kit, web/ts/render/host.ts). */
export function renderHostUrl(baseUrl: string): string {
    return `${baseUrl}/render/`;
}
