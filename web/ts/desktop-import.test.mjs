// Tests the compiled desktop handoff's origin, source-window and nonce checks.
// These checks keep unrelated pages from placing files in an import dialog.
import test from 'node:test';
import assert from 'node:assert/strict';

const { receiveDesktopImport } = await import('../static/util/desktop-import.js');
const nonce = 'abcdefghijklmnopqrstuvwx';
const content = 'BEGIN:VCALENDAR\r\nEND:VCALENDAR';

function setup(raw = `http://127.0.0.1:54321/${nonce}`) {
    const listeners = new Map();
    const sent = [];
    const files = [];
    const opener = { postMessage: (...args) => sent.push(args) };
    globalThis.location = { hash: '#desktop-import=' + encodeURIComponent(raw), pathname: '/', search: '' };
    globalThis.history = { state: null, replaceState() {} };
    globalThis.window = {
        opener, setTimeout,
        addEventListener: (type, fn) => listeners.set(type, fn),
        removeEventListener: type => listeners.delete(type),
    };
    const cleanup = receiveDesktopImport(file => files.push(file));
    const deliver = (overrides = {}) => listeners.get('message')?.({
        origin: 'http://127.0.0.1:54321', source: opener,
        data: { type: 'mycal-desktop-file', nonce, content }, ...overrides,
    });
    return { cleanup, sent, files, deliver, listeners };
}

test('only the matching helper window, origin and nonce can deliver a file', () => {
    const handoff = setup();
    try {
        assert.deepEqual(handoff.sent, [[{ type: 'mycal-desktop-ready', nonce }, 'http://127.0.0.1:54321']]);
        handoff.deliver({ origin: 'https://unrelated.example' });
        handoff.deliver({ source: {} });
        handoff.deliver({ data: { type: 'mycal-desktop-file', nonce: 'wrong', content } });
        handoff.deliver({ data: { type: 'mycal-desktop-file', nonce, content: 'https://example.com/file.ics' } });
        handoff.deliver({ data: { type: 'mycal-desktop-file', nonce, content: 'BEGIN:VCALENDAR' + 'x'.repeat(10 * 1024 * 1024) } });
        assert.deepEqual(handoff.files, []);
        handoff.deliver();
        assert.deepEqual(handoff.files, [content]);
        assert.equal(handoff.listeners.size, 0);
        handoff.deliver();
        assert.deepEqual(handoff.files, [content]);
    } finally {
        handoff.cleanup();
    }
});

test('a navigation fragment cannot nominate a non-loopback or malformed helper', () => {
    for (const raw of [
        `https://example.com/${nonce}`, `http://127.0.0.1.example.com:54321/${nonce}`,
        `http://127.0.0.1/${nonce}`, 'http://127.0.0.1:54321/short',
        `http://user@127.0.0.1:54321/${nonce}`, `http://127.0.0.1:54321/${nonce}?extra=1`,
    ]) {
        const handoff = setup(raw);
        try {
            assert.deepEqual(handoff.sent, [], raw);
            assert.equal(handoff.listeners.size, 0, raw);
        } finally {
            handoff.cleanup();
        }
    }
});
