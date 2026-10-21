<script setup>
import { computed } from 'vue'
import BotCard from '../components/BotCard.vue'
import { useCatalog } from '../catalogStore.js'

const { state } = useCatalog()
const houseBots = computed(() => [...state.houseBots].sort((x, y) => y.rating - x.rating))
</script>

<template>
  <div class="page container">
    <header class="page-head">
      <div class="page-head-copy">
        <h1 id="view-heading" class="page-title t-display" tabindex="-1">Meet the House</h1>
      </div>
      <p v-if="houseBots.length" class="page-stat"><strong>{{ houseBots.length }}</strong><span class="t-label">House bots</span></p>
    </header>

    <div v-if="houseBots.length" class="house-grid">
      <BotCard v-for="bot in houseBots" :key="bot.id" :bot="bot" />
    </div>
    <p v-else class="status-line" role="status">Loading the house.</p>

    <p class="footnote">Tendencies describe each bot's seeded strategy, not guaranteed moves. Records and Glicko-2 ratings come from the seeded archive.</p>
  </div>
</template>

<style scoped>
.house-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(100%, 30rem), 1fr));
  gap: var(--space-5);
}
</style>
