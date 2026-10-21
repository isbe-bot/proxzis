<script setup>
import { computed } from 'vue'
import AppLink from './AppLink.vue'
import AgentMark from './AgentMark.vue'
import { agentPath, botFacts } from '../utils/presentation.js'
import { number } from '../utils/arena.js'

const props = defineProps({
  bot: { type: Object, required: true },
  side: { type: String, default: null },
  align: { type: String, default: 'start' },
  link: { type: Boolean, default: true },
  size: { type: Number, default: 44 },
})
const facts = computed(() => botFacts(props.bot))
</script>

<template>
  <component
    :is="link ? AppLink : 'div'"
    v-bind="link ? { to: agentPath(bot), 'aria-label': `${bot.name} profile` } : {}"
    class="agent-badge"
    :class="[{ 'is-end': align === 'end', 'is-link': link }, side ? `side-${side}` : null]"
  >
    <AgentMark :bot="bot" :side="side" :size="size" />
    <span class="badge-copy">
      <strong class="badge-name" :title="bot.name">{{ bot.name }}</strong>
      <span class="badge-meta">{{ facts.tier }} · <span>{{ number(bot.rating) }}</span> rating</span>
      <span class="badge-meta">
        <span class="badge-record" :aria-label="`${facts.wins} wins, ${facts.losses} losses, ${facts.draws} draws`">{{ facts.record }}</span> W-L-D ·
        <span class="badge-marbles">{{ number(facts.balance) }}</span> marbles
      </span>
    </span>
  </component>
</template>

<style scoped>
.agent-badge {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-width: 0;
  max-width: 100%;
  color: var(--ink);
  text-decoration: none;
}
.agent-badge.is-end { flex-direction: row-reverse; text-align: right; }
.badge-copy { display: flex; flex-direction: column; min-width: 0; }
.badge-name,
.badge-meta {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.badge-name {
  font-size: var(--step-1);
  font-weight: var(--weight-bold);
  line-height: var(--leading-tight);
  text-decoration: underline;
  text-decoration-color: transparent;
  text-underline-offset: 0.2em;
  transition: text-decoration-color var(--dur-fast) var(--ease);
}
.is-link:hover .badge-name { text-decoration-color: var(--side, var(--ink)); }
.badge-meta { font-size: var(--step--1); line-height: 1.45; color: var(--ink-muted); }
.badge-record { color: var(--ink-secondary); }
.badge-marbles { color: var(--gold); }

@media (prefers-reduced-motion: reduce) {
  .badge-name { transition: none; }
}
</style>
