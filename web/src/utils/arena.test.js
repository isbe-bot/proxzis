import test from 'node:test'
import assert from 'node:assert/strict'
import { STANCES, resolveClash, createCredentials, validateRegistration } from './arena.js'

const bots = [{ name: 'Sovereign-1', email: 'sovereign@example.com' }, { name: 'Blinky' }]

test('all nine stance pairings follow the specified Parley cycle', () => {
  // STANCES = ['strike', 'feint', 'parry']
  // strike beats parry (b), feint beats strike (b for feint vs strike, so strike vs feint is 'b')
  // strike vs strike = draw, strike vs feint = b, strike vs parry = a
  // feint vs strike = a, feint vs feint = draw, feint vs parry = b
  // parry vs strike = b, parry vs feint = a, parry vs parry = draw
  const expected = [
    ['draw', 'b', 'a'],
    ['a', 'draw', 'b'],
    ['b', 'a', 'draw']
  ]
  STANCES.forEach((a, i) => STANCES.forEach((b, j) => assert.equal(resolveClash(a, b), expected[i][j])))
  assert.throws(() => resolveClash('unknown', 'strike'))
})
test('UUIDv7 encodes the timestamp, version and RFC variant', () => {
  const now = 1792498800123
  const { id, bearer } = createCredentials(now)
  assert.match(id, /^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/)
  assert.equal(parseInt(id.replaceAll('-', '').slice(0, 12), 16), now)
  assert.match(bearer, /^bg_[0-9a-f]{64}$/)
  assert.ok(!bearer.includes(id.replaceAll('-', '')))
})
test('credential generations use independent secure-random ID and secret draws', () => {
  const original = globalThis.crypto
  const calls = []
  Object.defineProperty(globalThis, 'crypto', { configurable: true, value: { getRandomValues(array) { calls.push(array.length); return array.fill(calls.length) } } })
  try {
    const result = createCredentials(1792498800123)
    assert.deepEqual(calls, [16, 32])
    assert.equal(result.bearer, `bg_${'02'.repeat(32)}`)
    assert.ok(result.id.endsWith('010101010101'))
  } finally { Object.defineProperty(globalThis, 'crypto', { configurable: true, value: original }) }
})
test('same-timestamp UUIDs and bearer values do not repeat across 1000 samples', () => {
  const ids = new Set(), bearers = new Set()
  for (let i = 0; i < 1000; i++) { const c = createCredentials(1792498800123); ids.add(c.id); bearers.add(c.bearer) }
  assert.equal(ids.size, 1000); assert.equal(bearers.size, 1000)
})
test('credential generation fails honestly if secure randomness is unavailable', () => {
  const original = globalThis.crypto
  Object.defineProperty(globalThis, 'crypto', { configurable: true, value: undefined })
  try { assert.throws(() => createCredentials(), /Secure random generation is unavailable/) }
  finally { Object.defineProperty(globalThis, 'crypto', { configurable: true, value: original }) }
})
test('registration validates trimmed lengths, Unicode names and allowed characters', () => {
  const persona = 'Reads the room and counters.'
  for (const name of [' ab ', 'a'.repeat(25), '_alpha', 'Bad<bot>', '   ']) assert.ok(validateRegistration(name, 'owner@example.com', persona, bots).name)
  for (const name of [' abc ', 'a'.repeat(24), 'Écho-零 2', 'New_Bot']) assert.deepEqual(validateRegistration(name, 'owner@example.com', persona, bots), {})
  for (const text of ['', ' '.repeat(13), 'a'.repeat(11), 'a'.repeat(181)]) assert.ok(validateRegistration('Newbot', 'owner@example.com', text, bots).persona)
  for (const text of ['a'.repeat(12), 'a'.repeat(180)]) assert.deepEqual(validateRegistration('Newbot', 'owner@example.com', text, bots), {})
})
test('registration rejects case-insensitive duplicates including local recruits', () => {
  assert.match(validateRegistration('  bLiNkY  ', 'owner@example.com', 'A valid persona.', bots).name, /reserved/)
  assert.match(validateRegistration('new bot', 'owner@example.com', 'A valid persona.', [...bots, { name: 'New Bot' }]).name, /already/)
})
