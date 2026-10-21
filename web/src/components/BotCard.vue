<script setup>
import { computed } from 'vue'
import UserBadge from './UserBadge.vue'
import { botFacts, dateLabel, STANCE_GLYPHS } from '../utils/presentation.js'
import { number, signed } from '../utils/arena.js'
const props = defineProps({ bot: { type: Object, required: true }, showProfile: { type: Boolean, default: true } })
defineEmits(['profile'])
const facts = computed(() => botFacts(props.bot))
</script>
<template>
  <article class="bot-card panel" :aria-label="`${bot.name} bot facts`">
    <div class="bot-card-top"><span class="badge neutral">{{ bot.house ? 'HOUSE BOT' : 'PLAYER BOT' }}</span><span class="eyebrow">{{ bot.local ? 'LOCAL DEMO' : 'SEEDED DEMO' }}</span></div>
    <UserBadge :bot="bot" :interactive="showProfile" @profile="$emit('profile', $event)" />
    <p class="house-persona">{{ bot.persona }}</p><blockquote v-if="bot.quote">“{{ bot.quote }}”</blockquote>
    <dl class="bot-facts"><div><dt>Marble balance</dt><dd class="mint">{{ number(facts.balance) }} ●</dd></div><div><dt>Matches played</dt><dd>{{ facts.played }}</dd></div><div><dt>Record · W-L-D</dt><dd class="record">{{ facts.record }}</dd></div><div><dt>Marbles won · gross</dt><dd>{{ number(facts.marblesWon) }} ●</dd></div><div><dt>Net earnings</dt><dd :class="facts.earnings >= 0 ? 'mint' : 'coral'">{{ signed(facts.earnings) }} ●</dd></div><div><dt>Last played</dt><dd>{{ dateLabel(facts.lastPlayed) }}</dd></div></dl>
    <p class="field-help">Demo balance = 1,000 seeded opening marbles + archive net earnings. No real funds.</p>
    <h3 class="eyebrow tactics-heading">Tactical tendencies</h3><div class="tactics-bar" aria-hidden="true"><span v-for="(value, i) in bot.tactics" :key="i" :class="['strike', 'feint', 'parry'][i]" :style="{ width: value + '%' }"></span></div><div class="tactics-legend"><span v-for="(value, i) in bot.tactics" :key="i">{{ Object.values(STANCE_GLYPHS)[i] }} {{ ['Strike', 'Feint', 'Parry'][i] }} <strong>{{ value }}%</strong></span></div>
    <p v-if="bot.local" class="demo-disclosure">Local-only recruit. Not connected to server matchmaking.</p><button v-if="showProfile" class="text-button bot-profile-link" @click="$emit('profile', bot)">Profile & match log ↗</button>
  </article>
</template>
