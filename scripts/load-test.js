import http from 'k6/http';
import exec from 'k6/execution';
import { check, fail } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

function positiveInteger(name, fallback, maximum) {
  const value = Number(__ENV[name] || fallback);
  if (!Number.isInteger(value) || value < 1 || value > maximum) {
    throw new Error(`${name} must be an integer between 1 and ${maximum}`);
  }
  return value;
}

const base = __ENV.BASE_URL || 'http://api:8080';
const rate = positiveInteger('LOAD_RATE', 100, 2000);
const seconds = positiveInteger('LOAD_SECONDS', 60, 600);
const conflictSeconds = positiveInteger('LOAD_CONFLICT_SECONDS', 20, 120);
// Multiples of ten give an exact 50/20/10/10/10 request mix and booking count.
if ((rate * seconds) % 10 !== 0) throw new Error('LOAD_RATE * LOAD_SECONDS must be divisible by 10');
const roomCount = 20;
const raceCases = 20;
const expectedBookings = rate * seconds / 10;
const days = Math.ceil(expectedBookings / (roomCount * 17));
const warmupSeconds = 5;
const drainSeconds = 11; // Longer than the per-request timeout; phases do not overlap.
const mixedRequests = new Counter('mixed_requests');
const mixedOK = new Rate('mixed_ok');
const requestMS = new Trend('request_ms', true);
const conflictOK = new Rate('conflict_ok');
const conflictRequests = new Counter('conflict_requests');
const raceOK = new Rate('race_ok');
const raceCount = new Counter('race_cases');
const expected2xx = http.expectedStatuses(200, 201);
const expected409 = http.expectedStatuses(409);
const weekdays = [1, 2, 3, 4, 5, 6, 7];

function arrival(duration, startTime, fn) {
  return {
    executor: 'constant-arrival-rate', rate, timeUnit: '1s',
    duration: `${duration}s`, startTime: `${startTime}s`, exec: fn,
    preAllocatedVUs: 50, maxVUs: 200, gracefulStop: '11s',
  };
}

const thresholds = {
  dropped_iterations: ['count==0'],
  // k6 can start the boundary iteration at t=duration; no under-delivery is allowed.
  mixed_requests: [`count>=${rate * seconds}`, `count<=${rate * seconds + 1}`],
  mixed_ok: ['rate>=0.999'],
  request_ms: ['p(95)<200'],
  conflict_ok: ['rate==1'],
  conflict_requests: [`count>=${rate * conflictSeconds}`, `count<=${rate * conflictSeconds + 1}`],
  race_ok: ['rate==1'],
  race_cases: [`count==${raceCases}`],
  'checks{phase:fixture}': ['rate==1'],
  'checks{phase:integrity}': ['rate==1'],
};
for (const operation of ['rooms', 'slots', 'schedule_get', 'booking', 'schedule_update']) {
  thresholds[`mixed_ok{operation:${operation}}`] = ['rate>=0.999'];
  thresholds[`request_ms{operation:${operation}}`] = ['p(95)<200'];
}

export const options = {
  setupTimeout: '120s', teardownTimeout: '60s',
  batch: 20, batchPerHost: 20,
  systemTags: ['status', 'method', 'name', 'scenario', 'expected_response'],
  summaryTrendStats: ['avg', 'med', 'p(95)', 'p(99)', 'max'],
  scenarios: {
    warmup: arrival(warmupSeconds, 0, 'warmup'),
    mixed: arrival(seconds, warmupSeconds + drainSeconds, 'mixed'),
    conflicts: arrival(conflictSeconds, warmupSeconds + seconds + 2 * drainSeconds, 'conflicts'),
    races: {
      executor: 'shared-iterations', vus: 1, iterations: raceCases,
      startTime: `${warmupSeconds + seconds + conflictSeconds + 3 * drainSeconds}s`,
      maxDuration: '60s', exec: 'race',
    },
  },
  thresholds,
};

function params(token, name, callback = expected2xx) {
  return {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    tags: { name }, timeout: '10s', redirects: 0, responseCallback: callback,
  };
}

function request(method, path, token, body, name, callback) {
  return http.request(method, base + path, body === null ? null : JSON.stringify(body), params(token, name, callback));
}

function json(response) {
  try {
    const value = response.json();
    return value && typeof value === 'object' ? value : {};
  } catch (_) { return {}; }
}

function requireThat(condition, message, phase = 'fixture') {
  if (!check(condition, { [message]: (ok) => Boolean(ok) }, { phase })) fail(message);
}

function fixture(method, path, token, body, expected) {
  const response = request(method, path, token, body, 'fixture');
  requireThat(response.status === expected, `${method} fixture status ${response.status}, expected ${expected}`);
  return json(response);
}

function schedule(startTime = '09:00', endTime = '18:00') {
  return { daysOfWeek: weekdays, startTime, endTime };
}

export function setup() {
  const admin = fixture('POST', '/dummyLogin', '', { role: 'admin' }, 200).token;
  const user = fixture('POST', '/dummyLogin', '', { role: 'user' }, 200).token;
  requireThat(admin && user, 'login tokens exist');
  const dates = Array.from({ length: days }, (_, index) => {
    const date = new Date();
    date.setUTCDate(date.getUTCDate() + index + 2);
    return date.toISOString().slice(0, 10);
  });
  const rooms = [];
  const slotsByRoom = [];
  const protectedBookings = [];
  const races = [];
  for (let i = 0; i < roomCount + raceCases; i++) {
    const room = fixture('POST', '/rooms/create', admin, { name: `load-${i}`, capacity: 6 }, 201).room;
    requireThat(room && room.id, 'created room has ID');
    const rule = fixture('POST', `/rooms/${room.id}/schedule/create`, admin, schedule(), 201).schedule;
    requireThat(rule && rule.id, 'created schedule has ID');
    const slots = [];
    for (const date of (i < roomCount ? dates : [dates[0]])) {
      const daySlots = fixture('GET', `/rooms/${room.id}/slots/list?date=${date}`, user, null, 200).slots;
      requireThat(Array.isArray(daySlots) && daySlots.length === 18, 'fixture contains 18 half-hour slots');
      slots.push(...daySlots);
    }
    // Explicit sorting makes fixture correctness independent of response order.
    slots.sort((a, b) => a.start.localeCompare(b.start));
    if (i >= roomCount) {
      races.push({ roomID: room.id, slotID: slots[0].id });
      continue;
    }
    const booking = fixture('POST', '/bookings/create', user, { slotId: slots[0].id }, 201).booking;
    requireThat(booking && booking.status === 'active', 'protected booking is active');
    protectedBookings.push(booking);
    rooms.push({ id: room.id, scheduleID: rule.id, protectedSlot: slots[0].id });
    slotsByRoom.push(slots.filter((slot) => slot.start.slice(11, 16) !== '09:00'));
  }
  // Round-robin fixtures keep booking and schedule updates on the same rooms.
  const bookingSlots = [];
  for (let i = 0; i < expectedBookings; i++) {
    bookingSlots.push(slotsByRoom[i % roomCount][Math.floor(i / roomCount)]);
  }
  requireThat(bookingSlots.every((slot) => slot && slot.id), 'enough unique booking slots');
  return { admin, user, dates, rooms, bookingSlots, protectedBookings, races };
}

export function warmup(data) {
  const room = data.rooms[exec.scenario.iterationInTest % roomCount];
  const response = request('GET', `/rooms/${room.id}/slots/list?date=${data.dates[0]}`, data.user, null, 'warmup');
  requireThat(response.status === 200, 'warmup succeeds');
}

export function mixed(data) {
  const index = exec.scenario.iterationInTest;
  const cycle = Math.floor(index / 10);
  const kind = index % 10;
  const room = data.rooms[cycle % roomCount];
  let method = 'GET', path, body = null, token = data.user, operation, valid;
  if (kind < 5) {
    path = '/rooms/list'; operation = 'rooms';
    valid = (r, b) => r.status === 200 && Array.isArray(b.rooms) && b.rooms.length === roomCount + raceCases;
  } else if (kind < 7) {
    path = `/rooms/${room.id}/slots/list?date=${data.dates[cycle % data.dates.length]}`; operation = 'slots';
    valid = (r, b) => r.status === 200 && Array.isArray(b.slots) && b.slots.every((slot) => slot.roomId === room.id);
  } else if (kind === 7) {
    path = `/rooms/${room.id}/schedule`; operation = 'schedule_get';
    valid = (r, b) => r.status === 200 && b.schedule && b.schedule.id === room.scheduleID && b.schedule.startTime === '09:00';
  } else if (kind === 8) {
    path = '/bookings/create'; method = 'POST'; operation = 'booking';
    body = { slotId: data.bookingSlots[cycle].id };
    valid = (r, b) => r.status === 201 && b.booking && b.booking.id && b.booking.slotId === body.slotId && b.booking.status === 'active';
  } else {
    path = `/rooms/${room.id}/schedule/update`; method = 'POST'; operation = 'schedule_update'; token = data.admin;
    body = schedule('09:00', Math.floor(cycle / roomCount) % 2 === 0 ? '19:00' : '18:00');
    valid = (r, b) => r.status === 200 && b.schedule && b.schedule.id === room.scheduleID && b.schedule.endTime === body.endTime;
  }
  const start = Date.now();
  const response = request(method, path, token, body, operation);
  requestMS.add(Date.now() - start, { operation });
  mixedRequests.add(1);
  // Malformed JSON/schema must record a failure, not skip the success metric.
  let ok = false;
  try { ok = Boolean(valid(response, json(response))); } catch (_) { /* Failed contract check. */ }
  mixedOK.add(ok, { operation });
}

export function conflicts(data) {
  // A single hot room deliberately contends on the same room/slot locks.
  const room = data.rooms[0];
  const booking = exec.scenario.iterationInTest % 2 === 0;
  const response = booking
    ? request('POST', '/bookings/create', data.user, { slotId: room.protectedSlot }, 'booking_conflict', expected409)
    : request('POST', `/rooms/${room.id}/schedule/update`, data.admin, schedule('10:00'), 'schedule_conflict', expected409);
  const body = json(response);
  conflictRequests.add(1);
  conflictOK.add(Boolean(response.status === 409 && body.error && body.error.code === (booking ? 'SLOT_ALREADY_BOOKED' : 'SCHEDULE_BOOKING_CONFLICT')));
}

export function race(data) {
  const item = data.races[exec.scenario.iterationInTest];
  const requests = Array.from({ length: 8 }, () => ({
    method: 'POST', url: base + '/bookings/create', body: JSON.stringify({ slotId: item.slotID }),
    params: params(data.user, 'race_booking', http.expectedStatuses(201, 400, 409)),
  }));
  requests.push({
    method: 'POST', url: `${base}/rooms/${item.roomID}/schedule/update`, body: JSON.stringify(schedule('10:00')),
    params: params(data.admin, 'race_update', http.expectedStatuses(200, 409)),
  });
  const responses = http.batch(requests);
  const update = responses.pop();
  let valid;
  if (update.status === 200) {
    valid = responses.every((r) => r.status === 400 && json(r).error?.code === 'INVALID_REQUEST');
  } else {
    valid = update.status === 409 && json(update).error?.code === 'SCHEDULE_BOOKING_CONFLICT'
      && responses.filter((r) => r.status === 201 && json(r).booking?.status === 'active').length === 1
      && responses.every((r) => r.status === 201 || (r.status === 409 && json(r).error?.code === 'SLOT_ALREADY_BOOKED'));
  }
  raceCount.add(1);
  raceOK.add(Boolean(valid));
}

export function teardown(data) {
  const response = request('GET', '/bookings/my', data.user, null, 'integrity');
  const bookings = json(response).bookings;
  requireThat(response.status === 200 && Array.isArray(bookings), 'bookings remain readable', 'integrity');
  const bySlot = new Map();
  for (const booking of bookings) {
    requireThat(booking.status === 'active' && !bySlot.has(booking.slotId), 'one active booking per slot', 'integrity');
    bySlot.set(booking.slotId, booking);
  }
  for (const protectedBooking of data.protectedBookings) {
    requireThat(bySlot.get(protectedBooking.slotId)?.id === protectedBooking.id, 'protected booking retains ID', 'integrity');
  }
  for (const slot of data.bookingSlots) {
    requireThat(bySlot.has(slot.id), 'successful workload bookings persist', 'integrity');
  }
  for (const room of data.rooms) {
    const current = request('GET', `/rooms/${room.id}/schedule`, data.user, null, 'integrity');
    const rule = json(current).schedule;
    requireThat(current.status === 200 && rule && rule.id === room.scheduleID && rule.startTime === '09:00'
      && ['18:00', '19:00'].includes(rule.endTime) && JSON.stringify(rule.daysOfWeek) === JSON.stringify(weekdays),
    'current schedule still accommodates workload bookings', 'integrity');
  }
  let raceBookings = 0;
  for (const item of data.races) {
    const current = request('GET', `/rooms/${item.roomID}/schedule`, data.user, null, 'integrity');
    const rule = json(current).schedule;
    const booked = bySlot.has(item.slotID);
    raceBookings += booked ? 1 : 0;
    requireThat(current.status === 200 && rule && rule.startTime === (booked ? '09:00' : '10:00'),
      'race has a serializable persisted outcome', 'integrity');
  }
  requireThat(bookings.length === roomCount + expectedBookings + raceBookings, 'no extra or missing bookings', 'integrity');
}

export function handleSummary(data) {
  const result = {
    measuredAt: new Date().toISOString(),
    config: { rate, seconds, conflictSeconds, roomCount, raceCases, expectedBookings, days },
    state: data.state, metrics: data.metrics, options: data.options,
  };
  const lines = Object.entries(data.metrics)
    .filter(([name]) => /^(mixed_|conflict_|race_|request_ms|dropped_iterations|checks\{)/.test(name))
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([name, metric]) => `${name}: ${JSON.stringify(metric.values)}`);
  return {
    '/tmp/summary.json': JSON.stringify(result, null, 2),
    stdout: '\n' + lines.join('\n') + '\n',
  };
}
