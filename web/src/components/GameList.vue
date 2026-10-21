<script setup>
import { ArrowRight, ChevronDown } from 'lucide-vue-next'
import AppLink from './AppLink.vue'
import AgentMark from './AgentMark.vue'
import RoundList from './RoundList.vue'
import { dateLabel, matchPath, resultLine } from '../utils/presentation.js'
import { number } from '../utils/arena.js'

defineProps({
  matches: { type: Array, required: true },
  expandable: Boolean,
  note: Boolean,
  label: { type: String, default: 'Matches' },
})
</script>

<template>
  <div class="game-list">
    <ol class="game-rows" :aria-label="label">
      <li v-for="match in matches" :key="match.id" class="game-row">
        <AppLink
          :to="matchPath(match)"
          class="game-link"
          :aria-label="`Replay ${match.id}: ${match.a.name} ${match.score.a}, ${match.b.name} ${match.score.b}. ${resultLine(match)}.`"
        >
          <span class="game-meta t-label"><span>{{ match.id }}</span><span>{{ dateLabel(match.date) }}</span></span>
          <span class="game-side side-a" :class="{ lost: match.winner === 'b' }">
            <AgentMark :bot="match.a" side="a" :size="28" />
            <span class="game-name" :title="match.a.name">{{ match.a.name }}</span>
          </span>
          <span class="game-center">
            <span class="game-score"><span class="score-a">{{ match.score.a }}</span><span class="game-clash" aria-hidden="true">–</span><span class="score-b">{{ match.score.b }}</span></span>
          </span>
          <span class="game-side side-b" :class="{ lost: match.winner === 'a' }">
            <span class="game-name" :title="match.b.name">{{ match.b.name }}</span>
            <AgentMark :bot="match.b" side="b" :size="28" />
          </span>
          <span class="game-stake"><span class="marble" aria-hidden="true"></span><span class="truncate"><strong class="t-num">{{ number(match.stake) }}</strong> a side</span></span>
          <span class="game-result" :title="resultLine(match)">{{ resultLine(match) }}</span>
          <ArrowRight class="game-go" :size="18" aria-hidden="true" />
        </AppLink>
        <details v-if="expandable" class="game-rounds">
          <summary :aria-label="`Round stances and results for ${match.id}`">{{ match.rounds.length }} rounds · stances and results <ChevronDown class="game-rounds-icon" :size="16" aria-hidden="true" /></summary>
          <RoundList :match="match" compact />
        </details>
      </li>
    </ol>
    <p v-if="note && matches.length" class="footnote">Seeded demo matches, best of seven. Stakes are per agent, per series, not per round.</p>
  </div>
</template>

<style scoped>
.game-list { container: games / inline-size; width: 100%; }
.game-rows { margin: 0; padding: 0; list-style: none; border-top: 1px solid var(--line); }
.game-row { border-bottom: 1px solid var(--line); }

.game-link {
  display: grid;
  grid-template-columns: 9.5rem minmax(0, 1fr) 6rem minmax(0, 1fr) 6.5rem 10rem 1.25rem;
  align-items: center;
  gap: var(--space-4);
  min-height: 5rem;
  padding: var(--space-3) var(--space-2);
  color: var(--ink);
  text-decoration: none;
  transition: background-color var(--dur-fast) var(--ease);
}
.game-link:hover { background: var(--surface); }

.game-meta { display: flex; flex-direction: column; min-width: 0; line-height: 1.6; }
.game-meta > span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.game-side {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-width: 0;
  font-weight: var(--weight-bold);
}
.game-side.side-b { justify-content: flex-end; text-align: right; }
.game-side.lost { color: var(--ink-muted); font-weight: var(--weight-medium); }
.game-name { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.game-center { display: flex; justify-content: center; min-width: 0; }
.game-score {
  display: flex;
  gap: var(--space-2);
  font-stretch: var(--stretch-display);
  font-size: var(--step-4);
  font-weight: var(--weight-black);
  line-height: 1;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.score-a { color: var(--cobalt); }
.score-b { color: var(--crimson); }
.game-clash { color: var(--line-strong); font-weight: var(--weight-regular); }

.game-stake {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
  font-size: var(--step--1);
  color: var(--ink-muted);
  white-space: nowrap;
}
.game-stake strong { color: var(--gold); font-weight: var(--weight-bold); }
.marble { flex: none; width: 8px; height: 8px; border-radius: 50%; background: var(--gold); }
.game-result {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--step--1);
  color: var(--ink-secondary);
}
.game-go { color: var(--ink-muted); transition: transform var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease); }
.game-link:hover .game-go { color: var(--ink); transform: translateX(3px); }

.game-rounds { padding: 0 var(--space-2); }
.game-rounds summary {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  min-height: 2.75rem;
  margin-top: calc(-1 * var(--space-3));
  font-size: var(--step--1);
  font-weight: var(--weight-medium);
  color: var(--ink-secondary);
  list-style: none;
  cursor: pointer;
}
.game-rounds summary::-webkit-details-marker { display: none; }
.game-rounds summary:hover { color: var(--ink); }
.game-rounds-icon { color: var(--ink-muted); transition: transform var(--dur-base) var(--ease); }
.game-rounds[open] .game-rounds-icon { transform: rotate(180deg); }
.game-rounds[open] { padding-bottom: var(--space-4); }

@container games (max-width: 899px) {
  .game-link {
    grid-template-columns: minmax(0, 1fr) 5.5rem minmax(0, 1fr) 1.25rem;
    row-gap: var(--space-2);
  }
  .game-meta { grid-column: 1 / -1; flex-direction: row; gap: var(--space-4); }
  .game-stake, .game-result { grid-row: 3; }
  .game-stake { grid-column: 1; }
  .game-result { grid-column: 2 / 4; text-align: right; }
  .game-go { grid-row: 2; grid-column: 4; }
}

@container games (max-width: 559px) {
  .game-link { grid-template-columns: minmax(0, 1fr) 4.5rem minmax(0, 1fr); padding-inline: 0; }
  .game-result { grid-column: 2 / -1; }
  .game-go, .game-side .agent-mark { display: none; }
  .game-score { font-size: var(--step-3); }
  .game-rounds { padding: 0; }
}

@media (prefers-reduced-motion: reduce) {
  .game-link, .game-go, .game-rounds-icon { transition: none; }
  .game-link:hover .game-go { transform: none; }
}
</style>
