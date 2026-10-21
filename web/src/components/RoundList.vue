<script setup>
import HandGlyph from './HandGlyph.vue'
import { stanceName } from '../utils/presentation.js'
import { resolveClash } from '../utils/arena.js'
const props = defineProps({ match: { type: Object, required: true }, compact: Boolean })
const winnerSide = round => {
  const winner = resolveClash(round.a, round.b)
  return winner === 'draw' ? 'is-tie' : `side-${winner}`
}
const winnerName = round => {
  const winner = resolveClash(round.a, round.b)
  return winner === 'draw' ? 'Draw' : props.match[winner].name
}
const state = (round, side) => {
  const winner = resolveClash(round.a, round.b)
  return winner === 'draw' ? 'is-draw' : winner === side ? 'is-win' : 'is-lose'
}
</script>

<template>
  <ol class="round-list" :class="{ compact }" :aria-label="`Round results for ${match.id}`">
    <li v-for="(round, i) in match.rounds" :key="i" class="round-item">
      <span class="round-number t-label">{{ String(i + 1).padStart(2, '0') }}</span>
      <div class="round-side side-a" :class="state(round, 'a')">
        <span class="round-stance">
          <span class="round-hand"><HandGlyph :stance="round.a" side="a" :owner="match.a.name" /></span>
          <span class="stance-name" aria-hidden="true">{{ stanceName(round.a) }}</span>
        </span>
        <p v-if="!compact && round.quoteA" class="round-quote"><span class="sr-only">{{ match.a.name }}: </span>“{{ round.quoteA }}”</p>
      </div>
      <span class="round-winner" :class="winnerSide(round)" :title="winnerName(round)">
        <span class="winner-name">{{ winnerName(round) }}</span>
        <span class="winner-dir" aria-hidden="true">{{ { 'side-a': '◂ Won', 'side-b': 'Won ▸' }[winnerSide(round)] || 'Draw' }}</span>
      </span>
      <div class="round-side side-b" :class="state(round, 'b')">
        <span class="round-stance">
          <span class="stance-name" aria-hidden="true">{{ stanceName(round.b) }}</span>
          <span class="round-hand"><HandGlyph :stance="round.b" side="b" :owner="match.b.name" /></span>
        </span>
        <p v-if="!compact && round.quoteB" class="round-quote"><span class="sr-only">{{ match.b.name }}: </span>“{{ round.quoteB }}”</p>
      </div>
    </li>
  </ol>
</template>

<style scoped>
.round-list {
  --hand-size: 2.75rem;
  --hand-slot: 3rem;
  margin: 0;
  padding: 0;
  list-style: none;
}
.round-item {
  display: grid;
  grid-template-columns: 2.5rem minmax(0, 1fr) 8rem minmax(0, 1fr);
  align-items: start;
  gap: var(--space-4);
  padding: var(--space-4) 0;
  border-bottom: 1px solid var(--line);
}
.round-item:last-child { border-bottom: 0; }
.round-number { display: flex; align-items: center; height: var(--hand-slot); }
.round-side { display: flex; flex-direction: column; gap: var(--space-2); min-width: 0; }
.round-side.side-b { align-items: flex-end; text-align: right; }
.round-stance { display: flex; align-items: center; gap: var(--space-3); min-width: 0; height: var(--hand-slot); }
.round-hand {
  display: grid;
  flex: none;
  place-items: center;
  width: var(--hand-slot);
  height: var(--hand-slot);
  overflow: hidden;
  transition: opacity var(--dur-base) var(--ease);
}
.stance-name {
  font-stretch: var(--stretch-display);
  font-size: var(--step-1);
  font-weight: var(--weight-black);
  letter-spacing: var(--tracking-display);
  white-space: nowrap;
  color: var(--side);
}
.is-lose .round-hand { opacity: 0.32; }
.is-lose .stance-name { color: var(--ink-muted); }
.is-draw .stance-name { color: var(--ink-secondary); }
.round-quote {
  max-width: 28rem;
  margin: 0;
  font-size: var(--step--1);
  line-height: 1.45;
  color: var(--ink-secondary);
  overflow-wrap: anywhere;
}
.round-winner {
  height: var(--hand-slot);
  min-width: 0;
  line-height: var(--hand-slot);
  overflow: hidden;
  text-align: center;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--step--1);
  font-weight: var(--weight-medium);
  color: var(--ink-secondary);
}
.winner-dir { display: none; }

.compact { --hand-size: 1.75rem; --hand-slot: 2rem; }
.compact .round-item {
  grid-template-columns: 2rem minmax(0, 1fr) 7rem minmax(0, 1fr);
  align-items: center;
  padding: var(--space-2) 0;
}
.compact .stance-name { font-size: var(--step-0); }

@media (max-width: 600px) {
  .round-item { grid-template-columns: 1.75rem minmax(0, 1fr) 4.5rem minmax(0, 1fr); gap: var(--space-2); }
  .compact .round-item { grid-template-columns: 1.75rem minmax(0, 1fr) 4.5rem minmax(0, 1fr); }
  .round-list { --hand-size: 2rem; --hand-slot: 2.25rem; }
  .stance-name { font-size: var(--step-0); }
  .round-stance { gap: var(--space-2); }
  .winner-name {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
  }
  .winner-dir { display: inline; font-family: var(--font-mono); font-size: var(--step--2); }
  .round-winner.side-a .winner-dir,
  .round-winner.side-b .winner-dir { color: var(--side); }
}

@media (prefers-reduced-motion: reduce) {
  .round-hand { transition: none; }
}
</style>
