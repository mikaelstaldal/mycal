// Covers demo/validate.js and demo/api.js: merged spans must match server validation.
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';
import test from 'node:test';
import assert from 'node:assert/strict';

const context = vm.createContext({ TextEncoder, URL });
for (const file of ['model', 'sanitize', 'validate', 'seed', 'store', 'api']) {
    vm.runInContext(await readFile(new URL(`../static/demo/${file}.js`, import.meta.url), 'utf8'), context);
}
const run = (code) => vm.runInContext(code, context);

for (const allDay of [false, true]) {
    for (const duration of ['P366D', 'P367D', 'P36500D']) {
        test(`create all_day=${allDay} duration=${duration}`, () => {
            const req = { title: 'event', all_day: allDay, duration,
                ...(allDay ? { start_date: '2026-01-01' } : { start_time: '2026-01-01T00:00:00Z' }) };
            const call = () => run(`validateCreateEventRequest(${JSON.stringify(req)})`);
            if (duration === 'P366D') assert.doesNotThrow(call);
            else assert.throws(call, /366 days/);
        });
    }
}
for (const patch of [
    { end_time: '9999-12-31T00:00:00Z' },
    { start_time: '1500-01-01T00:00:00Z' },
    { end_time: '2028-01-01T00:00:00Z' },
    { end_time: '2000-01-01T00:00:00Z' },
    { end_time: '2026-01-01T00:00:00Z' },
    { duration: 'P367D' },
    { duration: 'P366D' },
    { start_time: '2028-01-01T00:00:00Z', duration: 'PT1H' },
]) {
    for (const override of [false, true]) {
        test(`${override ? 'override' : 'update'} ${JSON.stringify(patch)}`, () => {
            run(`var state = { events: [Object.assign(newEvent(), {
                id: 1, title: 'event', start_time: '2026-01-01T00:00:00Z',
                end_time: '2026-01-01T01:00:00Z', recurrence_freq: 'DAILY'
            })], calendars: [], preferences: {} };`);
            const call = () => run(override
                ? `createOrUpdateOverride(state, 1, '2026-01-01T00:00:00Z', ${JSON.stringify(patch)})`
                : `updateEvent(state, 1, ${JSON.stringify(patch)})`);
            if (patch.duration === 'P366D' || patch.duration === 'PT1H') assert.doesNotThrow(call);
            else assert.throws(call, /date must be|366 days|after start_time/);
        });
    }
}
