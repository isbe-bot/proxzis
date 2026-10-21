<script setup>
import UserBadge from './UserBadge.vue'
import MatchOutcome from './MatchOutcome.vue'
import RoundList from './RoundList.vue'
import { dateLabel } from '../utils/presentation.js'
import { number } from '../utils/arena.js'
defineProps({ matches: { type: Array, required: true } })
defineEmits(['profile', 'replay'])
</script>
<template>
  <div class="game-list panel">
    <p class="game-list-note">Seeded demo games · Best-of-7 series · Wager is per bot, per series — not per round.</p>
    <div class="game-list-scroll" role="region" aria-label="Games; scroll horizontally on small screens" tabindex="0"><ol class="game-rows" aria-label="Match archive">
      <li v-for="match in matches" :key="match.id" class="game-row">
        <UserBadge :bot="match.a" @profile="$emit('profile', $event)" />
        <div class="game-center"><button class="game-score mono" :aria-label="`Replay ${match.id}, ${match.a.name} ${match.score.a} to ${match.b.name} ${match.score.b}`" @click="$emit('replay', match)"><span class="game-score-value">{{ match.score.a }}</span><span class="game-clash" aria-hidden="true">-</span><span class="game-score-value">{{ match.score.b }}</span></button><span class="eyebrow">BEST OF 7</span><strong class="game-wager">● {{ number(match.stake) }} / bot</strong><small>Series wager</small></div>
        <UserBadge :bot="match.b" facing="right" @profile="$emit('profile', $event)" />
        <div class="game-row-footer"><span>{{ match.id }} · {{ dateLabel(match.date) }}</span><MatchOutcome :match="match" /><button class="text-button small" @click="$emit('replay', match)">Replay ↗</button></div>
        <details class="game-rounds"><summary :aria-label="`Round stances and results for ${match.id}`">{{ match.rounds.length }} rounds · stances &amp; results</summary><RoundList :match="match" compact /></details>
      </li>
    </ol></div><p v-if="!matches.length" class="empty-copy">No games yet.</p>
  </div>
</template>
