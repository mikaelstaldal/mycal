// Deployment configuration injected by the server, not by the user.
//
// When MyCal is deployed under a path (-public-url https://example.com/cal),
// main.go splices an inline <script> into index.html that sets
// window.__serverConfig with the base URLs of the sibling MyMail and MyNotes
// instances assumed at /mymail and /mynotes on the same origin. A demo build
// sets `demo` instead. Nothing is injected when none of that applies, so the
// object is absent far more often than not.
//
// The values originate from the server, but they are read back out of a global
// that page script could have overwritten, so they are re-validated here before
// being used to build a request URL.

declare global {
    interface Window {
        __serverConfig?: { mymailUrl?: string; mynotesUrl?: string; demo?: boolean };
    }
}

// Absolute http(s) URL with no characters that could smuggle a second URL
// component past the concatenation in the callers.
const ABSOLUTE_HTTP_URL = /^https?:\/\/[^\s"'<>\\]+$/;

function siblingUrl(url: string | undefined): string {
    if (typeof url !== 'string' || !ABSOLUTE_HTTP_URL.test(url)) return '';
    return url.replace(/\/+$/, '');
}

// The MyMail base URL without a trailing slash, or '' when the integration is
// not configured. Callers treat '' as "hide the email action".
export function mymailUrl(): string {
    return siblingUrl(window.__serverConfig?.mymailUrl);
}

// The MyNotes base URL without a trailing slash, or '' when the integration is
// not configured. Callers treat '' as "hide the note panel".
export function mynotesUrl(): string {
    return siblingUrl(window.__serverConfig?.mynotesUrl);
}

// Whether this is the backend-less demo build (mycal -demo-server, or a bundle
// written by -demo-bundle). The app then starts a service worker that answers
// the REST API from browser-local storage; see web/ts/demo-client.ts.
export function isDemo(): boolean {
    return window.__serverConfig?.demo === true;
}
