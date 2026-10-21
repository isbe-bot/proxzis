// One elapsed-time source drives both hands, the reveal, the score and the replay loop.
// Each round: both rock fists drop onto the floor (1), pound once more (2), and
// the next downstroke lands as the move (Shoot). The loser starts fading at Shoot,
// the result holds briefly, the hands lift out and the next round drops in.
export const DROP_MS = 300
export const BUMP_MS = 240
export const BUMP_COUNT = 2
export const SHOOT_AT_MS = DROP_MS + BUMP_MS * BUMP_COUNT // 780 ms
export const EXIT_MS = 240
export const ROUND_MS = 2560
export const FINAL_HOLD_MS = 4000
// Hand height above the floor, in bump heights. Drop starts and exit ends out of view.
export const OFFSTAGE_LIFT = 8

const easeIn = p => p * p
const easeOut = p => 1 - (1 - p) * (1 - p)

function liftAt(t) {
  if (t < DROP_MS) return OFFSTAGE_LIFT * (1 - easeIn(t / DROP_MS))
  if (t < SHOOT_AT_MS) {
    const q = ((t - DROP_MS) % BUMP_MS) / BUMP_MS
    return q < 0.5 ? easeOut(q / 0.5) : 1 - easeIn((q - 0.5) / 0.5)
  }
  if (t >= ROUND_MS - EXIT_MS) return OFFSTAGE_LIFT * easeIn((t - (ROUND_MS - EXIT_MS)) / EXIT_MS)
  return 0
}

export function liveTimeline(elapsed, roundCount = 7) {
  if (!Number.isInteger(roundCount) || roundCount < 1) throw new RangeError('At least one round is required')
  const duration = roundCount * ROUND_MS
  const cycle = duration + FINAL_HOLD_MS
  const time = Number.isFinite(elapsed) ? Math.max(0, elapsed) : 0
  const local = time % cycle
  const complete = local >= duration
  const roundIndex = Math.min(Math.floor(local / ROUND_MS), roundCount - 1)
  const roundElapsed = complete ? ROUND_MS : local % ROUND_MS
  const phase = complete ? 'complete' : roundElapsed < DROP_MS ? 'drop' : roundElapsed < SHOOT_AT_MS ? 'count' : 'result'
  return {
    bout: Math.floor(time / cycle) + 1,
    roundIndex,
    roundElapsed,
    phase,
    // 0 while falling, 1-2 after each rock lands, 3 once Shoot lands.
    beat: phase === 'drop' ? 0 : phase === 'count' ? 1 + Math.floor((roundElapsed - DROP_MS) / BUMP_MS) : BUMP_COUNT + 1,
    lift: complete ? 0 : liftAt(roundElapsed),
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
