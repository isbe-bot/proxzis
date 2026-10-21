import test from 'node:test'
import assert from 'node:assert/strict'
import { createLiveClock, liveTimeline, SHAKE_MS, REVEAL_AT_MS, RESULT_AT_MS, ROUND_MS, FINAL_HOLD_MS } from './liveClock.js'
import { BEATS, resolveClash } from './arena.js'

// Boundaries are intentional: there is no fourth fist shake at Shoot.
test('exactly three complete fist shakes, then simultaneous Shoot/reveal', () => {
  for (const [index, cue] of ['Rock', 'Paper', 'Scissors'].entries()) {
    const start = index * SHAKE_MS
    assert.equal(liveTimeline(start).cue, cue)
    assert.equal(liveTimeline(start).shakeIndex, index)
    assert.equal(liveTimeline(start).handOffset, 0)
    assert.equal(liveTimeline(start + SHAKE_MS / 2).handOffset, -18)
    assert.ok(Math.abs(liveTimeline(start + SHAKE_MS - .001).handOffset) < .001)
    assert.equal(liveTimeline(start + SHAKE_MS - .001).phase, 'thinking')
  }
  const shoot = liveTimeline(REVEAL_AT_MS)
  assert.equal(shoot.cue, 'Shoot!')
  assert.equal(shoot.phase, 'reveal')
  assert.equal(shoot.shakeIndex, 3)
  assert.equal(shoot.handOffset, 0)
  assert.equal(shoot.resolved, 0)
  assert.equal(liveTimeline(RESULT_AT_MS - 1).phase, 'reveal')
  assert.equal(liveTimeline(RESULT_AT_MS).phase, 'result')
  assert.equal(liveTimeline(RESULT_AT_MS).resolved, 1)
})

test('rounds, final hold and replay have precise boundaries', () => {
  const next = liveTimeline(ROUND_MS)
  assert.equal(next.roundIndex, 1)
  assert.equal(next.cue, 'Rock')
  assert.equal(next.resolved, 1)
  assert.equal(next.remaining, ROUND_MS)
  assert.equal(liveTimeline(7 * ROUND_MS - 1).phase, 'result')
  const final = liveTimeline(7 * ROUND_MS)
  assert.equal(final.phase, 'complete')
  assert.equal(final.resolved, 7)
  assert.equal(final.roundIndex, 6)
  assert.equal(final.remaining, 0)
  const cycle = 7 * ROUND_MS + FINAL_HOLD_MS
  assert.equal(liveTimeline(cycle - 1).phase, 'complete')
  assert.deepEqual(liveTimeline(cycle), { ...liveTimeline(0), bout: 2 })
  assert.deepEqual(liveTimeline(cycle * 10 + 1377), { ...liveTimeline(1377), bout: 11 })
})

test('pause freezes fractional progress and resume excludes the real pause duration', () => {
  let now = 100
  const clock = createLiveClock(() => now)
  now += 725.5
  clock.pause()
  const frozen = liveTimeline(clock.elapsed())
  now += 120000
  clock.pause() // Idempotent, even after a long background pause.
  assert.deepEqual(liveTimeline(clock.elapsed()), frozen)
  clock.resume()
  now += 50
  clock.resume()
  assert.equal(clock.elapsed(), 775.5)
  now += 324.5
  assert.equal(liveTimeline(clock.elapsed()).cue, 'Shoot!')
})

test('missed/background frames catch up without incrementing ticks', () => {
  let now = 0
  const clock = createLiveClock(() => now)
  now = 10750 // No intervening samples.
  assert.deepEqual(liveTimeline(clock.elapsed()), liveTimeline(10750))
  now = 43200 * 4 + 700
  assert.equal(liveTimeline(clock.elapsed()).bout, 5)
  assert.equal(liveTimeline(clock.elapsed()).cue, 'Scissors')
})

test('restart clears running or paused progress and starts exactly on Rock', () => {
  let now = 0
  const clock = createLiveClock(() => now)
  for (const pause of [false, true]) {
    now += 10000
    if (pause) clock.pause()
    clock.restart()
    assert.equal(clock.elapsed(), 0)
    assert.equal(liveTimeline(clock.elapsed()).cue, 'Rock')
    now += 300
    assert.equal(clock.elapsed(), 300)
  }
})

test('timeline normalizes invalid elapsed values and supports actual round count', () => {
  for (const value of [-1, NaN, Infinity]) assert.deepEqual(liveTimeline(value), liveTimeline(0))
  assert.equal(liveTimeline(ROUND_MS, 1).phase, 'complete')
  assert.throws(() => liveTimeline(0, 0), RangeError)
})

test('game rules follow strike > parry > feint > strike', () => {
  assert.deepEqual(BEATS, { strike: 'parry', parry: 'feint', feint: 'strike' })
  for (const [stance, beaten] of Object.entries(BEATS)) {
    assert.equal(resolveClash(stance, beaten), 'a')
    assert.equal(resolveClash(beaten, stance), 'b')
    assert.equal(resolveClash(stance, stance), 'draw')
  }
})
