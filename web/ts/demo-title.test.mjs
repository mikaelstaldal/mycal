// Covers demo/api.js: titles must survive writes as text rather than HTML entities.
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';
import test from 'node:test';
import assert from 'node:assert/strict';

const context = vm.createContext({ TextEncoder, URL, Request, Response });
for (const file of ['model', 'sanitize', 'validate', 'seed', 'store', 'api']) {
    vm.runInContext(await readFile(new URL(`../static/demo/${file}.js`, import.meta.url), 'utf8'), context);
}
const run = (code) => vm.runInContext(code, context);

test('title punctuation survives create, update and recurrence override', async () => {
    const title = 'A & B: 3 < 5 > 2';
    run(`var state = { events: [], calendars: [], preferences: {} }`);
    run(`withStore = async (_mode, fn) => fn(state)`);
    await run(`apiV1EventsPost({ request: new Request('http://localhost/api/v1/events', {
        method: 'POST', body: JSON.stringify({ title: ${JSON.stringify(title)},
        start_time: '2026-01-01T10:00:00Z', end_time: '2026-01-01T11:00:00Z',
        recurrence_freq: 'DAILY' }) }) })`);
    assert.equal(run('state.events[0].title'), title);
    run(`updateEvent(state, state.events[0].id, { title: ${JSON.stringify(title)} })`);
    assert.equal(run('state.events[0].title'), title);
    run(`createOrUpdateOverride(state, state.events[0].id, '2026-01-02T10:00:00Z',
        { title: ${JSON.stringify(title)} })`);
    assert.equal(run('state.events[1].title'), title);
});

test('named entities decode like the Go sanitizer', () => {
    assert.equal(run(`sanitizeText('Caf&eacute; &nbsp; &CounterClockwiseContourIntegral;')`), 'Café \u00a0 ∳');
    assert.equal(run(`sanitizeText('&amp;lt;')`), '&lt;');
});

test('unknown entities never resolve inherited object properties', () => {
    for (const text of ['Q&constructor', 'x &toString y', '&hasOwnProperty']) {
        assert.equal(run(`sanitizeText(${JSON.stringify(text)})`), text);
    }
});

test('legacy entities decode prefixes in text but respect attribute boundaries', () => {
    assert.equal(run(`sanitizeText('x &lt5 y &copy2026 a&ampb')`), 'x <5 y ©2026 a&b');
    assert.equal(run(`decodeEntities('a&ampb', true)`), 'a&ampb');
    assert.equal(run(`decodeEntities('a&amp=b', true)`), 'a&amp=b');
    assert.equal(run(`decodeEntities('a&amp;b', true)`), 'a&b');
});
