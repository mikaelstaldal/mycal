// Covers demo/validate.js: OpenAPI event string bounds also apply in the offline backend.
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';
import test from 'node:test';
import assert from 'node:assert/strict';

const context = vm.createContext({ TextEncoder, URL });
for (const file of ['model', 'validate']) {
    vm.runInContext(await readFile(new URL(`../static/demo/${file}.js`, import.meta.url), 'utf8'), context);
}
for (const field of ['color', 'recurrence_until', 'duration']) {
    for (const operation of ['Create', 'Update']) {
        test(`${operation} rejects oversized ${field}`, () => {
            const req = { [field]: 'x'.repeat(101) };
            assert.throws(() => vm.runInContext(
                `validate${operation}EventRequest(${JSON.stringify(req)})`, context,
            ), /at most 100 characters/);
        });
    }
}

for (const field of ['color', 'recurrence_until', 'duration']) {
    test(`string limit accepts exactly 100 characters for ${field}`, () => {
        assert.doesNotThrow(() => vm.runInContext(
            `validateEventStringLengths(${JSON.stringify({ [field]: 'x'.repeat(100) })})`, context,
        ));
    });
}
for (const field of ['recurrence_by_day', 'recurrence_by_monthday', 'recurrence_by_month', 'exdates', 'rdates']) {
    test(`update rejects oversized ${field}`, () => {
        assert.throws(() => vm.runInContext(
            `validateUpdateEventRequest(${JSON.stringify({ [field]: 'x'.repeat(5001) })})`, context,
        ), /at most 5000 characters/);
    });
}
test('update enforces recurrence interval boundary', () => {
    assert.doesNotThrow(() => vm.runInContext('validateUpdateEventRequest({recurrence_interval:999})', context));
    assert.throws(() => vm.runInContext('validateUpdateEventRequest({recurrence_interval:1000})', context), /at most 999/);
});
