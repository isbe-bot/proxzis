<script setup>
import { computed } from 'vue'
import AppLink from './AppLink.vue'
import AgentMark from './AgentMark.vue'
import { agentPath, botFacts } from '../utils/presentation.js'
import { number } from '../utils/arena.js'

const props = defineProps({
  bots: { type: Array, required: true },
  label: { type: String, default: 'Standings' },
})
const rows = computed(() => [...props.bots].sort((x, y) => y.rating - x.rating))
const line = computed(() => rows.value.reduce((lowest, bot) => {
  if (!bot.house) return lowest
  if (!lowest || bot.rating < lowest.rating) return bot
  return lowest
}, null))
const below = bot => line.value && bot.id !== line.value.id && bot.rating < line.value.rating
</script>

<template>
  <div class="table-wrap" role="region" :aria-label="label" tabindex="0">
    <table class="standings">
      <caption class="sr-only">Archive standings. W-L-D is wins, losses and draws. Marbles won is gross winning payouts.</caption>
      <colgroup>
        <col class="col-rank" />
        <col />
        <col class="col-rating" />
        <col class="col-record" />
        <col class="col-marbles" />
      </colgroup>
      <thead>
        <tr>
          <th scope="col">#</th>
          <th scope="col">Agent</th>
          <th scope="col" class="num">Rating</th>
          <th scope="col" class="num">W-L-D</th>
          <th scope="col" class="num">Marbles won</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="!rows.length"><td colspan="5" class="empty">Loading standings.</td></tr>
        <tr v-for="(bot, i) in rows" :key="bot.id" :class="{ 'is-line': line && bot.id === line.id, 'is-below': below(bot) }">
          <td class="t-num rank">{{ i + 1 }}</td>
          <th scope="row">
            <AppLink :to="agentPath(bot)" class="agent-cell">
              <AgentMark :bot="bot" :size="30" />
              <span class="agent-name" :title="bot.name">{{ bot.name }}</span>
              <span v-if="bot.house" class="t-label house-tag">House</span>
            </AppLink>
          </th>
          <td class="num t-num">{{ number(bot.rating) }}<span v-if="bot.rd" class="rd"> ±{{ bot.rd }}</span></td>
          <td class="num t-num">{{ botFacts(bot).record }}</td>
          <td class="num t-num marbles">{{ number(botFacts(bot).marblesWon) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.table-wrap {
  overflow-x: auto;
  border: 1px solid var(--line);
  border-radius: var(--radius-surface);
  background: var(--surface);
}
.standings {
  width: 100%;
  min-width: 34rem;
  table-layout: fixed;
  border-collapse: collapse;
}
.col-rank { width: 3.5rem; }
.col-rating { width: 8.5rem; }
.col-record { width: 6.5rem; }
.col-marbles { width: 8.5rem; }
.standings th,
.standings td {
  height: 3.75rem;
  padding: 0 var(--space-5);
  border-bottom: 1px solid var(--line);
  background: none;
  text-align: left;
  font: inherit;
  font-size: var(--step-0);
  letter-spacing: normal;
  color: var(--ink);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.standings thead th {
  height: 3rem;
  font-family: var(--font-mono);
  font-size: var(--step--2);
  font-weight: var(--weight-medium);
  letter-spacing: var(--tracking-label);
  text-transform: uppercase;
  color: var(--ink-muted);
}
.standings tbody tr:last-child > * { border-bottom: 0; }
.standings tbody tr:hover > * { background: var(--surface-hover); }
.standings .num { text-align: right; }
.standings .rank { color: var(--ink-muted); }
.standings .t-num { font-size: var(--step--1); }
.standings .rd { color: var(--ink-muted); font-size: var(--step--2); }
.standings .marbles { color: var(--gold); }
.standings tr.is-line > * { border-bottom: 2px solid var(--ink-secondary); }
.standings tr.is-below .agent-name,
.standings tr.is-below .num:not(.marbles) { color: var(--ink-muted); }
.standings .empty { color: var(--ink-muted); text-align: center; }

.agent-cell {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-width: 0;
  min-height: 2.75rem;
  color: inherit;
  text-decoration: none;
}
.agent-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: var(--weight-bold);
}
.agent-cell:hover .agent-name { text-decoration: underline; text-underline-offset: 0.25em; }
.house-tag { flex: none; }

@media (max-width: 720px) {
  .standings th, .standings td { padding-inline: var(--space-3); }
  .col-rank { width: 2.5rem; }
  .col-rating { width: 7rem; }
  .col-record { width: 5.5rem; }
  .col-marbles { width: 6.5rem; }
}
</style>
