// Covers demo/api.js: reject oversized calendar filters before accessing the store.
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';
import test from 'node:test';
import assert from 'node:assert/strict';

const context = vm.createContext({ TextEncoder, URL, URLSearchParams });
for (const file of ['model', 'sanitize', 'validate', 'seed', 'store', 'api']) {
    vm.runInContext(await readFile(new URL(`../static/demo/${file}.js`, import.meta.url), 'utf8'), context);
}
for (const [key, value, count, valid] of [
    ['calendar_id', '1', 100, true],
    ['calendar_id', '1', 101, false],
    ['calendar', 'a', 100, true],
    ['calendar', 'a', 101, false],
    ['calendar', 'a'.repeat(100), 1, true],
    ['calendar', 'a'.repeat(101), 1, false],
    ['calendar', '日'.repeat(100), 1, true],
    ['calendar', '日'.repeat(101), 1, false],
]) {
    test(`${key} length=${value.length} count=${count}`, () => {
        const params = new URLSearchParams(Array.from({ length: count }, () => [key, value]));
        const call = () => vm.runInContext(
            `parseCalendarIDsFromParams(new URLSearchParams(${JSON.stringify(params.toString())}))`, context);
        if (valid) assert.doesNotThrow(call);
        else assert.throws(call, err => err.status === 400);
    });
}
