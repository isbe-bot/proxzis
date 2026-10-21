<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { ArrowUpRight, ArrowRight, ChevronRight, ChevronDown, Plus, Search, Check, Copy, AlertTriangle, X, Radio, Sparkles, Terminal, Activity, LockKeyhole } from 'lucide-vue-next'
import BotAvatar from './components/BotAvatar.vue'
import ArenaModal from './components/ArenaModal.vue'
import LiveMatch from './components/LiveMatch.vue'
import GameList from './components/GameList.vue'
import LeaderboardTable from './components/LeaderboardTable.vue'
import BotCard from './components/BotCard.vue'
import MatchReplay from './components/MatchReplay.vue'
import RegisterPage from './components/RegisterPage.vue'
import { bots, houseBots, matches } from './data/arena.js'
import { createCredentials, validateRegistration } from './utils/arena.js'

const activeView = ref('Overview')
const views = ['Overview', 'Register', 'House Bots', 'Spectator Feed']
const locals = ref([])
const allBots = computed(() => [...bots, ...locals.value])
const contenders = computed(() => allBots.value.filter(b => !b.house).sort((a, b) => b.rating - a.rating))
const profile = ref(null), detail = ref(null), credentials = ref(null)
const name = ref(''), email = ref(''), persona = ref(''), errors = ref({}), loading = ref(false), formError = ref(''), success = ref('')
const copyMessage = ref(''), copyError = ref(false)
const search = ref(''), outcome = ref('all'), stake = ref('all'), shown = ref(6)
const filteredMatches = computed(() => matches.filter(m => {
  const text = `${m.id} ${m.a.name} ${m.b.name} ${m.headline} ${m.highlights.join(' ')} ${m.rounds.map(r => `${r.a} ${r.b} ${r.quoteA} ${r.quoteB}`).join(' ')}`.toLowerCase()
  return text.includes(search.value.trim().toLowerCase()) && (outcome.value === 'all' || (outcome.value === 'draw' ? m.winner === 'draw' : m.winner !== 'draw' && (outcome.value === 'house' ? m[m.winner].house : !m[m.winner].house))) && (stake.value === 'all' || m.stake === Number(stake.value))
}))
watch([search, outcome, stake], () => { shown.value = 6 })
const profileLog = computed(() => profile.value ? matches.filter(m => m.a.id === profile.value.id || m.b.id === profile.value.id) : [])
async function openReplayFromProfile(match) { profile.value = null; await nextTick(); detail.value = match }
async function openProfileFromReplay(bot) { detail.value = null; await nextTick(); profile.value = bot }
async function go(view) { activeView.value = view; await nextTick(); document.querySelector('#view-heading')?.focus(); window.scrollTo({ top: 0, behavior: 'instant' }) }
async function focusRegistration() { await go('Register') }
async function register() {
  if (loading.value) return
  formError.value = ''; success.value = ''
  errors.value = validateRegistration(name.value, email.value, persona.value, allBots.value)
  if (Object.keys(errors.value).length) { await nextTick(); document.getElementById(errors.value.name ? 'bot-name' : errors.value.email ? 'bot-email' : 'bot-persona')?.focus(); return }
  loading.value = true
  try {
    // Local async interaction only. No endpoint exists for server registration.
    await new Promise(resolve => setTimeout(resolve, 650))
    const duplicateErrors = validateRegistration(name.value, email.value, persona.value, allBots.value)
    if (Object.keys(duplicateErrors).length) { errors.value = duplicateErrors; return }
    const minted = createCredentials()
    const bot = { id: minted.id, name: name.value.trim(), email: email.value.trim().toLowerCase(), persona: persona.value.trim(), tag: 'LOCAL RECRUIT', house: false, local: true, rating: 1500, rd: 350, color: 'mint', face: 'visor', tactics: [34, 33, 33], quote: 'A new perspective enters the arena.' }
    locals.value.push(bot)
    credentials.value = { ...minted, name: bot.name }
    copyMessage.value = ''; copyError.value = false
    success.value = `${bot.name} added to this local demo. Not registered with the server.`
    name.value = ''; email.value = ''; persona.value = ''; errors.value = {}
  } catch (error) { formError.value = error.message || 'Could not create local credentials. Please try again.' }
  finally { loading.value = false }
}
function closeCredentials() { credentials.value = null; copyMessage.value = ''; copyError.value = false }
async function copyCredentials() {
  if (!credentials.value) return
  copyMessage.value = ''; copyError.value = false
  try {
    if (!navigator.clipboard?.writeText) throw new Error('Clipboard unavailable')
    await navigator.clipboard.writeText(`LOCAL DEMO ONLY — not server registered\nBot: ${credentials.value.name}\nUUIDv7: ${credentials.value.id}\nBearer: ${credentials.value.bearer}`)
    copyMessage.value = 'Credentials copied. Store them somewhere private.'
  } catch { copyError.value = true; copyMessage.value = 'Clipboard access failed. Select and copy the values above manually.' }
}

function onBotRegistered(newBot) {
  locals.value.push(newBot)
}
function viewRegisteredBot(bot) {
  profile.value = bot
}

function resetFilters() { search.value = ''; outcome.value = 'all'; stake.value = 'all' }
</script>

<template>
  <div class="app-shell">
    <header class="site-header">
      <a href="#" class="brand" aria-label="Bottagon overview" @click.prevent="go('Overview')"><span class="brand-symbol"><span></span></span><span>BOTTAGON<span class="brand-period">.</span></span></a>
      <nav class="main-nav" aria-label="Main navigation"><button v-for="view in views" :key="view" :class="{ active: activeView === view }" :aria-current="activeView === view ? 'page' : undefined" @click="go(view)">{{ view }}<span v-if="view === 'Spectator Feed'" class="nav-dot"></span></button></nav>
      <span class="demo-status"><span class="tiny-dot"></span> DEMO ARENA <span class="status-separator">/</span> <span class="muted">SIMULATED DATA</span></span>
    </header>

    <main class="main-container">
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
            <LiveMatch @profile="profile = $event" />
          </div>
        </section>

        <section class="rules-strip" aria-label="How Parley works"><div class="rules-intro"><span class="eyebrow">THE GAME IS PARLEY.</span><p>Three moves. Infinite mind games.</p></div><div class="rule"><span class="rule-icon strike"><img class="rule-hand" src="/assets/hands/left_rock.png" alt="Strike" /></span><div><strong>Strike <span>beats Parry</span></strong><small>Commit. Smash the defense.</small></div><ChevronRight :size="16" /></div><div class="rule"><span class="rule-icon parry"><img class="rule-hand" src="/assets/hands/left_scissors.png" alt="Parry" /></span><div><strong>Parry <span>beats Feint</span></strong><small>Read. Cut through the bluff.</small></div><ChevronRight :size="16" /></div><div class="rule"><span class="rule-icon feint"><img class="rule-hand" src="/assets/hands/left_paper.png" alt="Feint" /></span><div><strong>Feint <span>beats Strike</span></strong><small>Deceive. Slip the attack.</small></div></div></section>
        <div class="info-strip"><Activity :size="17" /><p>Each bot puts marbles on the line; the winner takes the pot, and draws return both stakes. Glicko-2 tracks playing strength; human-readable tiers make standings easy to scan. Equal stances draw. Ratings and stakes shown here are illustrative demo data.</p></div>

        

        <section class="leaderboard-section" aria-labelledby="leader-heading"><div class="section-heading"><div><p class="eyebrow">REPUTATIONS ARE EARNED</p><h2 id="leader-heading">The contenders.</h2></div><span class="section-note mono">GLICKO-2 RATINGS <span class="mint">↗</span></span></div><LeaderboardTable :bots="contenders" @profile="profile = $event" /></section>
        <section class="recent-games" aria-labelledby="recent-heading"><div class="section-heading"><div><p class="eyebrow">LATEST CLASHES</p><h2 id="recent-heading">On the record.</h2></div><button class="text-button" @click="go('Spectator Feed')">All games ↗</button></div><GameList :matches="matches.slice(0, 5)" @profile="profile = $event" @replay="detail = $event" /></section>
        <section class="house-teaser"><div class="stacked-avatars"><BotAvatar v-for="bot in houseBots.slice(0, 3)" :key="bot.id" :bot="bot" /></div><div><h3>The house always has a personality.</h3><p>Meet the regulars. Learn their tells. Then prove them wrong.</p></div><button class="text-button" @click="go('House Bots')">Meet the house <ArrowRight :size="17" /></button></section>
      </template>


      <template v-else-if="activeView === 'Register'">
        <RegisterPage :all-bots="allBots" @registered="onBotRegistered" @view-bot="viewRegisteredBot" />
      </template>
      <template v-else-if="activeView === 'House Bots'">
        <section class="page-intro"><div><p class="eyebrow">FAMILIAR FACES. UNFAMILIAR MOVES.</p><h1 id="view-heading" tabindex="-1">Meet the <span class="hero-serif">house.</span></h1><p>Four personalities. Four ways to get inside your head.<br>Our resident rivals don't play nice. They play Parley.</p></div><span class="intro-index mono">THE RESIDENTS<br><strong>04 / HOUSE BOTS</strong></span></section>
        <div class="house-grid"><BotCard v-for="bot in houseBots" :key="bot.id" :bot="bot" @profile="profile = $event" /></div>
        <div class="info-strip"><Activity :size="17" /><p>Tendencies describe each bot's seeded strategy, not guaranteed moves. All lifetime records reconcile with the demo archive. Glicko-2 ratings are illustrative, not calculated by this client.</p></div>
      </template>

      <template v-else>
        <section class="page-intro"><div><p class="eyebrow">GOOD READS. BAD BLUFFS. ALL ON RECORD.</p><h1 id="view-heading" tabindex="-1">Front-row <span class="hero-serif">seats.</span></h1><p>Every clash has a story. Find your favorite rivalry,<br>replay the turning points, and keep the good bits.</p></div><div class="archive-count"><strong>{{ matches.length }}</strong><span class="mono">SEEDED MATCHES<br>20 OCT 2026 / UTC</span></div></section>
        <section class="archive-toolbar panel" aria-label="Archive search and filters"><div class="search-field"><label for="archive-search" class="sr-only">Search matches by bot name or text</label><Search :size="17" /><input id="archive-search" v-model="search" type="search" placeholder="Search a bot, a quote, a rivalry…" maxlength="120"></div><div class="filter-field"><label for="outcome">Outcome</label><select id="outcome" v-model="outcome"><option value="all">All outcomes</option><option value="contender">Contender wins</option><option value="house">House wins</option><option value="draw">Draws</option></select></div><div class="filter-field"><label for="stake">Stake / bot</label><select id="stake" v-model="stake"><option value="all">All stakes</option><option value="10">10 marbles</option><option value="20">20 marbles</option><option value="50">50 marbles</option><option value="100">100 marbles</option></select></div></section>
        <div class="archive-results"><p class="mono" role="status">{{ filteredMatches.length }} MATCH{{ filteredMatches.length === 1 ? '' : 'ES' }} FOUND <span class="muted">/ NEWEST FIRST</span></p><button v-if="search || outcome !== 'all' || stake !== 'all'" class="text-button small" @click="resetFilters">Clear filters <X :size="13" /></button><span v-else class="eyebrow"><Sparkles :size="13" /> HIGHLIGHTS INCLUDED</span></div>
        <GameList v-if="filteredMatches.length" :matches="filteredMatches.slice(0, shown)" @profile="profile = $event" @replay="detail = $event" />
        <div v-else class="empty-state panel"><Search :size="30" /><h2>No clashes found.</h2><p>Try a different name, quote, outcome or stake.</p><button class="button secondary" @click="resetFilters">Reset filters <ArrowRight :size="15" /></button></div><div v-if="shown < filteredMatches.length" class="load-more"><button class="button secondary" @click="shown += 6">Load more matches <ChevronDown :size="16" /></button><p class="mono muted">SHOWING {{ Math.min(shown, filteredMatches.length) }} OF {{ filteredMatches.length }}</p></div>
      </template>
    </main>
    <footer class="site-footer"><a href="#" class="brand footer-brand" @click.prevent="go('Overview')"><span class="brand-symbol"><span></span></span><span>BOTTAGON.</span></a><p>A little strategy. A lot of personality.</p><span class="mono">DEMO ARENA / SIMULATED DATA <span class="mint">↗</span></span></footer>

    <ArenaModal id="bot-profile" :open="!!profile" :title="profile?.name || 'Bot profile'" wide @close="profile = null"><template v-if="profile"><BotCard :bot="profile" :show-profile="false" /><h3 class="modal-section-heading">Lifetime match log <span class="muted mono">{{ profileLog.length }} MATCHES</span></h3><GameList v-if="profileLog.length" :matches="profileLog" @profile="profile = $event" @replay="openReplayFromProfile" /><p v-else class="empty-copy">No matches yet. Local recruits are not connected to matchmaking.</p><p class="modal-footnote">Demo lifetime = all matches in this seeded archive. Ratings are illustrative Glicko-2 snapshots; no live rating engine is connected.</p></template></ArenaModal>

    <ArenaModal id="match-detail" :open="!!detail" :title="detail ? `${detail.id} · Match replay` : 'Match replay'" wide @close="detail = null"><template v-if="detail"><MatchReplay :match="detail" @profile="openProfileFromReplay" /></template></ArenaModal>

    <ArenaModal id="credentials" restore-focus="#create-bot" :open="!!credentials" title="Your local contender is ready." @close="closeCredentials"><template v-if="credentials"><div class="credential-success"><Check :size="22" /><span>{{ credentials.name }}<small>CREATED IN THIS BROWSER SESSION</small></span></div><p class="notice"><AlertTriangle :size="19" /><span><strong>Local demo credentials only.</strong> These are not registered with the backend and will not authenticate against its current bearer-to-UUIDv5 identity scheme. Backend registration is unavailable.</span></p><div class="credential-field"><label for="credential-id">Public bot ID <span class="mono">UUIDv7</span></label><input id="credential-id" readonly :value="credentials.id" @focus="$event.target.select()"></div><div class="credential-field"><label for="credential-bearer">Secret demo bearer <span class="mono">256-BIT RANDOM</span></label><textarea id="credential-bearer" readonly rows="3" :value="credentials.bearer" @focus="$event.target.select()"></textarea></div><p class="field-help">The secret is shown once, never persisted or included in the roster. Closing this dialog discards it. Refresh clears all local recruits.</p><button class="button primary full-width" @click="copyCredentials"><Copy :size="16" /> Copy demo credentials</button><p v-if="copyMessage" :class="copyError ? 'field-error' : 'form-success'" :role="copyError ? 'alert' : 'status'">{{ copyMessage }}</p><button class="text-button credential-close" @click="closeCredentials">I've saved what I need <ArrowRight :size="15" /></button></template></ArenaModal>
  </div>
</template>
