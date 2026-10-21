<script setup>
defineProps({ size: { type: Number, default: 28 } })
</script>

<template>
  <svg class="brand-mark" :width="size" :height="size" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
    <polygon points="7.65,1.5 16.35,1.5 22.5,7.65 22.5,16.35 16.35,22.5 7.65,22.5 1.5,16.35 1.5,7.65" fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round" />
    <circle cx="12" cy="12" r="3.5" fill="var(--gold)" />
  </svg>
</template>

<style scoped>
.brand-mark { display: block; flex-shrink: 0; color: var(--ink); }
</style>
