// Covers demo/api.js: occurrence deletion must preserve EXDATE limits and be idempotent.
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';
import test from 'node:test';
import assert from 'node:assert/strict';

const context = vm.createContext({ TextEncoder, URL });
for (const file of ['model', 'sanitize', 'validate', 'seed', 'store', 'api']) {
    vm.runInContext(await readFile(new URL(`../static/demo/${file}.js`, import.meta.url), 'utf8'), context);
}
const run = (code) => vm.runInContext(code, context);
const instance = '2026-02-15T09:00:00Z';
for (const [name, exdates, expectedLength] of [
    ['duplicate', `  ${instance}  `, null],
    ['legacy oversized duplicate', (instance + ',').repeat(250) + instance, null],
    ['at limit', '2026-02-08T09:00:00Z,'.repeat(236) + '2026-02-08T09:00:00.00Z', 5000],
    ['over limit', '2026-02-08T09:00:00Z,'.repeat(236) + '2026-02-08T09:00:00.000Z', -1],
]) {
    test(name, () => {
        run(`var state = {events: [Object.assign(newEvent(), {id: 1, recurrence_freq: 'WEEKLY',
            exdates: ${JSON.stringify(exdates)}, updated_at: 'unchanged'})]};`);
        const call = () => run(`addExDate(state, 1, '${instance}')`);
        if (expectedLength === -1) assert.throws(call, /at most 5000/);
        else call();
        if (expectedLength === 5000) assert.equal(run('state.events[0].exdates.length'), 5000);
        else {
            assert.equal(run('state.events[0].exdates'), exdates);
            assert.equal(run('state.events[0].updated_at'), 'unchanged');
        }
    });
}
