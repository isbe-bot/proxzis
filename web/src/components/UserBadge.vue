<script setup>
import { computed } from 'vue'
import BotAvatar from './BotAvatar.vue'
import { botFacts } from '../utils/presentation.js'
import { number } from '../utils/arena.js'
const props = defineProps({ bot: { type: Object, required: true }, facing: { type: String, default: 'left' }, interactive: { type: Boolean, default: true } })
defineEmits(['profile'])
const facts = computed(() => botFacts(props.bot))
</script>
<template>
  <component :is="interactive ? 'button' : 'div'" class="user-badge" :class="{ 'faces-left': facing === 'right' }" :aria-label="interactive ? `View ${bot.name} profile` : undefined" @click="interactive && $emit('profile', bot)">
    <BotAvatar :bot="bot" /><span class="user-badge-copy"><strong>{{ bot.name }}</strong><small>{{ facts.tier }} · {{ number(bot.rating) }} rating</small><span class="badge-record mono" :aria-label="`${facts.wins} wins, ${facts.losses} losses, ${facts.draws} draws`">{{ facts.record }} <span class="muted">W-L-D</span></span><span class="badge-balance">● {{ number(facts.balance) }} <span class="muted">demo marbles</span></span></span>
  </component>
</template>
