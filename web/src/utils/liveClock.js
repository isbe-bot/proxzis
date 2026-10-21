// One elapsed-time source drives labels, both hands, reveal, scores and replay.
export const SHAKE_MS = 350
export const SHAKE_COUNT = 3
export const REVEAL_AT_MS = SHAKE_MS * SHAKE_COUNT // 1050 ms
export const RESULT_AT_MS = REVEAL_AT_MS + 400     // 1450 ms: quick transition directly into judging & fade
export const ROUND_MS = 5400
export const FINAL_HOLD_MS = 5400

export function liveTimeline(elapsed, roundCount = 7) {
  if (!Number.isInteger(roundCount) || roundCount < 1) throw new RangeError('At least one round is required')
  const duration = roundCount * ROUND_MS
  const cycle = duration + FINAL_HOLD_MS
  const time = Number.isFinite(elapsed) ? Math.max(0, elapsed) : 0
  const local = time % cycle
  const complete = local >= duration
  const roundIndex = Math.min(Math.floor(local / ROUND_MS), roundCount - 1)
  const roundElapsed = complete ? ROUND_MS : local % ROUND_MS
  const phase = complete ? 'complete' : roundElapsed < REVEAL_AT_MS ? 'thinking' : roundElapsed < RESULT_AT_MS ? 'reveal' : 'result'
  const shakeIndex = phase === 'thinking' ? Math.floor(roundElapsed / SHAKE_MS) : SHAKE_COUNT
  const shakeProgress = phase === 'thinking' ? (roundElapsed % SHAKE_MS) / SHAKE_MS : 0
  return {
    bout: Math.floor(time / cycle) + 1,
    roundIndex,
    roundElapsed,
    phase,
    shakeIndex,
    // Exactly one rise and return per cue; no independent CSS animation clock.
    handOffset: phase === 'thinking' && shakeProgress > 0 ? -18 * Math.sin(Math.PI * shakeProgress) : 0,
    cue: complete ? 'Match complete' : phase === 'thinking' ? ['Rock', 'Paper', 'Scissors'][shakeIndex] : phase === 'reveal' ? 'Shoot!' : 'Round result',
    resolved: complete ? roundCount : roundIndex + (phase === 'result' ? 1 : 0),
    remaining: complete ? 0 : ROUND_MS - roundElapsed,
  }
}

// The scheduler only samples this clock. Missed frames/background throttling do
// not slow the match down; explicit pauses exclude their actual elapsed time.
export function createLiveClock(now = () => performance.now()) {
  let origin = now()
  let held = 0
  let paused = false
  const elapsed = () => held + (paused ? 0 : Math.max(0, now() - origin))
  return {
    elapsed,
    pause() {
      if (paused) return
      held = elapsed()
      paused = true
    },
    resume() {
      if (!paused) return
      origin = now()
      paused = false
    },
    restart() {
      origin = now()
      held = 0
      paused = false
    },
  }
}
