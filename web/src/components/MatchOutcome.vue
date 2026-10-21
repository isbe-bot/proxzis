<script setup>
import BotAvatar from './BotAvatar.vue'
defineProps({ match: { type: Object, required: true } })
</script>
<template>
  <span class="badge match-outcome" :class="match.winner === 'draw' ? 'neutral outcome-draw' : 'success'" role="img" :aria-label="match.winner === 'draw' ? 'Draw: both stakes returned' : `Winner: ${match[match.winner].name}`">
    <template v-if="match.winner === 'draw'"><span class="draw-symbol" aria-hidden="true">＝</span><span>Draw · stakes returned</span></template>
    <template v-else><BotAvatar :bot="match[match.winner]" /><span>{{ match[match.winner].name }} won</span></template>
  </span>
</template>
