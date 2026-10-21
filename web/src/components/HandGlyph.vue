<script setup>
import { computed } from 'vue'
import { STANCE_GLYPHS, stanceName } from '../utils/presentation.js'

const props = defineProps({
  stance: { type: String, required: true },
  side: { type: String, default: 'a' },
  owner: { type: String, required: true },
  concealed: Boolean,
  outcome: { type: String, default: null },
  // Replaces the generated accessible name.
  label: { type: String, default: null },
})

const handSrc = computed(() => {
  const isRight = props.side === 'b' || props.side === 'right'
  const prefix = isRight ? 'right' : 'left'
  const pose = props.concealed || !STANCE_GLYPHS[props.stance] ? 'rock' : props.stance
  return `/assets/hands/${prefix}_${pose}.png`
})
</script>

<template>
  <span
    class="hand-symbol"
    :class="[
      `side-${side}`,
      !concealed && outcome ? `hand-${outcome}` : null
    ]"
    role="img"
    :aria-label="label || `${owner}: ${concealed ? 'closed fist, stance concealed' : stanceName(stance)}${!concealed && outcome ? `, ${outcome === 'win' ? 'round won' : outcome === 'lose' ? 'round lost' : 'round drawn'}` : ''}`"
  >
    <img
      class="hand-img"
      :src="handSrc"
      :alt="concealed ? 'Closed fist' : stanceName(stance)"
      aria-hidden="true"
      draggable="false"
    />    <span class="hand-symbol-glyph sr-only" aria-hidden="true">{{ concealed ? '✊' : STANCE_GLYPHS[stance] }}</span>
  </span>
</template>

<style scoped>
.hand-symbol {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  vertical-align: middle;
  flex-shrink: 0;
  transition: opacity var(--dur-slow) var(--ease);
  position: relative;
}

.hand-img {
  display: block;
  width: var(--hand-size, 1.5em);
  height: auto;
  max-height: calc(var(--hand-size, 1.5em) * 1.1);
  object-fit: contain;
  pointer-events: none;
  filter: drop-shadow(var(--shadow-object));
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border-width: 0;
}

/* Outcome states: winning hand stays full, losing hand fades to transparent but still visible */
.hand-win {
  opacity: 1;
}

.hand-lose {
  opacity: 0.32;
}

.hand-draw {
  opacity: 0.85;
}
</style>
