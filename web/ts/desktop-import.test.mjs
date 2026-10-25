// Tests the compiled URL-fragment decoder: UTF-8, bounds, malformed input,
// and removal of the payload before any UI/API work.
import test from 'node:test';
import assert from 'node:assert/strict';

const { decodeDesktopImport, takeDesktopImport } = await import('../static/util/desktop-import.js');
const content = 'BEGIN:VCALENDAR\r\nSUMMARY:Möte 🗓️ </textarea><script>alert(1)</script>\r\nEND:VCALENDAR';
const encode = text => Buffer.from(text).toString('base64url');

function setup(fragment) {
    let replaced;
    globalThis.location = { hash: fragment, pathname: '/mycal/', search: '?view=week' };
    globalThis.history = { state: { keep: true }, replaceState: (...args) => { replaced = args; } };
    return () => replaced;
}

test('base64url preserves Unicode and untrusted markup as plain text', () => {
    assert.equal(decodeDesktopImport(encode(content)), content);
});

test('malformed, non-UTF8, non-calendar and oversized payloads are rejected', () => {
    for (const raw of ['', 'a', '!bad', 'YWJj=', 'x'.repeat(16385), encode('https://example.com/file.ics'),
        Buffer.from([0xff]).toString('base64url'), encode('BEGIN:VCALENDAR' + 'x'.repeat(12 * 1024))]) {
        assert.throws(() => decodeDesktopImport(raw), undefined, raw.slice(0, 30));
    }
});

test('payload is consumed and removed from the current history entry', () => {
    const replacement = setup('#import=' + encode(content));
    assert.equal(takeDesktopImport(), content);
    assert.deepEqual(replacement(), [{ keep: true }, '', '/mycal/?view=week']);
});

test('malformed imports are also removed before showing an error', t => {
    t.mock.method(globalThis, 'setTimeout', () => 0);
    const replacement = setup('#import=bad!');
    assert.equal(takeDesktopImport(), null);
    assert.deepEqual(replacement(), [{ keep: true }, '', '/mycal/?view=week']);
});

test('unrelated navigation fragments are left alone', () => {
    const replacement = setup('#subscribe=https://example.com/feed');
    assert.equal(takeDesktopImport(), null);
    assert.equal(replacement(), undefined);
});
