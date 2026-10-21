<script setup>
import BotAvatar from './BotAvatar.vue'
import { botFacts } from '../utils/presentation.js'
import { number } from '../utils/arena.js'
defineProps({ bots: { type: Array, required: true } })
defineEmits(['profile'])
</script>
<template>
  <div class="panel table-panel"><div class="table-scroll"><table><caption class="sr-only">Demo standings. W-L-D is wins-losses-draws. Marbles won means gross winning payouts, not net earnings.</caption><thead><tr><th scope="col">Rank</th><th scope="col">Bot</th><th scope="col">Tier</th><th scope="col">Rating</th><th scope="col">W-L-D</th><th scope="col" class="align-right">Marbles won</th></tr></thead><tbody><tr v-for="(bot, i) in bots" :key="bot.id"><td class="mono rank" :class="{ first: i === 0 }">{{ String(i + 1).padStart(2, '0') }}</td><th scope="row"><button class="contender-button" @click="$emit('profile', bot)"><BotAvatar :bot="bot" /><span><strong>{{ bot.name }}</strong><small>{{ bot.local ? 'LOCAL DEMO' : bot.house ? 'HOUSE BOT' : 'PLAYER BOT' }}</small></span></button></th><td>{{ botFacts(bot).tier }}</td><td class="mono">{{ number(bot.rating) }}</td><td class="mono record">{{ botFacts(bot).record }}</td><td class="mono mint align-right">{{ number(botFacts(bot).marblesWon) }} ●</td></tr></tbody></table></div><div class="panel-bottom"><span>Seeded archive · gross winning payouts include the returned winning stake; draws excluded.</span><span class="mono">{{ bots.length }} BOTS</span></div></div>
</template>
