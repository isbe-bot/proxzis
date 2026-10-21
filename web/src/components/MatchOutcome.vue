<script setup>
import AgentMark from './AgentMark.vue'
defineProps({ match: { type: Object, required: true } })
</script>

<template>
  <span
    class="match-outcome"
    :class="match.winner === 'draw' ? 'outcome-draw' : `outcome-win side-${match.winner}`"
    role="img"
    :aria-label="match.winner === 'draw' ? 'Draw: both stakes returned' : `Winner: ${match[match.winner].name}`"
  >
    <template v-if="match.winner === 'draw'">
      <span class="draw-mark" aria-hidden="true">=</span>
      <span class="outcome-text">Draw · stakes returned</span>
    </template>
    <template v-else>
      <AgentMark :bot="match[match.winner]" :side="match.winner" :size="20" />
      <span class="outcome-text">{{ match[match.winner].name }} won</span>
    </template>
  </span>
</template>

<style scoped>
.match-outcome {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  max-width: 100%;
  min-width: 0;
  height: 2rem;
  padding: 0 var(--space-3) 0 var(--space-1);
  border: 1px solid var(--line);
  border-radius: var(--radius-control);
  background: var(--surface-raised);
  font-size: var(--step--1);
  font-weight: var(--weight-medium);
  color: var(--ink);
  vertical-align: middle;
}
.outcome-draw { padding-left: var(--space-2); color: var(--ink-secondary); }
.draw-mark {
  display: grid;
  flex: none;
  place-items: center;
  width: 1.25rem;
  height: 1.25rem;
  font-family: var(--font-mono);
  font-weight: var(--weight-bold);
  color: var(--ink-muted);
}
.outcome-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
