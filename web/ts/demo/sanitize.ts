// Mirrors internal/sanitize — the HTML allowlist applied to every write path.
//
// The Go server runs bluemonday with the policy in internal/sanitize/sanitize.go;
// this is a hand-written equivalent, because a service worker has no DOMParser
// and pulling in a sanitiser library would mean a bundler. It follows the same
// rules:
//
//   - only the elements the policy allows survive; anything else has its tags
//     dropped while its text is kept, except for the elements whose content
//     bluemonday discards outright (script, style, and friends);
//   - the only attributes kept are href and title on <a>, and href must parse to
//     an http:, https: or mailto: URL (or be relative);
//   - an absolute link gets target="_blank" and rel="noreferrer", matching
//     AddTargetBlankToFullyQualifiedLinks + RequireNoReferrerOnLinks;
//   - all text and attribute values are HTML-escaped on the way out, so nothing
//     in the input can create markup that was not allowed here.
//
// Being a demo does not make this optional: the sanitiser is what keeps a
// crafted event description from running script in the visitor's page, and
// AGENTS.md requires it on create and update alike.

/** Elements the policy allows, plus <a>, which AllowAttrs("href") implies. */
const ALLOWED_ELEMENTS = new Set([
    'a',
    'b', 'i', 'u', 'em', 'strong',
    'p', 'br', 'hr',
    'ul', 'ol', 'li',
    'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
    'blockquote', 'code', 'pre',
    'sub', 'sup',
    'div', 'span',
    'table', 'thead', 'tbody', 'tr', 'th', 'td',
]);

/** Allowed elements that never have content or a closing tag. */
const VOID_ELEMENTS = new Set(['br', 'hr']);

/**
 * Elements whose content is dropped along with their tags, as bluemonday's
 * default skipElementsContent does. Keeping the text of a <script> would turn
 * code into visible prose at best and, in an unlucky context, back into code.
 */
const SKIP_CONTENT_ELEMENTS = new Set([
    'frame', 'frameset', 'iframe', 'noembed', 'noframes', 'noscript',
    'object', 'script', 'servlet', 'style', 'title',
]);

/** The URL schemes the policy allows (AllowURLSchemes). */
const ALLOWED_URL_SCHEMES = new Set(['http', 'https', 'mailto']);

/**
 * Sanitises an HTML string, keeping only allowed tags and attributes. Mirrors
 * sanitize.HTML.
 */
function sanitizeHTML(input: string): string {
    if (input === '') return '';

    const out: string[] = [];
    // Allowed elements we have opened and still owe a closing tag.
    const open: string[] = [];
    // The element whose content we are discarding, and how deeply it nests.
    let skipping: string | null = null;
    let skipDepth = 0;

    let i = 0;
    while (i < input.length) {
        const lt = input.indexOf('<', i);
        if (lt < 0) {
            if (skipping === null) out.push(escapeTextNode(input.slice(i)));
            break;
        }
        if (lt > i && skipping === null) out.push(escapeTextNode(input.slice(i, lt)));

        const tag = readTag(input, lt);
        if (tag === null) {
            // A stray "<" that starts no tag: it is text, and escaping it is
            // what keeps it text.
            if (skipping === null) out.push('&lt;');
            i = lt + 1;
            continue;
        }
        i = tag.end;

        if (tag.kind === 'comment' || tag.kind === 'other') continue;

        if (skipping !== null) {
            if (tag.name === skipping) {
                if (tag.kind === 'start' && !tag.selfClosing) skipDepth++;
                else if (tag.kind === 'end' && --skipDepth === 0) skipping = null;
            }
            continue;
        }

        if (SKIP_CONTENT_ELEMENTS.has(tag.name)) {
            if (tag.kind === 'start' && !tag.selfClosing) {
                skipping = tag.name;
                skipDepth = 1;
            }
            continue;
        }

        if (!ALLOWED_ELEMENTS.has(tag.name)) continue; // tags dropped, text kept

        if (tag.kind === 'end') {
            const at = open.lastIndexOf(tag.name);
            if (at < 0) continue; // unmatched close tag: drop it
            // Close everything opened inside it, so the output stays balanced.
            for (let d = open.length - 1; d >= at; d--) out.push('</' + open[d] + '>');
            open.length = at;
            continue;
        }

        if (VOID_ELEMENTS.has(tag.name)) {
            out.push('<' + tag.name + '>');
            continue;
        }
        out.push('<' + tag.name + renderAttributes(tag.name, tag.attrs) + '>');
        if (!tag.selfClosing) open.push(tag.name);
    }

    for (let d = open.length - 1; d >= 0; d--) out.push('</' + open[d] + '>');
    return out.join('');
}

interface ParsedTag {
    kind: 'start' | 'end' | 'comment' | 'other';
    name: string;
    attrs: Array<{ name: string; value: string }>;
    selfClosing: boolean;
    /** Index just past the tag in the source. */
    end: number;
}

/**
 * Reads the tag starting at `<` at position `lt`, or returns null when what
 * follows is not a tag at all and the `<` is plain text.
 */
function readTag(s: string, lt: number): ParsedTag | null {
    if (s.startsWith('<!--', lt)) {
        const close = s.indexOf('-->', lt + 4);
        return { kind: 'comment', name: '', attrs: [], selfClosing: false, end: close < 0 ? s.length : close + 3 };
    }
    if (s.startsWith('<!', lt) || s.startsWith('<?', lt)) {
        const close = s.indexOf('>', lt);
        return { kind: 'other', name: '', attrs: [], selfClosing: false, end: close < 0 ? s.length : close + 1 };
    }

    let p = lt + 1;
    const end = p < s.length && s[p] === '/';
    if (end) p++;
    const nameStart = p;
    while (p < s.length && /[A-Za-z0-9]/.test(s[p])) p++;
    if (p === nameStart) return null;
    const name = s.slice(nameStart, p).toLowerCase();

    const attrs: Array<{ name: string; value: string }> = [];
    let selfClosing = false;
    while (p < s.length) {
        while (p < s.length && /[\s/]/.test(s[p])) {
            if (s[p] === '/' && s[p + 1] === '>') selfClosing = true;
            p++;
        }
        if (p >= s.length) break;
        if (s[p] === '>') {
            p++;
            break;
        }
        const attrNameStart = p;
        while (p < s.length && !/[\s/>=]/.test(s[p])) p++;
        const attrName = s.slice(attrNameStart, p).toLowerCase();
        while (p < s.length && /\s/.test(s[p])) p++;
        let value = '';
        if (s[p] === '=') {
            p++;
            while (p < s.length && /\s/.test(s[p])) p++;
            const quote = s[p];
            if (quote === '"' || quote === "'") {
                p++;
                const close = s.indexOf(quote, p);
                value = s.slice(p, close < 0 ? s.length : close);
                p = close < 0 ? s.length : close + 1;
            } else {
                const valueStart = p;
                while (p < s.length && !/[\s>]/.test(s[p])) p++;
                value = s.slice(valueStart, p);
            }
        }
        if (attrName !== '') attrs.push({ name: attrName, value: decodeEntities(value) });
    }
    return { kind: end ? 'end' : 'start', name, attrs, selfClosing, end: p };
}

/**
 * The attribute string for an allowed start tag: everything not on the
 * allowlist is dropped, and a link that leaves this origin is marked the way
 * the Go policy marks it.
 */
function renderAttributes(name: string, attrs: Array<{ name: string; value: string }>): string {
    if (name !== 'a') return ''; // no other element has an allowed attribute

    let href = '';
    let title = '';
    for (const attr of attrs) {
        if (attr.name === 'href') href = attr.value;
        else if (attr.name === 'title') title = attr.value;
    }

    let out = '';
    const url = sanitizeURL(href);
    if (url !== '') out += ' href="' + escapeAttr(url) + '"';
    if (title !== '') out += ' title="' + escapeAttr(title) + '"';
    if (url !== '' && isAbsoluteURL(url)) out += ' target="_blank" rel="noreferrer"';
    return out;
}

/**
 * An href the policy accepts, or "" when it does not. Relative URLs pass
 * (AllowRelativeURLs), absolute ones only with an allowed scheme.
 */
function sanitizeURL(raw: string): string {
    const url = raw.trim();
    if (url === '') return '';
    // Control characters are how "java\tscript:" gets past a naive scheme check.
    // eslint-disable-next-line no-control-regex
    if (/[\u0000-\u001f\u007f]/.test(url)) return '';
    const scheme = /^([A-Za-z][A-Za-z0-9+.-]*):/.exec(url);
    if (scheme !== null && !ALLOWED_URL_SCHEMES.has(scheme[1].toLowerCase())) return '';
    // A protocol-relative URL inherits the page's scheme, which is allowed, but
    // "<a href='//evil'>" is not something the demo content ever needs.
    if (scheme === null && url.startsWith('//')) return '';
    return url;
}

/** True for a URL with a scheme, i.e. one bluemonday calls fully qualified. */
function isAbsoluteURL(url: string): boolean {
    return /^[A-Za-z][A-Za-z0-9+.-]*:/.test(url);
}

/** Escapes text content, the way the Go tokenizer re-escapes what it emits. */
function escapeText(s: string): string {
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

/**
 * Escapes a text node: decode first, then escape, so that sanitising already
 * sanitised text is a no-op. Without the decode, every update of an event
 * would turn "&amp;" into "&amp;amp;" and the description would rot one save at
 * a time; the Go tokenizer decodes for the same reason.
 */
function escapeTextNode(s: string): string {
    return escapeText(decodeEntities(s));
}

/** Escapes an attribute value, quotes included, so it cannot end its own attribute. */
function escapeAttr(s: string): string {
    return escapeText(s).replace(/"/g, '&#34;').replace(/'/g, '&#39;');
}

/**
 * Decodes the few entities that matter when deciding whether an attribute value
 * is a permitted URL: "&#106;avascript:" must not slip through as a relative
 * URL and then be re-encoded into something a browser executes.
 */
function decodeEntities(s: string): string {
    return s
        .replace(/&#x([0-9a-fA-F]+);?/g, (_m, hex: string) => codePoint(parseInt(hex, 16)))
        .replace(/&#(\d+);?/g, (_m, dec: string) => codePoint(parseInt(dec, 10)))
        .replace(/&quot;?/gi, '"')
        .replace(/&apos;?/gi, "'")
        .replace(/&lt;?/gi, '<')
        .replace(/&gt;?/gi, '>')
        .replace(/&amp;?/gi, '&');
}

function codePoint(n: number): string {
    if (!Number.isFinite(n) || n < 0 || n > 0x10ffff) return '';
    return String.fromCodePoint(n);
}
