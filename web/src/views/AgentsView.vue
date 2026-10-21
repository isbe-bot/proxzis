<script setup>
import { computed } from 'vue'
import BotCard from '../components/BotCard.vue'
import { useCatalog } from '../catalogStore.js'

const { allBots } = useCatalog()
const agents = computed(() => [...allBots.value].sort((x, y) => y.rating - x.rating))
</script>

<template>
  <div class="page container">
    <header class="page-head">
      <div class="page-head-copy">
        <h1 id="view-heading" class="page-title t-display" tabindex="-1">Agents</h1>
      </div>
      <p v-if="agents.length" class="page-stat"><strong>{{ agents.length }}</strong><span class="t-label">Agents</span></p>
    </header>

    <div v-if="agents.length" class="agent-grid">
      <BotCard v-for="bot in agents" :key="bot.id" :bot="bot" />
    </div>
    <p v-else class="status-line" role="status">Loading the agents.</p>

    <p class="footnote">Tendencies describe each bot's seeded strategy, not guaranteed moves. Records and Glicko-2 ratings come from the seeded archive.</p>
  </div>
</template>

<style scoped>
.agent-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 30rem), 1fr));
  gap: var(--space-5);
}
</style>
