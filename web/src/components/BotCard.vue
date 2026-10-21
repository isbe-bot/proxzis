<script setup>
import { computed } from 'vue'
import { ArrowRight } from 'lucide-vue-next'
import AppLink from './AppLink.vue'
import AgentMark from './AgentMark.vue'
import { agentPath, botFacts, dateLabel } from '../utils/presentation.js'
import { number, signed } from '../utils/arena.js'

const props = defineProps({
  bot: { type: Object, required: true },
  showProfile: { type: Boolean, default: true },
  header: { type: Boolean, default: true },
})
const facts = computed(() => botFacts(props.bot))
const STANCES = ['Rock', 'Paper', 'Scissors']
const kind = computed(() => props.bot.house ? 'House bot' : 'Contender')
</script>

<template>
  <article class="agent-card" :class="{ 'is-card': showProfile }" :aria-label="`${bot.name} facts`">
    <header v-if="header" class="card-head">
      <AgentMark :bot="bot" :size="48" />
      <div class="card-id">
        <h2 class="card-name" :title="bot.name">{{ bot.name }}</h2>
        <p class="card-kind t-label">{{ kind }} · {{ facts.tier }} · <span class="t-num">{{ number(bot.rating) }}</span></p>
      </div>
    </header>

    <p v-if="bot.persona" class="card-persona">{{ bot.persona }}</p>
    <blockquote v-if="bot.quote" class="card-quote" :title="showProfile ? bot.quote : undefined">“{{ bot.quote }}”</blockquote>

    <dl class="card-facts">
      <div><dt class="t-label">Marble balance</dt><dd class="t-num is-gold">{{ number(facts.balance) }}</dd></div>
      <div><dt class="t-label">Matches played</dt><dd class="t-num">{{ facts.played }}</dd></div>
      <div><dt class="t-label">Record · W-L-D</dt><dd class="record">{{ facts.record }}</dd></div>
      <div><dt class="t-label">Marbles won</dt><dd class="t-num is-gold">{{ number(facts.marblesWon) }}</dd></div>
      <div><dt class="t-label">Net earnings</dt><dd class="t-num">{{ signed(facts.earnings) }}</dd></div>
      <div><dt class="t-label">Last played</dt><dd class="is-date" :title="dateLabel(facts.lastPlayed)">{{ dateLabel(facts.lastPlayed) }}</dd></div>
    </dl>
    <p class="card-note">Demo balance = 1,000 seeded opening marbles + archive net earnings. Marbles won is gross winning payouts. No real funds.</p>

    <div v-if="bot.tactics?.length" class="tendencies">
      <h3 class="t-label">Tactical tendencies</h3>
      <div class="tendency-bar" aria-hidden="true">
        <span v-for="(value, i) in bot.tactics" :key="i" :class="`tend-${i}`" :style="{ flexGrow: value }"></span>
      </div>
      <ul class="tendency-legend">
        <li v-for="(value, i) in bot.tactics" :key="i"><span class="swatch" :class="`tend-${i}`" aria-hidden="true"></span>{{ STANCES[i] }} <strong class="t-num">{{ value }}%</strong></li>
      </ul>
    </div>

    <p v-if="bot.local" class="card-note">Local-only recruit. Not connected to server matchmaking.</p>
    <AppLink v-if="showProfile" :to="agentPath(bot)" class="btn-text card-link" :aria-label="`${bot.name} profile and match log`">Profile and match log <ArrowRight :size="16" aria-hidden="true" /></AppLink>
  </article>
</template>

<style scoped>
.agent-card {
  container: card / inline-size;
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  min-width: 0;
  padding: var(--space-6);
  border: 1px solid var(--line);
  border-radius: var(--radius-surface);
  background: var(--surface);
}

.card-head { display: flex; align-items: center; gap: var(--space-4); min-width: 0; }
.card-id { min-width: 0; }
.card-name {
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-stretch: var(--stretch-display);
  font-size: var(--step-3);
  font-weight: var(--weight-black);
  letter-spacing: var(--tracking-display);
  line-height: var(--leading-tight);
}
.card-kind { margin: var(--space-1) 0 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.card-persona { margin: 0; font-size: var(--step-0); color: var(--ink-secondary); }
.card-quote {
  margin: 0;
  padding-left: var(--space-4);
  border-left: 2px solid var(--line-strong);
  font-size: var(--step--1);
  color: var(--ink-muted);
}
.is-card .card-persona {
  display: -webkit-box;
  height: calc(3 * 1.55em);
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
  line-clamp: 3;
}
.is-card .card-quote { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.card-facts {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(9.5rem, 1fr));
  margin: 0;
  border-top: 1px solid var(--line);
  border-left: 1px solid var(--line);
}
.card-facts div {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: var(--space-2);
  min-width: 0;
  height: 4.75rem;
  padding: var(--space-3) var(--space-4);
  border-right: 1px solid var(--line);
  border-bottom: 1px solid var(--line);
}
.card-facts dt { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.card-facts dd {
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-stretch: var(--stretch-display);
  font-size: var(--step-2);
  font-weight: var(--weight-black);
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
  color: var(--ink);
}
.card-facts dd.record { font-size: var(--step-2); }
.card-facts dd.is-gold { color: var(--gold); }
.card-facts dd.is-date { font-stretch: var(--stretch-text); font-size: var(--step--1); font-weight: var(--weight-bold); line-height: 1.6; }
.card-note { margin: calc(-1 * var(--space-3)) 0 0; font-size: var(--step--2); color: var(--ink-muted); }

.tendencies h3 { margin: 0 0 var(--space-3); }
.tendency-bar { display: flex; gap: 2px; height: 8px; overflow: hidden; border-radius: 2px; }
.tendency-bar span { flex-basis: 0; min-width: 2px; }
.tend-0 { background: var(--ink); }
.tend-1 { background: var(--ink-muted); }
.tend-2 { background: var(--carbon-600); }
.tendency-legend {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2) var(--space-5);
  margin: var(--space-3) 0 0;
  padding: 0;
  list-style: none;
  font-size: var(--step--1);
  color: var(--ink-secondary);
}
.tendency-legend li { display: inline-flex; align-items: center; gap: var(--space-2); white-space: nowrap; }
.tendency-legend strong { color: var(--ink); font-weight: var(--weight-bold); }
.swatch { width: 10px; height: 10px; border-radius: 2px; }

.card-link { align-self: flex-start; margin-top: auto; }

@container card (max-width: 26rem) {
  .card-facts { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .card-facts div { padding: var(--space-3); }
  .card-facts dd.is-date { font-size: var(--step--2); line-height: 1.9; }
}
</style>
