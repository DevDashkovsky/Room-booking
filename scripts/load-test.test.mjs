import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';
import { createContext, SourceTextModule, SyntheticModule } from 'node:vm';

// Execute the real k6 script with controlled HTTP responses. A schema exception
// must emit a failed sample; otherwise k6 can report a false-green success rate.
async function harness(response, iteration = 0) {
  const samples = new Map();
  const requests = [];
  class Metric {
    constructor(name) { this.name = name; }
    add(value) {
      const values = samples.get(this.name) || [];
      values.push(value);
      samples.set(this.name, values);
    }
  }
  const http = {
    expectedStatuses: () => ({}),
    request: (...args) => { requests.push(args); return response; },
    batch: () => response,
  };
  const context = createContext({ __ENV: {} });
  const dependencies = {
    'k6/http': { default: http },
    'k6/execution': { default: { scenario: { iterationInTest: iteration } } },
    k6: { check: (value) => Boolean(value), fail: (message) => { throw new Error(message); } },
    'k6/metrics': { Counter: Metric, Rate: Metric, Trend: Metric },
  };
  const module = new SourceTextModule(await readFile(new URL('./load-test.js', import.meta.url), 'utf8'), { context });
  await module.link((name) => {
    const exports = dependencies[name];
    assert.ok(exports, `unexpected dependency ${name}`);
    return new SyntheticModule(Object.keys(exports), function () {
      for (const [key, value] of Object.entries(exports)) this.setExport(key, value);
    }, { context });
  });
  await module.evaluate();
  const data = {
    admin: 'admin', user: 'user', dates: ['2030-01-01'],
    rooms: Array.from({ length: 20 }, (_, i) => ({ id: `room-${i}`, scheduleID: `schedule-${i}`, protectedSlot: `slot-${i}` })),
    bookingSlots: [{ id: 'slot-new' }], races: [{ roomID: 'room-race', slotID: 'slot-race' }],
  };
  return { api: module.namespace, samples, requests, data };
}

for (const [name, iteration, status, body] of [
  ['null room list', 0, 200, null],
  ['malformed slot entry', 5, 200, { slots: [null] }],
  ['missing schedule', 7, 200, {}],
  ['missing created booking', 8, 201, null],
  ['missing updated schedule', 9, 200, null],
  ['server failure', 0, 503, { error: { code: 'UNAVAILABLE' } }],
]) {
  test(`${name} records a failed measured request`, async () => {
    const h = await harness({ status, json: () => body }, iteration);
    h.api.mixed(h.data);
    assert.equal(h.requests.length, 1);
    assert.deepEqual(h.samples.get('mixed_requests'), [1]);
    assert.deepEqual(h.samples.get('mixed_ok'), [false]);
  });
}

test('invalid JSON records a failed request', async () => {
  const h = await harness({ status: 200, json() { throw new Error('invalid JSON'); } });
  h.api.mixed(h.data);
  assert.deepEqual(h.samples.get('mixed_ok'), [false]);
});

for (const iteration of [0, 1]) {
  test(`409 without error code fails conflict validation (${iteration})`, async () => {
    const h = await harness({ status: 409, json: () => null }, iteration);
    h.api.conflicts(h.data);
    assert.equal(h.requests.length, 1);
    assert.deepEqual(h.samples.get('conflict_requests'), [1]);
    assert.deepEqual(h.samples.get('conflict_ok'), [false]);
  });
}

test('a valid room response contributes exactly one success', async () => {
  const h = await harness({ status: 200, json: () => ({ rooms: Array.from({ length: 40 }, () => ({})) }) });
  h.api.mixed(h.data);
  assert.equal(h.requests.length, 1);
  assert.deepEqual(h.samples.get('mixed_ok'), [true]);
});

test('two successful bookings in a race fail even with expected HTTP statuses', async () => {
  const booked = { status: 201, json: () => ({ booking: { status: 'active' } }) };
  const conflict = { status: 409, json: () => ({ error: { code: 'SLOT_ALREADY_BOOKED' } }) };
  const update = { status: 409, json: () => ({ error: { code: 'SCHEDULE_BOOKING_CONFLICT' } }) };
  const h = await harness([booked, booked, ...Array(6).fill(conflict), update]);
  h.api.race(h.data);
  assert.deepEqual(h.samples.get('race_ok'), [false]);
});

test('summary omits fixture tokens', async () => {
  const h = await harness({});
  const output = h.api.handleSummary({ metrics: {}, setup_data: { admin: 'sensitive-token' } });
  assert.ok(!output['/tmp/summary.json'].includes('sensitive-token'));
});
