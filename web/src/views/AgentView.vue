<script setup>
import { computed, ref, watch } from 'vue'
import { ArrowLeft } from 'lucide-vue-next'
import AgentMark from '../components/AgentMark.vue'
import BotCard from '../components/BotCard.vue'
import GameList from '../components/GameList.vue'
import { getBot, listMatches } from '../api/catalog.js'
import { botFacts } from '../utils/presentation.js'
import { number } from '../utils/arena.js'

const props = defineProps({ id: { type: String, required: true } })
const profile = ref(null)
const profileLog = ref([])
const profileError = ref('')
let controller = null

watch(() => props.id, async id => {
  controller?.abort()
  const current = new AbortController()
  controller = current
  profile.value = null
  profileLog.value = []
  profileError.value = ''
  try {
    const [bot, log] = await Promise.all([getBot(id, current.signal), listMatches({ bot: id, limit: 50 }, current.signal)])
    if (current !== controller) return
    profile.value = bot
    profileLog.value = log.items
    document.title = `${bot.name} · BOTTAGON`
  } catch (error) {
    if (error.name === 'AbortError' || current !== controller) return
    profileError.value = error.status === 404 ? 'There is no agent with that id in the archive.' : (error.message || 'The agent profile is unavailable.')
  }
}, { immediate: true })

const kicker = computed(() => profile.value
  ? `${profile.value.house ? 'House bot' : 'Contender'} · ${botFacts(profile.value).tier} · ${number(profile.value.rating)} rating`
  : 'Agent')
</script>

<template>
  <div class="page container">
    <RouterLink :to="{ path: '/', hash: '#standings' }" aria-current-value="false" class="back-link"><ArrowLeft :size="16" aria-hidden="true" /> Standings</RouterLink>
    <header class="page-head">
      <div class="page-head-copy agent-head">
        <AgentMark v-if="profile" :bot="profile" :size="64" class="agent-head-mark" />
        <div class="agent-head-copy">
          <p class="page-kicker t-label">{{ kicker }}</p>
          <h1 id="view-heading" class="page-title t-display" tabindex="-1">{{ profile?.name || 'Agent profile' }}</h1>
        </div>
      </div>
    </header>

    <p v-if="profileError" class="notice" role="alert">{{ profileError }} <RouterLink :to="{ path: '/', hash: '#standings' }" aria-current-value="false" class="btn-text">See the standings</RouterLink></p>
    <p v-else-if="!profile" class="status-line" role="status">Loading the agent profile.</p>

    <div v-if="profile" class="agent-layout">
      <BotCard :bot="profile" :show-profile="false" :header="false" class="agent-facts" />
      <section class="agent-log" aria-labelledby="log-heading">
        <div class="log-head">
          <h2 id="log-heading" class="log-title">Match log</h2>
          <span class="t-label">{{ profileLog.length }} {{ profileLog.length === 1 ? 'match' : 'matches' }}</span>
        </div>
        <GameList v-if="profileLog.length" :matches="profileLog" expandable note :label="`${profile.name} match log`" />
        <div v-else class="empty-state"><h2>No matches yet.</h2><p>This agent has not played in the archive.</p></div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.agent-head { display: flex; align-items: center; gap: var(--space-5); }
.agent-head-mark { flex: none; }
.agent-head-copy { min-width: 0; }
.agent-head .page-title { font-size: var(--step-5); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.agent-layout {
  display: grid;
  grid-template-columns: minmax(0, 24rem) minmax(0, 1fr);
  align-items: start;
  gap: var(--space-7);
}
.agent-facts { position: sticky; top: calc(var(--header-height) + var(--space-5)); }
.log-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-4);
  margin-bottom: var(--space-4);
}
.log-title {
  margin: 0;
  font-stretch: var(--stretch-display);
  font-size: var(--step-3);
  font-weight: var(--weight-black);
  letter-spacing: var(--tracking-display);
}

@media (max-width: 960px) {
  .agent-layout { grid-template-columns: minmax(0, 1fr); }
  .agent-facts { position: static; }
}
@media (max-width: 480px) {
  .agent-head { gap: var(--space-4); }
}
</style>
