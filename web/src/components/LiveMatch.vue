<script setup>
import { computed, ref, watch, nextTick, onMounted, onBeforeUnmount, useId } from 'vue'
import HandGlyph from './HandGlyph.vue'
import AgentMark from './AgentMark.vue'
import { handOutcome, stanceName } from '../utils/presentation.js'
import { number, resolveClash, summarizeRounds } from '../utils/arena.js'
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
const elapsed = ref(0)
const chatEl = ref(null)
const chatId = useId()
const chatOpen = ref(false)
const clock = createLiveClock()

const timeline = computed(() => liveTimeline(elapsed.value, liveRounds.value.length || 1))
const phase = computed(() => timeline.value.phase)
const isComplete = computed(() => phase.value === 'complete')
// Both fists stay rock through the drop and the count; Shoot lands as the move.
const concealed = computed(() => phase.value === 'drop' || phase.value === 'count')

const roundIndex = computed(() => timeline.value.roundIndex)
const round = computed(() => liveRounds.value[roundIndex.value])
const resolved = computed(() => Math.min(timeline.value.resolved, liveRounds.value.length))
const score = computed(() => summarizeRounds(liveRounds.value.slice(0, resolved.value)))
const winner = computed(() => round.value ? resolveClash(round.value.a, round.value.b) : 'draw')

const seriesWinner = computed(() => {
  if (!isComplete.value || !left.value || !right.value) return null
  const { a, b } = score.value
  if (a === b) return null
  return a > b ? { side: 'a', name: left.value.name } : { side: 'b', name: right.value.name }
})
const lostBet = side => Boolean(seriesWinner.value && seriesWinner.value.side !== side)
const finalScore = computed(() => {
  const { a, b } = score.value
  return `${Math.max(a, b)} - ${Math.min(a, b)}`
})

const pipState = i => {
  if (i >= resolved.value) return i === roundIndex.value && !isComplete.value ? 'current' : 'open'
  const w = resolveClash(liveRounds.value[i].a, liveRounds.value[i].b)
  return w === 'a' ? 'won-a' : w === 'b' ? 'won-b' : 'drawn'
}

const log = computed(() => {
  if (!left.value || !right.value || !liveRounds.value.length) return []
  const items = [{ id: 'begin', kind: 'system', text: 'Game begins.' }]
  for (let i = 0; i < resolved.value; i++) {
    const played = liveRounds.value[i]
    if (played.quoteA) items.push({ id: `a-${i}`, kind: 'a', name: left.value.name, text: played.quoteA })
    if (played.quoteB) items.push({ id: `b-${i}`, kind: 'b', name: right.value.name, text: played.quoteB })
    const side = resolveClash(played.a, played.b)
    const n = i + 1
    if (side === 'draw') items.push({ id: `r-${i}`, kind: 'system', text: `Round ${n} is a draw.` })
    else if (side === 'a') items.push({ id: `r-${i}`, kind: 'system', text: `${left.value.name} wins round ${n} with ${stanceName(played.a)}.` })
    else items.push({ id: `r-${i}`, kind: 'system', text: `${right.value.name} wins round ${n} with ${stanceName(played.b)}.` })
  }
  if (isComplete.value) {
    const closing = seriesWinner.value ? `Game complete. ${seriesWinner.value.name} wins.` : 'Game complete. Draw.'
    items.push({ id: 'done', kind: 'system', text: closing })
  }
  return items
})

function pinChat() {
  const el = chatEl.value
  if (el) el.scrollTop = el.scrollHeight
}

// Each replay starts with an empty chat; new lines keep it pinned to the bottom.
watch(() => [timeline.value.bout, log.value.length], ([bout], previous) => {
  nextTick(() => {
    const el = chatEl.value
    if (!el) return
    if (!previous || bout !== previous[0]) el.scrollTop = 0
    else pinChat()
  })
})

function toggleChat() {
  chatOpen.value = !chatOpen.value
  if (chatOpen.value) nextTick(pinChat)
}

let frame
function sample() { elapsed.value = clock.elapsed() }
function animate() {
  sample()
  if (!document.hidden) frame = requestAnimationFrame(animate)
}
function schedule() {
  cancelAnimationFrame(frame)
  if (!document.hidden) frame = requestAnimationFrame(animate)
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
  <div class="duel-frame">
  <div class="duel-slot">
  <section class="duel" :class="{ 'chat-closed': !chatOpen }" aria-label="Arena">
    <div class="duel-play">
      <div class="duel-body">
        <p v-if="!(left && right && round)" class="duel-loading" role="status">Loading the exhibition.</p>

        <div v-else class="duel-stage" :style="{ '--lift': timeline.lift.toFixed(3) }">
          <div class="corner side-a">
            <button class="corner-name" type="button" :title="left.name" @click="emit('profile', left)">
              <AgentMark :bot="left" side="a" :size="28" :label="score.a" />
              <span class="corner-id"><strong>{{ left.name }}</strong><small v-if="stake > 0" class="t-num"><span class="corner-bet" :class="{ lost: lostBet('a') }"><span class="sr-only">Pass bet </span>{{ lostBet('a') ? `\u2212${number(stake)}` : number(stake) }}<span class="sr-only"> marbles</span></span></small></span>
            </button>
            <div v-if="!isComplete" class="hand-slot">
              <span class="hand-motion">
                <HandGlyph :stance="round.a" side="a" :owner="left.name" :concealed="concealed" :outcome="handOutcome(phase, winner, 'a')" />
              </span>
            </div>
          </div>

          <div class="corner side-b">
            <button class="corner-name" type="button" :title="right.name" @click="emit('profile', right)">
              <AgentMark :bot="right" side="b" :size="28" :label="score.b" />
              <span class="corner-id"><strong>{{ right.name }}</strong><small v-if="stake > 0" class="t-num"><span class="corner-bet" :class="{ lost: lostBet('b') }"><span class="sr-only">Pass bet </span>{{ lostBet('b') ? `\u2212${number(stake)}` : number(stake) }}<span class="sr-only"> marbles</span></span></small></span>
            </button>
            <div v-if="!isComplete" class="hand-slot">
              <span class="hand-motion">
                <HandGlyph :stance="round.b" side="b" :owner="right.name" :concealed="concealed" :outcome="handOutcome(phase, winner, 'b')" />
              </span>
            </div>
          </div>

          <div v-if="isComplete" class="duel-final" :class="seriesWinner ? `won-${seriesWinner.side}` : 'is-draw'" role="status">
            <span class="final-kicker">Match complete</span>
            <strong class="final-name" :title="seriesWinner?.name">{{ seriesWinner ? seriesWinner.name : 'Draw' }}</strong>
            <span class="final-score t-num">{{ seriesWinner ? 'Winner' : 'Drawn' }} {{ finalScore }}</span>
          </div>

          <div class="duel-score">
            <p class="sr-only">Score: {{ left.name }} {{ score.a }}, {{ right.name }} {{ score.b }}</p>
            <ol class="pips" aria-label="Rounds">
              <li v-for="(r, i) in liveRounds" :key="i" :class="pipState(i)"><span class="sr-only">Round {{ i + 1 }}: {{ pipState(i) }}</span></li>
            </ol>
            <button class="chat-toggle" type="button" :aria-expanded="chatOpen" :aria-controls="chatId" :aria-label="chatOpen ? 'Hide match chat' : 'Show match chat'" :title="chatOpen ? 'Hide chat' : 'Show chat'" @click="toggleChat">
              <svg viewBox="0 0 16 16" width="16" height="16" aria-hidden="true">
                <path d="M2.5 3.5h11v7h-6l-3 2.5v-2.5h-2z" fill="none" stroke="currentColor" stroke-width="1.25" stroke-linejoin="round" />
                <path d="M5.5 7h.01M8 7h.01M10.5 7h.01" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" />
              </svg>
            </button>
          </div>
        </div>
      </div>

      <div class="chat-dock">
        <div :id="chatId" ref="chatEl" class="chat" role="log" aria-label="Match chat" :tabindex="chatOpen ? 0 : -1" :inert="!chatOpen">
          <ol :key="timeline.bout" class="chat-log">
            <li v-for="item in log" :key="item.id" class="chat-line" :class="`is-${item.kind}`">
              <span v-if="item.kind === 'system'" class="chat-system">{{ item.text }}</span>
              <template v-else>
                <span class="chat-name">{{ item.name }}</span>
                <span class="chat-text">{{ item.text }}</span>
              </template>
            </li>
          </ol>
        </div>
      </div>
    </div>
  </section>
  </div>
  </div>
</template>

<style scoped>
/*
  The fight is a fixed block: names, hands, then the round pips centered under
  both fists. The chat column scrolls beside it and slides behind the fight
  when closed. Text truncates; the hands are clipped by their slot.
*/
/*
  The card owns its size so it drops into the home page or a modal unchanged:
  two corners, the inset around them, and the chat column. It never grows
  past that, and below 480px of room it stacks the chat under the fight.
  The frame measures the room, so toggling the chat never flips the layout.
  The slot is the fight's width and hosts place it with --duel-justify
  (default center). The card pins to the slot's left edge and the open chat
  extends past it to the right, so the fight never moves. When the room
  can't fit a chat's width on both sides, the slot reserves the open width.
*/
.duel-frame {
  --corner-width: 8.5rem;
  --corner-gap: var(--space-2);
  --arena-pad: var(--space-5);
  --arena-height: 15.25rem;
  --chat-width: 14rem;
  --score-gap: 50px;
  --fight-width: calc(var(--corner-width) * 2 + var(--corner-gap) + var(--arena-pad) * 2 + 2px);
  --slide: var(--dur-slow) var(--ease);
  container: duel / inline-size;
  display: flex;
  justify-content: var(--duel-justify, center);
  width: 100%;
  min-width: 0;
}
.duel-slot {
  display: flex;
  width: min(100%, var(--fight-width));
  min-width: 0;
}

.duel {
  --chat-track: var(--chat-width);
  position: relative;
  flex: none;
  width: calc(var(--fight-width) + var(--chat-track));
  min-width: 0;
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: var(--radius-surface);
  box-shadow: var(--shadow-2);
  overflow: hidden;
  transition: width var(--slide);
}
.duel.chat-closed { --chat-track: 0px; }

.duel-play {
  display: grid;
  grid-template-columns: minmax(0, 1fr) var(--chat-track);
  height: var(--arena-height);
  transition: grid-template-columns var(--slide);
}

.duel-body {
  --row-name: 2.25rem;
  --name-gap: var(--space-2);
  --slot-width: 6.5rem;
  --hand-size: 5.5rem;
  --headroom: 1.5rem;
  --slot-height: calc(var(--hand-size) * 0.75 + var(--headroom));
  --bump-height: 16px;
  --swing-angle: 9deg;
  position: relative;
  z-index: 1;
  display: grid;
  min-width: 0;
  padding: var(--arena-pad);
  background: var(--surface);
  overflow: hidden;
}

.duel-loading {
  display: grid;
  place-items: center;
  height: 100%;
  margin: 0;
  color: var(--ink-muted);
  font-size: var(--step--1);
}

/*
  Three rows across two equal columns: names on top, hands in the middle,
  pips on the bottom. Each name is centered over its own hand.
*/
.duel-stage {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, var(--corner-width)));
  grid-template-rows: var(--row-name) minmax(var(--slot-height), 1fr) auto;
  justify-content: center;
  column-gap: var(--corner-gap);
  row-gap: var(--space-3);
  height: 100%;
}

.corner { display: contents; }
.corner.side-a > * { grid-column: 1; }
.corner.side-b > * { grid-column: 2; }
.corner-name { grid-row: 1; }
.side-a .corner-name { justify-self: start; }
.side-b .corner-name { justify-self: end; flex-direction: row-reverse; text-align: right; }
.side-b .corner-id { align-items: flex-end; }
.hand-slot { grid-row: 2; }

.corner-name {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  max-width: 100%;
  min-width: 0;
  height: var(--row-name);
  padding: 0;
  border: 0;
  background: none;
  color: var(--ink);
  text-align: left;
  cursor: pointer;
}
.corner-id { display: flex; flex-direction: column; min-width: 0; }
.corner-id strong,
.corner-id small {
  display: block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.corner-id strong {
  font-size: var(--step--1);
  font-weight: var(--weight-bold);
  line-height: var(--leading-tight);
  text-decoration: underline;
  text-decoration-color: transparent;
  text-underline-offset: 0.25em;
  transition: text-decoration-color var(--dur-fast) var(--ease);
}
.corner-name:hover strong { text-decoration-color: var(--side); }
.corner-id small { font-size: var(--step--2); line-height: 1.4; color: var(--ink-muted); }
.hand-slot {
  grid-row: 2;
  align-self: end;
  display: grid;
  place-items: end center;
  width: min(var(--slot-width), 100%);
  height: var(--slot-height);
  overflow: hidden;
}
.hand-motion {
  display: inline-flex;
  transform-origin: var(--elbow-x) 50%;
  transform:
    translateY(calc(max(var(--lift, 0) - 1, 0) * var(--bump-height) * -1))
    rotate(calc(min(var(--lift, 0), 1) * var(--swing)));
  will-change: transform;
}
.hand-slot { justify-self: center; align-self: end; }

/* Settlement replaces both hands with the result, centered across the stage. */
.duel-final {
  grid-column: 1 / -1;
  grid-row: 2;
  display: grid;
  align-content: center;
  justify-items: center;
  margin-bottom: calc(var(--space-3) - var(--score-gap));
  gap: var(--space-1);
  min-width: 0;
  text-align: center;
}
.final-kicker,
.final-score {
  font-family: var(--font-mono);
  font-size: var(--step--2);
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--ink-muted);
}
.final-name {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-stretch: var(--stretch-display);
  font-size: var(--step-4);
  font-weight: var(--weight-black);
  line-height: var(--leading-tight);
  letter-spacing: -0.01em;
  color: var(--ink);
}
.won-a .final-score { color: var(--cobalt); }
.won-b .final-score { color: var(--crimson); }
.side-a .hand-motion { --elbow-x: -55%; --swing: calc(var(--swing-angle) * -1); }
.side-b .hand-motion { --elbow-x: 155%; --swing: var(--swing-angle); }

.duel-score {
  grid-column: 1 / -1;
  grid-row: 3;
  position: relative;
  margin-top: calc(var(--score-gap) - var(--space-3));
  display: flex;
  justify-content: center;
  min-width: 0;
}

.pips {
  --pip-width: 14px;
  --pip-gap: 4px;
  display: flex;
  gap: var(--pip-gap);
  justify-content: center;
  width: 100%;
  margin: 0;
  padding: 0;
  list-style: none;
}
.pips li {
  flex: 0 1 var(--pip-width);
  min-width: 4px;
  height: 5px;
  border-radius: 1.5px;
  background: var(--carbon-700);
  transition: background-color var(--dur-base) var(--ease);
}
.pips li.current { background: var(--ink-muted); }
.pips li.won-a { background: var(--cobalt); }
.pips li.won-b { background: var(--crimson); }
.pips li.drawn { background: var(--ink-secondary); }

.chat-toggle {
  position: absolute;
  top: 50%;
  right: 0;
  display: grid;
  place-items: center;
  width: 1.5rem;
  height: 1.5rem;
  padding: 0;
  border: 1px solid transparent;
  border-radius: var(--radius-control);
  background: none;
  color: var(--ink-secondary);
  cursor: pointer;
  transform: translateY(-50%);
  transition: color var(--dur-fast) var(--ease), border-color var(--dur-fast) var(--ease);
}
.chat-toggle:hover { color: var(--ink); border-color: var(--line); }
.chat-toggle:focus-visible { outline: 2px solid var(--focus-ring); outline-offset: 2px; }
.chat-closed .chat-toggle { color: var(--ink-muted); }
.chat-closed .chat-toggle:hover { color: var(--ink); }

/* The dock collapses with the track; the chat keeps its size and slides under the fight. */
.chat-dock {
  min-width: 0;
  min-height: 0;
}
.chat {
  width: var(--chat-width);
  height: 100%;
  transition: transform var(--slide), visibility 0s;
  min-height: 0;
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
  border-left: 1px solid var(--line);
  background: var(--bg-sunken);
  padding: var(--space-5) var(--space-4) var(--space-5) var(--space-5);
}
.chat::-webkit-scrollbar { width: 6px; }
.chat::-webkit-scrollbar-track { background: transparent; }
.chat::-webkit-scrollbar-thumb { background: var(--line-strong); border-radius: 3px; }
.chat::-webkit-scrollbar-button { display: none; }
/* Chromium drops the ::-webkit-scrollbar rules once scrollbar-color is set. */
@supports not selector(::-webkit-scrollbar) {
  .chat { scrollbar-width: thin; scrollbar-color: var(--line-strong) transparent; }
}
.chat:focus-visible { outline: 2px solid var(--focus-ring); outline-offset: -2px; }
.chat-closed .chat {
  transform: translateX(-100%);
  visibility: hidden;
  transition: transform var(--slide), visibility 0s var(--dur-slow);
}
.chat-log {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin: 0;
  padding: 0;
  list-style: none;
}
.chat-line { margin: 0; min-width: 0; }
.chat-line.is-system + .chat-line:not(.is-system),
.chat-line:not(.is-system) + .chat-line.is-system { margin-top: var(--space-1); }
.chat-system {
  display: block;
  font-size: calc(var(--step--2) - 1px);
  line-height: 1.4;
  text-align: center;
  color: var(--ink-muted);
}
.chat-name {
  display: block;
  margin-bottom: 2px;
  font-size: calc(var(--step--2) - 1px);
  font-weight: var(--weight-bold);
  line-height: 1.3;
}
.is-a .chat-name { color: var(--cobalt); }
.is-b .chat-name { color: var(--crimson); }
.chat-text {
  display: block;
  font-size: calc(var(--step--1) - 1px);
  line-height: 1.4;
  color: var(--ink);
  overflow-wrap: anywhere;
}

.corner-bet {
  font-weight: var(--weight-bold);
  color: var(--gold);
  transition: color var(--dur-base) var(--ease);
}
.corner-bet::before {
  content: "";
  display: inline-block;
  width: 0.55em;
  height: 0.55em;
  margin-right: 0.3em;
  border-radius: 50%;
  background: currentColor;
  vertical-align: 0.05em;
}
.corner-bet.lost { color: var(--crimson); }

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
}

/* Under fight + 2 × chat (48.5rem + border), centering on the fight would push the open chat out of the room. */
@container duel (max-width: 49rem) {
  .duel-slot { width: min(100%, calc(var(--fight-width) + var(--chat-width))); }
  .duel { flex: 0 1 auto; width: min(100%, calc(var(--fight-width) + var(--chat-track))); }
}

@container duel (max-width: 479px) {
  .duel { --chat-rows: 10rem; width: 100%; }
  .duel.chat-closed { --chat-rows: 0px; }
  .duel-play {
    --arena-pad: var(--space-4);
    grid-template-columns: 1fr;
    grid-template-rows: auto var(--chat-rows);
    height: auto;
    transition: grid-template-rows var(--slide);
  }
  .duel-body {
    --slot-width: 5.25rem;
    --hand-size: 4.5rem;
    --headroom: 1.25rem;
    --bump-height: 12px;
  }
  .duel-stage { column-gap: var(--space-4); }
  .chat {
    width: 100%;
    height: 10rem;
    border-left: 0;
    border-top: 1px solid var(--line);
  }
  .chat-closed .chat { transform: translateY(-100%); }
  .pips { --pip-width: 10px; --pip-gap: 3px; }
}

@media (prefers-reduced-motion: reduce) {
  .hand-motion { transform: none !important; }
  .pips li, .corner-id strong, .corner-bet { transition: none; }
  .duel, .duel-play, .chat, .chat-closed .chat, .chat-toggle { transition: none; }
}
</style>
