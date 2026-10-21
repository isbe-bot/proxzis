<script setup>
import HandGlyph from './HandGlyph.vue'
import { stanceName } from '../utils/presentation.js'
import { resolveClash } from '../utils/arena.js'
defineProps({ match: { type: Object, required: true }, compact: Boolean })
</script>
<template>
  <ol class="round-list" :class="{ compact }" :aria-label="`Round results for ${match.id}`">
    <li v-for="(round, i) in match.rounds" :key="i" class="round-item">
      <span class="round-number">{{ String(i + 1).padStart(2, '0') }}</span>
      <div class="round-clash"><span class="stance" :class="round.a"><HandGlyph :stance="round.a" side="a" :owner="match.a.name" /><span class="stance-name" aria-hidden="true">{{ stanceName(round.a) }}</span></span><span class="clash-divider" aria-hidden="true">×</span><span class="stance" :class="round.b"><HandGlyph :stance="round.b" side="b" :owner="match.b.name" /><span class="stance-name" aria-hidden="true">{{ stanceName(round.b) }}</span></span></div>
      <span class="round-winner">{{ resolveClash(round.a, round.b) === 'draw' ? 'Draw' : match[resolveClash(round.a, round.b)].name }}</span>
      <div v-if="!compact" class="round-quotes"><p><strong>{{ match.a.name }}:</strong> “{{ round.quoteA }}”</p><p><strong>{{ match.b.name }}:</strong> “{{ round.quoteB }}”</p></div>
    </li>
  </ol>
</template>
