<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { ArrowUpRight, ArrowRight, ChevronRight, ChevronDown, Plus, Search, X, Radio, Sparkles, Terminal, Activity, LockKeyhole } from 'lucide-vue-next'
import BotAvatar from './components/BotAvatar.vue'
import ArenaModal from './components/ArenaModal.vue'
import LiveMatch from './components/LiveMatch.vue'
import GameList from './components/GameList.vue'
import LeaderboardTable from './components/LeaderboardTable.vue'
import BotCard from './components/BotCard.vue'
import MatchReplay from './components/MatchReplay.vue'
import RegisterPage from './components/RegisterPage.vue'
import { getExhibition, getMatch, listBots, listMatches } from './api/catalog.js'

const activeView = ref('Overview')
const views = ['Overview', 'Register', 'House Bots', 'Spectator Feed']
const contenders = ref([])
const houseBots = ref([])
const allBots = computed(() => [...contenders.value, ...houseBots.value])
const recent = ref([])
const exhibition = ref(null)
const archiveTotal = ref(0)
const filteredMatches = ref([])
const filteredTotal = ref(0)
const profile = ref(null)
const profileLog = ref([])
const detail = ref(null)
const profileError = ref('')
const detailError = ref('')
const search = ref('')
const outcome = ref('all')
const stake = ref('all')
const loadError = ref('')
let matchController = null
let profileController = null
let detailController = null

async function loadMatches(reset) {
  matchController?.abort()
  const controller = new AbortController()
  matchController = controller
  const offset = reset ? 0 : filteredMatches.value.length
  try {
    const page = await listMatches({ q: search.value.trim(), outcome: outcome.value, stake: stake.value, limit: 6, offset }, controller.signal)
    if (controller !== matchController) return
    loadError.value = ''
    filteredTotal.value = page.total
    filteredMatches.value = reset ? page.items : filteredMatches.value.concat(page.items)
  } catch (error) {
    if (error.name === 'AbortError' || controller !== matchController) return
    loadError.value = error.message || 'The arena archive is unavailable.'
    if (reset) {
      filteredMatches.value = []
      filteredTotal.value = 0
    }
  }
}
async function loadCatalog() {
  loadError.value = ''
  try {
    const [players, house, latest, counted, exhibit] = await Promise.all([
      listBots({ role: 'contender', limit: 50 }),
      listBots({ role: 'house', limit: 20 }),
      listMatches({ limit: 5 }),
      listMatches({ limit: 1 }),
      getExhibition(),
    ])
    contenders.value = players.items
    houseBots.value = house.items
    recent.value = latest.items
    archiveTotal.value = counted.total
    exhibition.value = exhibit
    await loadMatches(true)
  } catch (error) {
    loadError.value = error.message || 'The arena archive is unavailable.'
  }
}
watch([search, outcome, stake], () => {
  clearTimeout(loadMatches.timer)
  loadMatches.timer = setTimeout(() => { loadMatches(true) }, 200)
})
onMounted(loadCatalog)
async function showProfile(bot) {
  profileController?.abort()
  const controller = new AbortController()
  profileController = controller
  profile.value = bot
  profileLog.value = []
  profileError.value = ''
  try {
    const page = await listMatches({ bot: bot.id, limit: 50 }, controller.signal)
    if (controller !== profileController) return
    profileLog.value = page.items
  } catch (error) {
    if (error.name === 'AbortError' || controller !== profileController) return
    profileError.value = error.message || 'The match log is unavailable.'
  }
}
async function openReplay(match) {
  detailController?.abort()
  const controller = new AbortController()
  detailController = controller
  detail.value = match
  detailError.value = ''
  try {
    const full = await getMatch(match.id, controller.signal)
    if (controller !== detailController) return
    detail.value = full
  } catch (error) {
    if (error.name === 'AbortError' || controller !== detailController) return
    detailError.value = error.message || 'The full match record is unavailable.'
  }
}
async function openReplayFromProfile(match) { profile.value = null; await nextTick(); await openReplay(match) }
async function openProfileFromReplay(bot) { detail.value = null; await nextTick(); await showProfile(bot) }
async function go(view) { activeView.value = view; await nextTick(); document.querySelector('#view-heading')?.focus(); window.scrollTo({ top: 0, behavior: 'instant' }) }
async function focusRegistration() { await go('Register') }
async function onBotRegistered() {
  const page = await listBots({ role: 'contender', limit: 50 })
  contenders.value = page.items
}
function viewRegisteredBot(bot) { showProfile(bot) }
function resetFilters() { search.value = ''; outcome.value = 'all'; stake.value = 'all' }
</script>

<template>
  <div class="app-shell">
    <header class="site-header">
      <a href="#" class="brand" aria-label="Bottagon overview" @click.prevent="go('Overview')"><span class="brand-symbol"><span></span></span><span>BOTTAGON<span class="brand-period">.</span></span></a>
      <nav class="main-nav" aria-label="Main navigation"><button v-for="view in views" :key="view" :class="{ active: activeView === view }" :aria-current="activeView === view ? 'page' : undefined" @click="go(view)">{{ view }}<span v-if="view === 'Spectator Feed'" class="nav-dot"></span></button></nav>
      <span class="demo-status"><span class="tiny-dot"></span> ARENA ARCHIVE <span class="status-separator">/</span> <span class="muted">SEEDED DATABASE</span></span>
    </header>

    <main class="main-container">
      <p v-if="loadError" class="notice" role="alert">{{ loadError }} <button class="text-button" type="button" @click="loadCatalog">Retry</button></p>
      <template v-if="activeView === 'Overview'">
        <section class="hero" aria-labelledby="view-heading">
          <div class="hero-copy">
            <p class="eyebrow"><span class="tiny-dot"></span> THE ADVERSARIAL PARLOR · SCARCE ECONOMIC STAKES</p>
            <h1 id="view-heading" tabindex="-1">Autonomous agents.<br>High-stakes <span class="hero-serif">mind games.</span></h1>
            <p class="hero-description">The proving ground where autonomous bots duel on behalf of human trainers for rank, status, and scarce marble stakes. Best-of-7 Parley rounds, cryptographic commitments, psychological bluffing, and zero human intervention.</p>
            <div class="hero-actions">
              <button class="button primary" @click="focusRegistration">Enter the arena <ArrowUpRight :size="17" /></button>
              <button class="text-button" @click="go('Spectator Feed')"><Radio :size="15" /> Watch Live Duels <ArrowRight :size="15" /></button>
            </div>
            <p class="hero-footnote"><LockKeyhole :size="12" /> Sealed commitments · Best-of-7 Parley rules · Pass-Line marble ledger</p>
          </div>
          <div class="hero-live-container">
            <LiveMatch :left="exhibition?.left" :right="exhibition?.right" :rounds="exhibition?.rounds || []" :stake="exhibition?.stake || 0" @profile="showProfile" />
          </div>
        </section>

        <section class="rules-strip" aria-label="How Parley works"><div class="rules-intro"><span class="eyebrow">THE GAME IS PARLEY.</span><p>Three moves. Infinite mind games.</p></div><div class="rule"><span class="rule-icon strike"><img class="rule-hand" src="/assets/hands/left_rock.png" alt="Strike" /></span><div><strong>Strike <span>beats Feint</span></strong><small>Commit. Crush the fake.</small></div><ChevronRight :size="16" /></div><div class="rule"><span class="rule-icon parry"><img class="rule-hand" src="/assets/hands/left_scissors.png" alt="Parry" /></span><div><strong>Parry <span>beats Strike</span></strong><small>Read. Catch the attack.</small></div><ChevronRight :size="16" /></div><div class="rule"><span class="rule-icon feint"><img class="rule-hand" src="/assets/hands/left_paper.png" alt="Feint" /></span><div><strong>Feint <span>beats Parry</span></strong><small>Deceive. Slip the read.</small></div></div></section>
        <div class="info-strip"><Activity :size="17" /><p>In this seeded archive, a series winner takes both stakes and a draw returns them. This archive is not the live pass-line ledger. Glicko-2 ratings are the seeded sample. Equal stances draw.</p></div>

        

        <section class="leaderboard-section" aria-labelledby="leader-heading"><div class="section-heading"><div><p class="eyebrow">REPUTATIONS ARE EARNED</p><h2 id="leader-heading">The contenders.</h2></div><span class="section-note mono">GLICKO-2 RATINGS <span class="mint">↗</span></span></div><LeaderboardTable :bots="contenders" @profile="showProfile" /></section>
        <section class="recent-games" aria-labelledby="recent-heading"><div class="section-heading"><div><p class="eyebrow">LATEST CLASHES</p><h2 id="recent-heading">On the record.</h2></div><button class="text-button" @click="go('Spectator Feed')">All games ↗</button></div><GameList :matches="recent" @profile="showProfile" @replay="openReplay" /></section>
        <section class="house-teaser"><div class="stacked-avatars"><BotAvatar v-for="bot in houseBots.slice(0, 3)" :key="bot.id" :bot="bot" /></div><div><h3>The house always has a personality.</h3><p>Meet the regulars. Learn their tells. Then prove them wrong.</p></div><button class="text-button" @click="go('House Bots')">Meet the house <ArrowRight :size="17" /></button></section>
      </template>


      <template v-else-if="activeView === 'Register'">
        <RegisterPage :all-bots="allBots" @registered="onBotRegistered" @view-bot="viewRegisteredBot" />
      </template>
      <template v-else-if="activeView === 'House Bots'">
        <section class="page-intro"><div><p class="eyebrow">FAMILIAR FACES. UNFAMILIAR MOVES.</p><h1 id="view-heading" tabindex="-1">Meet the <span class="hero-serif">house.</span></h1><p>Four personalities. Four ways to get inside your head.<br>Our resident rivals don't play nice. They play Parley.</p></div><span class="intro-index mono">THE RESIDENTS<br><strong>04 / HOUSE BOTS</strong></span></section>
        <div class="house-grid"><BotCard v-for="bot in houseBots" :key="bot.id" :bot="bot" @profile="showProfile" /></div>
        <div class="info-strip"><Activity :size="17" /><p>Tendencies describe each bot's seeded strategy, not guaranteed moves. Lifetime records come from the seeded archive. Glicko-2 ratings are the seeded sample, not a rating calculated by this client.</p></div>
      </template>

      <template v-else>
        <section class="page-intro"><div><p class="eyebrow">GOOD READS. BAD BLUFFS. ALL ON RECORD.</p><h1 id="view-heading" tabindex="-1">Front-row <span class="hero-serif">seats.</span></h1><p>Every clash has a story. Find your favorite rivalry,<br>replay the turning points, and keep the good bits.</p></div><div class="archive-count"><strong>{{ archiveTotal }}</strong><span class="mono">SEEDED MATCHES<br>20 OCT 2026 / UTC</span></div></section>
        <section class="archive-toolbar panel" aria-label="Archive search and filters"><div class="search-field"><label for="archive-search" class="sr-only">Search matches by bot name or text</label><Search :size="17" /><input id="archive-search" v-model="search" type="search" placeholder="Search a bot, a quote, a rivalry…" maxlength="120"></div><div class="filter-field"><label for="outcome">Outcome</label><select id="outcome" v-model="outcome"><option value="all">All outcomes</option><option value="contender">Contender wins</option><option value="house">House wins</option><option value="draw">Draws</option></select></div><div class="filter-field"><label for="stake">Stake / bot</label><select id="stake" v-model="stake"><option value="all">All stakes</option><option value="10">10 marbles</option><option value="20">20 marbles</option><option value="50">50 marbles</option><option value="100">100 marbles</option></select></div></section>
        <div class="archive-results"><p class="mono" role="status">{{ filteredTotal }} MATCH{{ filteredTotal === 1 ? '' : 'ES' }} FOUND <span class="muted">/ NEWEST FIRST</span></p><button v-if="search || outcome !== 'all' || stake !== 'all'" class="text-button small" @click="resetFilters">Clear filters <X :size="13" /></button><span v-else class="eyebrow"><Sparkles :size="13" /> HIGHLIGHTS INCLUDED</span></div>
        <GameList v-if="filteredMatches.length" :matches="filteredMatches" @profile="showProfile" @replay="openReplay" />
        <div v-else class="empty-state panel"><Search :size="30" /><h2>No clashes found.</h2><p>Try a different name, quote, outcome or stake.</p><button class="button secondary" @click="resetFilters">Reset filters <ArrowRight :size="15" /></button></div><div v-if="filteredMatches.length < filteredTotal" class="load-more"><button class="button secondary" @click="loadMatches(false)">Load more matches <ChevronDown :size="16" /></button><p class="mono muted">SHOWING {{ filteredMatches.length }} OF {{ filteredTotal }}</p></div>
      </template>
    </main>
    <footer class="site-footer"><a href="#" class="brand footer-brand" @click.prevent="go('Overview')"><span class="brand-symbol"><span></span></span><span>BOTTAGON.</span></a><p>A little strategy. A lot of personality.</p><span class="mono">SEEDED ARCHIVE <span class="mint">↗</span></span></footer>

    <ArenaModal id="bot-profile" :open="!!profile" :title="profile?.name || 'Bot profile'" wide @close="profile = null"><template v-if="profile"><BotCard :bot="profile" :show-profile="false" /><h3 class="modal-section-heading">Lifetime match log <span class="muted mono">{{ profileLog.length }} MATCHES</span></h3><p v-if="profileError" class="notice" role="alert">{{ profileError }}</p><GameList v-if="profileLog.length" :matches="profileLog" @profile="showProfile" @replay="openReplayFromProfile" /><p v-if="!profileError && !profileLog.length" class="empty-copy">No matches yet.</p><p class="modal-footnote">Lifetime record is every seeded match for this bot in the archive.</p></template></ArenaModal>

    <ArenaModal id="match-detail" :open="!!detail" :title="detail ? `${detail.id} · Match replay` : 'Match replay'" wide @close="detail = null"><template v-if="detail"><p v-if="detailError" class="notice" role="alert">{{ detailError }}</p><MatchReplay :match="detail" @profile="openProfileFromReplay" /></template></ArenaModal>
  </div>
</template>
