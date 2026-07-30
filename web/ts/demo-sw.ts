// The demo service worker's entry point: lifecycle and request interception.
//
// Registered only by a demo build (see web/ts/demo-client.ts), it intercepts
// every same-scope /api/v1 request and answers it from browser-local storage,
// so the whole web UI runs with no server behind it. Requests for anything else
// — the app's own scripts, styles, and vendor bundles — are left alone and go
// to the network as usual. Unlike the sibling MyNotes worker there is no
// navigation fallback: MyCal has no client-side routing, so every navigation is
// the one page the host already serves.
//
// This is a classic worker script, compiled by web/ts/demo/tsconfig.json
// against lib.webworker: the emulated backend under web/ts/demo/ is pulled into
// this scope with importScripts, so the modules share one global scope rather
// than importing each other. That keeps the demo free of a bundler and of the
// DOM-typed app code, which cannot run here.

// The emulated backend, in dependency order: each file mirrors the Go package
// named in its header, and none of them imports another — they all land in this
// one scope.
importScripts(
    'demo/model.js',
    'demo/sanitize.js',
    'demo/validate.js',
    'demo/store.js',
    'demo/api.js',
);

/** `self`, typed. lib.webworker declares it as the generic worker scope. */
const sw = self as unknown as ServiceWorkerGlobalScope;

/** The path prefix every emulated endpoint sits under, relative to the scope. */
const API_PREFIX = 'api/v1';

/** The message a client sends to ask this worker to take control of it. */
const CLAIM_MESSAGE = 'mycal-demo:claim';

// ── Lifecycle ────────────────────────────────────────────────────────────────

// Take over as soon as installed, then claim the pages that are already open:
// the page registers the worker and waits for it, so without claiming, the very
// first load would find no controller and have to reload itself.
sw.addEventListener('install', () => {
    sw.skipWaiting();
});

sw.addEventListener('activate', (event) => {
    // Seeding here rather than on the first request means the store is ready
    // before the page asks for anything, and a failure shows up in the worker's
    // lifecycle rather than as one broken request.
    event.waitUntil(Promise.all([sw.clients.claim(), seedStore()]));
});

// Claiming on activate covers a first load, but not a hard reload (Ctrl-Shift-R):
// the browser loads that navigation with this worker bypassed, so the page ends
// up uncontrolled while the worker stays activated and activate never fires
// again. The page notices and asks here (see web/ts/demo-client.ts).
sw.addEventListener('message', (event) => {
    const data = event.data as { type?: string } | null;
    if (data === null || data.type !== CLAIM_MESSAGE) return;
    event.waitUntil(sw.clients.claim());
});

// ── Interception ─────────────────────────────────────────────────────────────

sw.addEventListener('fetch', (event) => {
    const path = apiPath(event.request.url);
    if (path === null) return; // scripts, styles, vendor bundles: straight to the network
    event.respondWith(handleApiRequest(path, event.request));
});

/**
 * The registration scope: the deployment's base path, since the page registers
 * the worker from its own directory. Every URL the demo resolves is relative to
 * this, so a bundle works at the origin root and under a path alike.
 */
function scopeURL(): URL {
    return new URL(sw.registration.scope);
}

/**
 * The API path of a request, or null when it is not an API request this worker
 * serves. Returns the part after the /api/v1 prefix with a leading slash, so
 * "https://host/cal/api/v1/events/1" under scope "/cal/" yields "/events/1".
 */
function apiPath(requestURL: string): string | null {
    const url = new URL(requestURL);
    const scope = scopeURL();
    if (url.origin !== scope.origin) return null;
    if (!url.pathname.startsWith(scope.pathname)) return null;
    const rest = url.pathname.slice(scope.pathname.length);
    if (rest !== API_PREFIX && !rest.startsWith(API_PREFIX + '/')) return null;
    return rest.slice(API_PREFIX.length);
}

