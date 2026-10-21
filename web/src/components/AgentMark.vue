<script setup>
import { computed } from 'vue'

const props = defineProps({
  bot: { type: Object, required: true },
  side: { type: String, default: null },
  size: { type: Number, default: 32 },
  // Replaces the initial, e.g. a running score.
  label: { type: [String, Number], default: null },
})

const initial = computed(() => props.label ?? (props.bot.name || '?').trim().charAt(0).toUpperCase())
</script>

<template>
  <span
    class="agent-mark"
    :class="[side ? `side-${side} on-side` : null, { house: bot.house }]"
    :style="{ '--mark-size': `${size}px` }"
    aria-hidden="true"
  >
    <svg viewBox="0 0 24 24" focusable="false">
      <polygon points="7.65,1.5 16.35,1.5 22.5,7.65 22.5,16.35 16.35,22.5 7.65,22.5 1.5,16.35 1.5,7.65" />
    </svg>
    <span v-if="size >= 24" class="agent-initial" :class="{ 't-num': label != null }">{{ initial }}</span>
  </span>
</template>

<style scoped>
.agent-mark {
  position: relative;
  display: inline-grid;
  place-items: center;
  width: var(--mark-size);
  height: var(--mark-size);
  flex-shrink: 0;
}
.agent-mark svg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}
.agent-mark polygon {
  fill: transparent;
  stroke: var(--line-strong);
  stroke-width: 1.5;
  stroke-linejoin: round;
  vector-effect: non-scaling-stroke;
}
.agent-mark.house polygon { fill: var(--carbon-700); }
.agent-mark.on-side polygon { stroke: var(--side); fill: var(--side-tint); }
.agent-initial {
  position: relative;
  font-stretch: var(--stretch-display);
  font-weight: var(--weight-bold);
  font-size: calc(var(--mark-size) * 0.44);
  line-height: 1;
  color: var(--ink);
}
</style>
