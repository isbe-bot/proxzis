<script setup>
import { computed } from 'vue'
import AgentMark from './AgentMark.vue'
import { dateLabel, settlement, stanceName } from '../utils/presentation.js'
import { number, resolveClash } from '../utils/arena.js'

const props = defineProps({
  match: { type: Object, required: true },
})
const emit = defineEmits(['chat'])

const tone = computed(() => props.match.winner === 'draw' ? 'is-draw' : `won-${props.match.winner}`)
const winnerName = computed(() => props.match.winner === 'draw' ? 'Draw' : props.match[props.match.winner].name)
const finalScore = computed(() => {
  const { a, b } = props.match.score
  return props.match.winner === 'draw' ? `${a} - ${b}` : `${Math.max(a, b)} - ${Math.min(a, b)}`
})
const scoreLine = computed(() => `${props.match.winner === 'draw' ? 'Drawn' : 'Winner'} ${finalScore.value}`)

function purse(side) {
  const net = settlement(props.match, side).net
  if (net > 0) return { text: `+${number(net)}`, tone: 'won', label: `Won ${number(net)} marbles` }
  if (net < 0) return { text: `\u2212${number(Math.abs(net))}`, tone: 'lost', label: `Lost ${number(Math.abs(net))} marbles` }
  return { text: '0', tone: 'even', label: 'Stake returned' }
}

function roundWinner(round) {
  return resolveClash(round.a, round.b)
}

function pipState(round) {
  const winner = roundWinner(round)
  return winner === 'draw' ? 'drawn' : `won-${winner}`
}

function pipLabel(round) {
  const winner = roundWinner(round)
  return winner === 'draw' ? 'draw' : `${props.match[winner].name} won`
}

function moveState(round, side) {
  const winner = roundWinner(round)
  if (winner === 'draw') return 'is-draw'
  return winner === side ? 'is-win' : 'is-lose'
}

function moveLabel(round, side) {
  const winner = roundWinner(round)
  if (winner === 'draw') return 'drawn'
  return winner === side ? 'won' : 'lost'
}
</script>

<template>
  <article class="match-card" :class="tone">
    <header class="card-top">
      <div class="card-meta">
        <span class="t-label">{{ match.id }}</span>
        <time class="t-label" :datetime="match.date">{{ dateLabel(match.date) }}</time>
      </div>
      <button class="chat-open" type="button" :aria-label="`Open match chat for ${match.id}`" title="Match chat" @click="emit('chat', match)">
        <svg viewBox="0 0 16 16" width="18" height="18" aria-hidden="true">
          <path d="M2.5 3.5h11v7h-6l-3 2.5v-2.5h-2z" fill="none" stroke="currentColor" stroke-width="1.25" stroke-linejoin="round" />
          <path d="M5.5 7h.01M8 7h.01M10.5 7h.01" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" />
        </svg>
      </button>
    </header>

    <div class="card-stage">
      <div v-for="side in ['a', 'b']" :key="side" class="fighter" :class="`side-${side}`">
        <AgentMark :bot="match[side]" :side="side" :size="28" :label="match.score[side]" />
        <span class="fighter-id">
          <strong :title="match[side].name">{{ match[side].name }}</strong>
          <small class="t-num">
            <span class="purse" :class="purse(side).tone" :aria-label="purse(side).label"><span aria-hidden="true">{{ purse(side).text }}</span></span>
          </small>
        </span>
      </div>

      <div class="marquee" :class="tone">
        <span class="marquee-kicker">Match complete</span>
        <strong class="marquee-name" :title="winnerName">{{ winnerName }}</strong>
        <span class="marquee-score t-num">{{ scoreLine }}</span>
      </div>

      <div class="lights">
        <p class="sr-only">Score: {{ match.a.name }} {{ match.score.a }}, {{ match.b.name }} {{ match.score.b }}</p>
        <ol class="pips" aria-label="Rounds">
          <li v-for="(round, i) in match.rounds" :key="i" :class="pipState(round)"><span class="sr-only">Round {{ i + 1 }}: {{ pipLabel(round) }}</span></li>
        </ol>
      </div>
    </div>

    <div class="sheet-wrap">
      <table class="sheet">
        <caption class="sr-only">Moves per round for {{ match.id }}</caption>
        <thead>
          <tr>
            <th scope="col"><span class="sr-only">Round</span></th>
            <th scope="col" class="side-a" :title="match.a.name">{{ match.a.name }}</th>
            <th scope="col" class="side-b" :title="match.b.name">{{ match.b.name }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(round, i) in match.rounds" :key="i">
            <th scope="row">{{ String(i + 1).padStart(2, '0') }}</th>
            <td v-for="side in ['a', 'b']" :key="side" :class="[`side-${side}`, moveState(round, side)]">
              {{ stanceName(round[side]) }}
              <span class="sr-only">{{ moveLabel(round, side) }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </article>
</template>

<style scoped>
.match-card {
  display: flex;
  flex: 1;
  flex-direction: column;
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  padding: var(--space-5);
  overflow: hidden;
  background: var(--surface);
  border: 1px solid var(--line);
  border-top: 2px solid var(--line-strong);
  border-radius: var(--radius-surface);
  box-shadow: var(--shadow-1);
}
.match-card.won-a { border-top-color: var(--cobalt); }
.match-card.won-b { border-top-color: var(--crimson); }
.match-card.is-draw { border-top-color: var(--ink-muted); }

.card-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}
.card-meta {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  min-width: 0;
}
.card-meta .t-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-open {
  display: grid;
  flex: none;
  place-items: center;
  width: 2.75rem;
  height: 2.75rem;
  padding: 0;
  border: 1px solid transparent;
  border-radius: var(--radius-control);
  background: none;
  color: var(--ink-secondary);
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease), border-color var(--dur-fast) var(--ease);
}
.chat-open:hover { color: var(--ink); border-color: var(--line); }

.card-stage {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  align-items: center;
  column-gap: var(--space-4);
  row-gap: var(--space-4);
}

.fighter {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
}
.fighter.side-b { flex-direction: row-reverse; justify-self: end; text-align: right; }
.fighter-id { display: flex; flex-direction: column; min-width: 0; }
.fighter.side-b .fighter-id { align-items: flex-end; }
.fighter-id strong,
.fighter-id small {
  display: block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fighter-id strong {
  font-size: var(--step--1);
  font-weight: var(--weight-bold);
  line-height: var(--leading-tight);
}
.fighter-id small { font-size: var(--step--2); line-height: 1.4; }

.purse { font-weight: var(--weight-bold); color: var(--ink-muted); }
.purse.won { color: var(--gold); }
.purse.lost { color: var(--crimson); }
.purse::before {
  content: "";
  display: inline-block;
  width: 0.55em;
  height: 0.55em;
  margin-right: 0.3em;
  border-radius: 50%;
  background: currentColor;
  vertical-align: 0.05em;
}

.marquee {
  grid-column: 1 / -1;
  display: grid;
  justify-items: center;
  gap: var(--space-1);
  min-width: 0;
  padding: var(--space-3) var(--space-2);
  border-radius: var(--radius-control);
  background: var(--bg-sunken);
  text-align: center;
}
.marquee.won-a { background: var(--cobalt-tint); }
.marquee.won-b { background: var(--crimson-tint); }
.marquee-kicker,
.marquee-score {
  font-family: var(--font-mono);
  font-size: var(--step--2);
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--ink-muted);
}
.marquee-name {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-stretch: var(--stretch-display);
  font-size: var(--step-3);
  font-weight: var(--weight-black);
  line-height: var(--leading-tight);
  letter-spacing: -0.01em;
  color: var(--ink);
}
.marquee.won-a .marquee-score { color: var(--cobalt); }
.marquee.won-b .marquee-score { color: var(--crimson); }

.lights {
  grid-column: 1 / -1;
  display: flex;
  justify-content: center;
  min-width: 0;
}
.pips {
  --pip-width: 18px;
  --pip-gap: 5px;
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
  height: 8px;
  border-radius: 2px;
  background: var(--carbon-700);
}
.pips li.won-a { background: var(--cobalt); }
.pips li.won-b { background: var(--crimson); }
.pips li.drawn { background: var(--ink-secondary); }

.sheet-wrap {
  flex: 1 1 auto;
  min-height: 0;
  margin-top: var(--space-4);
  overflow: auto;
  background: var(--bg-sunken);
  border: 1px solid var(--line);
  border-radius: var(--radius-control);
  scrollbar-gutter: stable;
}
.sheet-wrap::-webkit-scrollbar { width: 6px; }
.sheet-wrap::-webkit-scrollbar-track { background: transparent; }
.sheet-wrap::-webkit-scrollbar-thumb { background: var(--line-strong); border-radius: 3px; }
.sheet-wrap::-webkit-scrollbar-button { display: none; }
@supports not selector(::-webkit-scrollbar) {
  .sheet-wrap { scrollbar-width: thin; scrollbar-color: var(--line-strong) transparent; }
}

.sheet {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
  font-size: var(--step--1);
}
.sheet th,
.sheet td {
  padding: var(--space-1) var(--space-2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: left;
  font-weight: var(--weight-medium);
}
.sheet thead th,
.sheet tbody th {
  font-family: var(--font-mono);
  font-size: var(--step--2);
  font-weight: var(--weight-medium);
  letter-spacing: var(--tracking-label);
  text-transform: uppercase;
  color: var(--ink-muted);
}
.sheet thead th {
  position: sticky;
  top: 0;
  background: var(--bg-sunken);
}
.sheet tbody th,
.sheet td { border-top: 1px solid var(--line); }
.sheet .side-b { text-align: right; }
.sheet thead .side-a { color: var(--cobalt); }
.sheet thead .side-b { color: var(--crimson); }
.sheet td.is-win { color: var(--side); font-weight: var(--weight-bold); }
.sheet td.is-lose { color: var(--ink-muted); }
.sheet td.is-draw { color: var(--ink-secondary); }
.sheet th:first-child,
.sheet td:first-child {
  width: 2rem;
  padding-inline: var(--space-1);
  letter-spacing: 0;
}

@media (prefers-reduced-motion: reduce) {
  .chat-open { transition: none; }
}
</style>
