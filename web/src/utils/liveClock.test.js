import test from 'node:test'
import assert from 'node:assert/strict'
import { createLiveClock, liveTimeline, DROP_MS, BUMP_MS, BUMP_COUNT, SHOOT_AT_MS, EXIT_MS, ROUND_MS, FINAL_HOLD_MS, OFFSTAGE_LIFT } from './liveClock.js'
import { BEATS, resolveClash } from './arena.js'

const CYCLE = 7 * ROUND_MS + FINAL_HOLD_MS

// 1, 2, Shoot: the drop lands as the first beat, one more pound follows, and
// the next downstroke lands as the move. There is no third rock contact.
test('rock fists drop in, pound the floor twice, then Shoot lands as the move', () => {
  const start = liveTimeline(0)
  assert.equal(start.phase, 'drop')
  assert.equal(start.lift, OFFSTAGE_LIFT)
  assert.equal(start.beat, 0)
  assert.equal(start.resolved, 0)

  let rockContacts = 0
  for (let t = 0; t < SHOOT_AT_MS; t++) {
    const frame = liveTimeline(t)
    assert.notEqual(frame.phase, 'result', `move shown early at ${t} ms`)
    assert.equal(frame.resolved, 0)
    if (frame.lift === 0) rockContacts++
  }
  assert.equal(BUMP_COUNT, 2)
  assert.equal(rockContacts, 2)

  for (let beat = 1; beat <= BUMP_COUNT; beat++) {
    const contact = DROP_MS + (beat - 1) * BUMP_MS
    assert.equal(liveTimeline(contact).lift, 0)
    assert.equal(liveTimeline(contact).beat, beat)
    assert.equal(liveTimeline(contact + BUMP_MS / 2).lift, 1)
    assert.ok(liveTimeline(contact + BUMP_MS - .001).lift < .001)
  }

  const shoot = liveTimeline(SHOOT_AT_MS)
  assert.equal(shoot.phase, 'result')
  assert.equal(shoot.lift, 0)
  assert.equal(shoot.beat, 3)
  assert.equal(shoot.resolved, 1)
})

test('the result holds briefly, then both hands lift out before the next drop', () => {
  assert.ok(SHOOT_AT_MS <= 800, 'the count stays quick')
  assert.ok(ROUND_MS <= 3000, 'rounds move on quickly')
  for (let t = SHOOT_AT_MS; t < ROUND_MS - EXIT_MS; t += 10) assert.equal(liveTimeline(t).lift, 0)
  assert.ok(ROUND_MS - EXIT_MS - SHOOT_AT_MS >= 1200, 'long enough to read both moves and the result')
  assert.equal(liveTimeline(ROUND_MS - 1).phase, 'result')
  assert.ok(liveTimeline(ROUND_MS - 1).lift > OFFSTAGE_LIFT * 0.9)
})

test('rounds, final hold and replay have precise boundaries', () => {
  const next = liveTimeline(ROUND_MS)
  assert.equal(next.roundIndex, 1)
  assert.equal(next.phase, 'drop')
  assert.equal(next.resolved, 1)
  assert.equal(next.remaining, ROUND_MS)
  assert.equal(liveTimeline(7 * ROUND_MS - 1).phase, 'result')
  const final = liveTimeline(7 * ROUND_MS)
  assert.equal(final.phase, 'complete')
  assert.equal(final.lift, 0)
  assert.equal(final.resolved, 7)
  assert.equal(final.roundIndex, 6)
  assert.equal(final.remaining, 0)
  assert.equal(liveTimeline(CYCLE - 1).phase, 'complete')
  assert.deepEqual(liveTimeline(CYCLE), { ...liveTimeline(0), bout: 2 })
  assert.deepEqual(liveTimeline(CYCLE * 10 + 1377), { ...liveTimeline(1377), bout: 11 })
})

test('pause freezes fractional progress and resume excludes the real pause duration', () => {
  let now = 100
  const clock = createLiveClock(() => now)
  now += 725.5
  clock.pause()
  const frozen = liveTimeline(clock.elapsed())
  assert.equal(frozen.phase, 'count')
  assert.equal(frozen.beat, 2)
  now += 120000
  clock.pause() // Idempotent, even after a long background pause.
  assert.deepEqual(liveTimeline(clock.elapsed()), frozen)
  clock.resume()
  now += 50
  clock.resume()
  assert.equal(clock.elapsed(), 775.5)
  now += 324.5
  assert.equal(liveTimeline(clock.elapsed()).phase, 'result')
})

test('missed/background frames catch up without incrementing ticks', () => {
  let now = 0
  const clock = createLiveClock(() => now)
  now = 10750 // No intervening samples.
  assert.deepEqual(liveTimeline(clock.elapsed()), liveTimeline(10750))
  now = CYCLE * 4 + 700
  assert.equal(liveTimeline(clock.elapsed()).bout, 5)
  assert.equal(liveTimeline(clock.elapsed()).beat, 2)
})

test('restart clears running or paused progress and starts exactly on the drop', () => {
  let now = 0
  const clock = createLiveClock(() => now)
  for (const pause of [false, true]) {
    now += 10000
    if (pause) clock.pause()
    clock.restart()
    assert.equal(clock.elapsed(), 0)
    assert.equal(liveTimeline(clock.elapsed()).phase, 'drop')
    now += 300
    assert.equal(clock.elapsed(), 300)
  }
})

test('timeline normalizes invalid elapsed values and supports actual round count', () => {
  for (const value of [-1, NaN, Infinity]) assert.deepEqual(liveTimeline(value), liveTimeline(0))
  assert.equal(liveTimeline(ROUND_MS, 1).phase, 'complete')
  assert.throws(() => liveTimeline(0, 0), RangeError)
})

test('game rules follow rock > scissors > paper > rock', () => {
  assert.deepEqual(BEATS, { rock: 'scissors', scissors: 'paper', paper: 'rock' })
  for (const [stance, beaten] of Object.entries(BEATS)) {
    assert.equal(resolveClash(stance, beaten), 'a')
    assert.equal(resolveClash(beaten, stance), 'b')
    assert.equal(resolveClash(stance, stance), 'draw')
  }
})
