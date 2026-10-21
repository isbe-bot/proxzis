import test from 'node:test'
import assert from 'node:assert/strict'
import { bots, matches, recordFor } from '../data/arena.js'
import { validateRegistration } from './arena.js'
import { botFacts, humanTier, recordLabel, settlement, dateLabel, handOutcome, DEMO_OPENING_BALANCE } from './presentation.js'

test('human tiers have explicit, readable boundary values', () => {
  for (const [rating, tier] of [[1500, 'Silver'], [1549, 'Silver'], [1550, 'Gold'], [1599, 'Gold'], [1600, 'Diamond'], [1649, 'Diamond'], [1650, 'Master']]) assert.equal(humanTier(rating), tier)
})
test('records use strict ASCII W-L-D hyphens, including zero records', () => {
  assert.equal(recordLabel({ wins: 12, losses: 3, draws: 0 }), '12-3-0')
  assert.equal(botFacts({ id: 'new', rating: 1500 }).record, '0-0-0')
})
test('settlements distinguish gross payouts, net wins/losses and refunds', () => {
  const match = { stake: 20, winner: 'a' }
  assert.deepEqual(settlement(match, 'a'), { wager: 20, payout: 40, net: 20, result: 'Won' })
  assert.deepEqual(settlement(match, 'b'), { wager: 20, payout: 0, net: -20, result: 'Lost' })
  assert.deepEqual(settlement({ ...match, winner: 'draw' }, 'a'), { wager: 20, payout: 20, net: 0, result: 'Draw · stake returned' })
})
test('all demo balances, gross wins, counts and latest plays reconcile with archive', () => {
  let net = 0, gross = 0
  for (const bot of bots) {
    const facts = botFacts(bot)
    const record = recordFor(bot.id)
    const log = matches.filter(m => m.a.id === bot.id || m.b.id === bot.id)
    assert.equal(facts.balance, DEMO_OPENING_BALANCE + record.earnings)
    assert.equal(facts.played, record.played)
    assert.equal(facts.earnings, record.earnings)
    assert.equal(facts.lastPlayed, log.map(m => m.date).sort().at(-1))
    assert.equal(facts.marblesWon, log.reduce((sum, m) => sum + (m.winner !== 'draw' && m[m.winner].id === bot.id ? m.stake * 2 : 0), 0))
    assert.match(facts.record, /^\d+-\d+-\d+$/)
    net += facts.earnings; gross += facts.marblesWon
  }
  assert.equal(net, 0)
  assert.equal(gross, matches.filter(m => m.winner !== 'draw').reduce((sum, m) => sum + 2 * m.stake, 0))
})
test('unplayed local recruits have an explicitly seeded balance and no latest play', () => {
  const facts = botFacts({ id: 'new-local', rating: 1500 })
  assert.equal(facts.balance, 1000)
  assert.equal(facts.marblesWon, 0)
  assert.equal(facts.lastPlayed, null)
  assert.equal(dateLabel(facts.lastPlayed), 'Not played yet')
})
test('latest play is independent of archive ordering and draws do not count as gross wins', () => {
  const facts = botFacts(bots[0], [
    { a: bots[0], b: bots[1], winner: 'a', stake: 10, date: '2026-01-01T00:00:00.000Z' },
    { a: bots[1], b: bots[0], winner: 'draw', stake: 50, date: '2026-03-01T00:00:00.000Z' },
    { a: bots[1], b: bots[0], winner: 'a', stake: 20, date: '2026-02-01T00:00:00.000Z' },
  ])
  assert.equal(facts.record, '1-1-1')
  assert.equal(facts.lastPlayed, '2026-03-01T00:00:00.000Z')
  assert.equal(facts.marblesWon, 20)
  assert.equal(facts.earnings, -10)
})
test('four-argument registration validates owner email for both forms', () => {
  const args = ['New Bot', 'A valid persona description.', bots]
  assert.deepEqual(validateRegistration(args[0], 'owner@example.com', args[1], args[2]), {})
  for (const email of ['', 'invalid', 'x@']) assert.ok(validateRegistration(args[0], email, args[1], args[2]).email)
  assert.match(validateRegistration(args[0], ' OWNER@EXAMPLE.COM ', args[1], [{ name: 'Other', email: 'owner@example.com' }]).email, /already/)
})

test('hand outcomes are per-side and never leak before the scored result', () => {
  for (const phase of ['thinking', 'reveal', 'unknown']) {
    for (const winner of ['a', 'b', 'draw']) for (const side of ['a', 'b']) assert.equal(handOutcome(phase, winner, side), null)
  }
  for (const phase of ['result', 'complete']) {
    assert.equal(handOutcome(phase, 'a', 'a'), 'win')
    assert.equal(handOutcome(phase, 'a', 'b'), 'lose')
    assert.equal(handOutcome(phase, 'b', 'a'), 'lose')
    assert.equal(handOutcome(phase, 'b', 'b'), 'win')
    for (const side of ['a', 'b']) assert.equal(handOutcome(phase, 'draw', side), 'draw')
  }
})
