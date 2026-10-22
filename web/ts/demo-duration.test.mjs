// Covers demo/model.js: duration grammar and arithmetic limits must match model.ParseDuration.
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';
import test from 'node:test';
import assert from 'node:assert/strict';

const context = vm.createContext({});
vm.runInContext(await readFile(new URL('../static/demo/model.js', import.meta.url), 'utf8'), context);
const parse = (input) => vm.runInContext(`parseDuration(${JSON.stringify(input)})`, context);

for (const [input, seconds] of [
    ['PT1H', 3600], ['PT30M', 1800], ['P1D', 86400], ['P2W', 1209600],
    ['P1DT2H30M5S', 95405], ['pt1h', 3600], ['PT0H1S', 1],
    ['PT9223372036S', 9223372036], ['P106751DT23H47M16S', 9223372036], ['PT00001S', 1],
]) {
    test(`duration ${input}`, () => assert.equal(parse(input), seconds * 1000));
}
for (const input of [
    '', '1H', 'P', 'PT', 'PD', 'P1', 'PT1H5', 'PTTT1H', 'P1DT', 'P1T2H',
    'P1DTT1H', 'PT1TH', 'PT1M1H', 'PT1H1H', 'P1D1D', 'P1W1D', 'P1WT1H',
    'PT1.5H', 'PT-1H', 'PT+1H', 'P1Y', 'P1M', 'PT1D', 'P0D', 'PT0S', 'PT1H\n',
    'PT9223372037S', 'P106752D', 'PT2562048H', 'PT153722868M', 'P15251W',
    'P106751DT23H47M17S', 'PT9223372036854775807H', 'PT18446744073709551616S',
]) {
    test(`reject duration ${JSON.stringify(input)}`, () => assert.throws(() => parse(input)));
}
