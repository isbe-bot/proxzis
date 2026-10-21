<script setup>
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { Pause, Play } from 'lucide-vue-next'
import HandGlyph from './HandGlyph.vue'
import { handOutcome } from '../utils/presentation.js'
import { resolveClash, summarizeRounds } from '../utils/arena.js'
import { createLiveClock, liveTimeline } from '../utils/liveClock.js'

const props = defineProps({
  left: { type: Object, default: null },
  right: { type: Object, default: null },
  rounds: { type: Array, default: () => [] },
  stake: { type: Number, default: 0 },
})
const emit = defineEmits(['profile'])
const left = computed(() => props.left)
const right = computed(() => props.right)
const liveRounds = computed(() => props.rounds || [])
const elapsed = ref(0), paused = ref(false)
const clock = createLiveClock()
const userWager = computed(() => left.value && props.stake > 0 ? { amount: props.stake, targetName: left.value.name, pot: props.stake * 2 } : null)

// Snappy round cadence: trimmed ending so results don't linger
// 2200ms per round keeps the action moving briskly
const ROUND_DURATION_MS = 2200
const FINAL_HOLD_DURATION_MS = 2400
const LAND_MS = 280

const totalDuration = computed(() => liveRounds.value.length * ROUND_DURATION_MS)
const cycleDuration = computed(() => totalDuration.value + FINAL_HOLD_DURATION_MS)

const roundInfo = computed(() => {
  if (!liveRounds.value.length) return { roundIndex: 0, roundElapsed: 0, isMatchComplete: false, isAirborne: true, resolved: 0 }
  const time = Math.max(0, elapsed.value)
  const local = time % cycleDuration.value
  const isMatchComplete = local >= totalDuration.value
  const rIndex = isMatchComplete
    ? liveRounds.value.length - 1
    : Math.min(Math.floor(local / ROUND_DURATION_MS), liveRounds.value.length - 1)
  const rElapsed = isMatchComplete ? ROUND_DURATION_MS : local % ROUND_DURATION_MS
  const isAirborne = !isMatchComplete && rElapsed < LAND_MS
  const resolvedCount = isMatchComplete ? liveRounds.value.length : (rElapsed >= LAND_MS ? rIndex + 1 : rIndex)

  return {
    roundIndex: rIndex,
    roundElapsed: rElapsed,
    isMatchComplete,
    isAirborne,
    resolved: resolvedCount,
  }
})

// Standard timeline reference
const timeline = computed(() => liveTimeline(elapsed.value, liveRounds.value.length || 1))
const phase = computed(() => timeline.value.phase)

const roundIndex = computed(() => roundInfo.value.roundIndex)
const round = computed(() => liveRounds.value[roundIndex.value])
const resolved = computed(() => roundInfo.value.resolved)
const score = computed(() => summarizeRounds(liveRounds.value.slice(0, resolved.value)))
const winner = computed(() => round.value ? resolveClash(round.value.a, round.value.b) : 'draw')
const isAirborne = computed(() => roundInfo.value.isAirborne)

// Immediate outcome for throw reveal: active right on floor hit
const revealOutcome = (side) => {
  if (isAirborne.value) return null
  return winner.value === 'draw' ? 'draw' : winner.value === side ? 'win' : 'lose'
}

// Tumbler / Waterfall vertical motion
const tumblerOffset = computed(() => {
  if (roundInfo.value.isMatchComplete) return 0
  const t = roundInfo.value.roundElapsed

  if (t < LAND_MS) {
    // Waterfall drop from above (-160px) straight to baseline (0px)
    const p = t / LAND_MS
    return -160 * (1 - p * p)
  } else if (t >= 1800) {
    // Tumbler drops down off bottom quickly as next roll cascades in
    const p = (t - 1800) / 400
    return 150 * (p * p)
  }
  return 0
})

const currentBanterA = computed(() => round.value?.quoteA || left.value?.quote || '')
const currentBanterB = computed(() => round.value?.quoteB || right.value?.quote || '')

let frame
function sample() { elapsed.value = clock.elapsed() }
function animate() {
  sample()
  if (!paused.value && !document.hidden) frame = requestAnimationFrame(animate)
}
function schedule() {
  cancelAnimationFrame(frame)
  if (!paused.value && !document.hidden) frame = requestAnimationFrame(animate)
}
function togglePause() {
  if (paused.value) clock.resume()
  else clock.pause()
  paused.value = !paused.value
  sample()
  schedule()
}
function visibilityChanged() { sample(); schedule() }

onMounted(() => {
  clock.restart()
  schedule()
  document.addEventListener('visibilitychange', visibilityChanged)
})
onBeforeUnmount(() => {
  cancelAnimationFrame(frame)
  document.removeEventListener('visibilitychange', visibilityChanged)
})
</script>

<template>
  <section class="live-panel panel" aria-labelledby="live-title">
    <div class="panel-top">
      <div class="heading-inline">
        <span class="live-dot" :class="{ paused }"></span>
        <h2 id="live-title">Inside the arena</h2>
        <span class="badge neutral">SIMULATION</span>
      </div>
      <div class="live-controls">
        <button class="icon-button" :aria-label="paused ? 'Resume simulation' : 'Pause simulation'" :aria-pressed="paused" @click="togglePause">
          <Play v-if="paused" :size="15" />
          <Pause v-else :size="15" />
        </button>
      </div>
    </div>

    <!-- Active User Wager Pill / Bet Display -->
    <div v-if="userWager" class="wager-banner" aria-label="Active user wager">
      <div class="wager-chip">
        <span class="chip-icon">●</span>
        <span class="chip-label">ACTIVE WAGER:</span>
        <strong class="chip-value">{{ userWager.amount }} MARBLES</strong>
      </div>
      <div class="wager-meta">
        <span class="wager-target">ON <strong>{{ userWager.targetName }}</strong> TO WIN</span>
        <span class="wager-divider">/</span>
        <span class="wager-pot">POT: <strong>{{ userWager.pot }} MARBLES</strong></span>
      </div>
    </div>

    <p v-if="!(left && right && round)" class="mono" role="status">Loading the exhibition.</p>
    <!-- Arena Stage with Waterfall / Tumbler Motion -->
    <div v-if="left && right && round" class="arena-stage" :style="{ '--hand-offset': `${tumblerOffset + (timeline.handOffset * 0)}px` }">
      <div class="arena-grid" aria-hidden="true"></div>
      <div class="arena-orbit" aria-hidden="true"></div>

      <!-- Left Fighter -->
      <div class="fighter fighter-left">
        <div class="hand-facing tumbler-slot">
          <span class="hand-motion">
            <HandGlyph
              :stance="round.a"
              side="a"
              :owner="left.name"
              :concealed="isAirborne"
              :outcome="revealOutcome('a') || handOutcome(phase, winner, 'a')"
            />
          </span>
        </div>
        <button class="fighter-name-link" @click="emit('profile', left)">
          {{ left.name }}
        </button>
        <!-- Banter for the round below user's name -->
        <p class="round-banter" aria-live="polite">“{{ currentBanterA }}”</p>
      </div>

      <!-- Center Score (clean, no ro-sham-bo text) -->
      <div class="arena-score">
        <span class="eyebrow">{{ roundInfo.isMatchComplete ? 'FINAL SCORE' : `ROUND ${roundIndex + 1} OF ${liveRounds.length}` }}</span>
        <div class="score-digits">
          <span>{{ score.a }}</span><i>-</i><span>{{ score.b }}</span>
        </div>
        <span v-if="paused" class="arena-paused">PAUSED</span>
      </div>

      <!-- Right Fighter -->
      <div class="fighter fighter-right">
        <div class="hand-facing tumbler-slot">
          <span class="hand-motion">
            <HandGlyph
              :stance="round.b"
              side="b"
              :owner="right.name"
              :concealed="isAirborne"
              :outcome="revealOutcome('b') || handOutcome(phase, winner, 'b')"
            />
          </span>
        </div>
        <button class="fighter-name-link" @click="emit('profile', right)">
          {{ right.name }}
        </button>
        <!-- Banter for the round below user's name -->
        <p class="round-banter" aria-live="polite">“{{ currentBanterB }}”</p>
      </div>
    </div>

    <!-- Round progression track -->
    <div v-if="left && right && round" class="round-track" aria-label="Simulation round progress">
      <span
        v-for="(r, i) in liveRounds"
        :key="i"
        :class="{
          done: i < resolved,
          current: i === roundIndex && !roundInfo.isMatchComplete,
          'left-win': i < resolved && resolveClash(r.a, r.b) === 'a',
          'right-win': i < resolved && resolveClash(r.a, r.b) === 'b'
        }"
      >{{ i + 1 }}</span>
    </div>
  </section>
</template>

<style scoped>
/* Wager Display Banner */
.wager-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px 14px;
  background: rgba(197, 239, 137, 0.08);
  border: 1px solid rgba(197, 239, 137, 0.22);
  border-radius: 8px;
  padding: 8px 14px;
  margin: 4px 0 12px;
  font-family: inherit;
}

.wager-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--mint, #c5ef89);
}

.chip-icon {
  font-size: 11px;
  color: var(--mint, #c5ef89);
  filter: drop-shadow(0 0 4px rgba(197, 239, 137, 0.5));
}

.chip-label {
  font-weight: 600;
  letter-spacing: 0.5px;
  opacity: 0.9;
}

.chip-value {
  font-weight: 800;
  color: #fff;
}

.wager-meta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 11px;
  color: var(--text-muted, #9bb098);
}

.wager-target strong {
  color: var(--mint, #c5ef89);
}

.wager-divider {
  opacity: 0.35;
}

.wager-pot strong {
  color: #fff;
}

/* Arena Stage: strict horizontal baseline / center alignment */
.arena-stage {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  position: relative;
  overflow: hidden;
  padding: 24px 16px;
  min-height: 250px;
}

.fighter {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  position: relative;
  z-index: 2;
  text-align: center;
  max-width: 100%;
}

.fighter-left,
.fighter-right {
  align-items: center;
}

/* Hands sized large with exact horizontal alignment */
.hand-facing {
  --hand-size: 135px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 155px;
  height: 135px;
  position: relative;
}

/* Tumbler slot machine container to allow waterfall entry and exit */
.tumbler-slot {
  overflow: visible;
}

/* Vertical tumbler motion */
.hand-motion {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transform: translateY(var(--hand-offset, 0px));
  will-change: transform;
}

/* Fighter name: clean link, no avatar, no arrow */
.fighter-name-link {
  margin-top: 14px;
  background: none;
  border: none;
  padding: 0;
  font-family: inherit;
  font-size: 14px;
  font-weight: 700;
  color: var(--text, #e3ebd8);
  text-decoration: underline;
  text-underline-offset: 4px;
  text-decoration-color: rgba(255, 255, 255, 0.3);
  cursor: pointer;
  transition: color 0.15s ease, text-decoration-color 0.15s ease;
}

.fighter-name-link:hover {
  color: var(--mint, #c5ef89);
  text-decoration-color: var(--mint, #c5ef89);
}

.fighter-name-link:focus-visible,
.live-controls button:focus-visible {
  outline: 2px solid var(--mint, #c5ef89);
  outline-offset: 4px;
}

/* Banter for the round directly below user's name */
.round-banter {
  margin-top: 6px;
  font-size: 11px;
  line-height: 1.4;
  color: var(--text-muted, #9bb098);
  font-style: italic;
  max-width: 160px;
  min-height: 2.8em;
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
  transition: opacity 0.2s ease;
}

/* Center score section */
.arena-score {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  z-index: 2;
  padding: 0 16px 8px;
}

.arena-paused {
  min-height: 1.4em;
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 2px;
  color: var(--mint, #c5ef89);
  text-align: center;
  text-transform: uppercase;
}

@media (max-width: 600px) {
  .hand-facing {
    --hand-size: 96px;
    width: 110px;
    height: 96px;
  }
  .fighter-name-link {
    font-size: 11px;
  }
  .round-banter {
    font-size: 11px;
    max-width: 120px;
  }
  .wager-banner {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
}

@media (max-width: 420px) {
  .hand-facing {
    --hand-size: 78px;
    width: 88px;
    height: 78px;
  }
  .arena-stage {
    padding: 16px 8px;
  }
  .round-banter {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .hand-motion { transform: none !important; }
  .live-dot { animation: none; }
}
</style>
