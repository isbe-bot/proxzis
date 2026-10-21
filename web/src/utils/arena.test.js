import test from 'node:test'
import assert from 'node:assert/strict'
import { STANCES, resolveClash, summarizeRounds, matchOutcome, createCredentials, validateRegistration } from './arena.js'
import { bots, matches, liveRounds, recordFor } from '../data/arena.js'

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
test('every archive match has seven valid rounds, consistent scores and outcome', () => {
  assert.equal(matches.length, 32)
  assert.equal(new Set(matches.map(m => m.id)).size, 32)
  for (const m of matches) {
    assert.notEqual(m.a.id, m.b.id)
    assert.equal(m.rounds.length, 7)
    assert.deepEqual(m.score, summarizeRounds(m.rounds))
    assert.equal(m.winner, matchOutcome(m))
    assert.ok(m.score.a + m.score.b <= 7)
    assert.ok(m.rounds.every(r => STANCES.includes(r.a) && STANCES.includes(r.b)))
    assert.ok(m.highlights.length >= 2)
  }
})
test('seed data exercises draws and wins by both bot categories and both sides', () => {
  assert.ok(matches.some(m => m.winner === 'draw'))
  assert.ok(matches.some(m => m.winner === 'a'))
  assert.ok(matches.some(m => m.winner === 'b'))
  assert.ok(matches.some(m => m.winner !== 'draw' && m[m.winner].house))
  assert.ok(matches.some(m => m.winner !== 'draw' && !m[m.winner].house))
})
test('all tactical tendency percentages total 100', () => {
  for (const bot of bots) {
    assert.equal(bot.tactics.reduce((a, b) => a + b, 0), 100)
    assert.ok(bot.tactics.every(n => n >= 0 && n <= 100))
  }
})
test('all lifetime records and earnings reconcile with the complete archive', () => {
  let totalPlayed = 0, wins = 0, losses = 0, draws = 0, earnings = 0
  for (const bot of bots) {
    const log = matches.filter(m => m.a.id === bot.id || m.b.id === bot.id)
    const r = recordFor(bot.id)
    assert.equal(r.played, log.length)
    assert.equal(r.wins + r.losses + r.draws, r.played)
    assert.equal(r.wins, log.filter(m => m.winner !== 'draw' && m[m.winner].id === bot.id).length)
    assert.equal(r.losses, log.filter(m => m.winner !== 'draw' && m[m.winner].id !== bot.id).length)
    assert.equal(r.draws, log.filter(m => m.winner === 'draw').length)
    assert.equal(r.earnings, log.reduce((sum, m) => sum + (m.winner === 'draw' ? 0 : m[m.winner].id === bot.id ? m.stake : -m.stake), 0))
    totalPlayed += r.played; wins += r.wins; losses += r.losses; draws += r.draws; earnings += r.earnings
  }
  assert.equal(totalPlayed, matches.length * 2)
  assert.equal(wins, losses)
  assert.equal(draws, matches.filter(m => m.winner === 'draw').length * 2)
  assert.equal(earnings, 0)
  assert.deepEqual(recordFor('new-local'), { wins: 0, losses: 0, draws: 0, earnings: 0, played: 0 })
})
test('live exhibition ends at the claimed four-to-two score', () => {
  assert.deepEqual(summarizeRounds(liveRounds), { a: 2, b: 4 })
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
