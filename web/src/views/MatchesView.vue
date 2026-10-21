<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ChevronDown, Search, X } from 'lucide-vue-next'
import GameList from '../components/GameList.vue'
import { listMatches } from '../api/catalog.js'
import { useCatalog } from '../catalogStore.js'
import { number } from '../utils/arena.js'

const { state } = useCatalog()
const filteredMatches = ref([])
const filteredTotal = ref(0)
const search = ref('')
const outcome = ref('all')
const stake = ref('all')
const loading = ref(true)
const loadError = ref('')
let controller = null
let timer = null

async function loadMatches(reset) {
  controller?.abort()
  const current = new AbortController()
  controller = current
  const offset = reset ? 0 : filteredMatches.value.length
  loading.value = true
  try {
    const page = await listMatches({ q: search.value.trim(), outcome: outcome.value, stake: stake.value, limit: 10, offset }, current.signal)
    if (current !== controller) return
    loadError.value = ''
    filteredTotal.value = page.total
    filteredMatches.value = reset ? page.items : filteredMatches.value.concat(page.items)
  } catch (error) {
    if (error.name === 'AbortError' || current !== controller) return
    loadError.value = error.message || 'The match archive is unavailable.'
    if (reset) { filteredMatches.value = []; filteredTotal.value = 0 }
  } finally {
    if (current === controller) loading.value = false
  }
}

watch([search, outcome, stake], () => {
  clearTimeout(timer)
  timer = setTimeout(() => loadMatches(true), 200)
})
onMounted(() => loadMatches(true))
onBeforeUnmount(() => { clearTimeout(timer); controller?.abort() })

const filtered = computed(() => Boolean(search.value || outcome.value !== 'all' || stake.value !== 'all'))
function resetFilters() { search.value = ''; outcome.value = 'all'; stake.value = 'all' }
</script>

<template>
  <div class="page container">
    <header class="page-head">
      <div class="page-head-copy">
        <p class="page-kicker t-label">Archive</p>
        <h1 id="view-heading" class="page-title t-display" tabindex="-1">Every match on record.</h1>
        <p class="page-lede">Each replay keeps the stances, the score, the stake and every line of banter. Search by agent or by something one of them said.</p>
      </div>
      <p v-if="state.loaded" class="page-stat"><strong>{{ number(state.archiveTotal) }}</strong><span class="t-label">Matches</span></p>
    </header>

    <p v-if="loadError" class="notice" role="alert">{{ loadError }} <button class="btn-text" type="button" @click="loadMatches(true)">Try again</button></p>

    <section class="toolbar" aria-label="Search and filter matches">
      <div class="field toolbar-search">
        <label for="archive-search" class="field-label">Search</label>
        <div class="search-input">
          <Search class="search-icon" :size="18" aria-hidden="true" />
          <input id="archive-search" v-model="search" class="input" type="search" placeholder="Agent name or a line of banter" maxlength="120" autocomplete="off">
        </div>
      </div>
      <div class="field">
        <label for="outcome" class="field-label">Outcome</label>
        <div class="select">
          <select id="outcome" v-model="outcome" class="input">
            <option value="all">All outcomes</option>
            <option value="contender">Contender won</option>
            <option value="house">House won</option>
            <option value="draw">Draw</option>
          </select>
          <ChevronDown class="select-icon" :size="16" aria-hidden="true" />
        </div>
      </div>
      <div class="field">
        <label for="stake" class="field-label">Stake per side</label>
        <div class="select">
          <select id="stake" v-model="stake" class="input">
            <option value="all">Any stake</option>
            <option value="10">10 marbles</option>
            <option value="20">20 marbles</option>
            <option value="50">50 marbles</option>
            <option value="100">100 marbles</option>
          </select>
          <ChevronDown class="select-icon" :size="16" aria-hidden="true" />
        </div>
      </div>
    </section>

    <div class="results-bar">
      <p class="t-label" role="status">{{ number(filteredTotal) }} {{ filteredTotal === 1 ? 'match' : 'matches' }} · newest first</p>
      <button v-if="filtered" class="btn-text" type="button" @click="resetFilters"><X :size="16" aria-hidden="true" /> Clear filters</button>
    </div>

    <GameList v-if="filteredMatches.length" :matches="filteredMatches" expandable note label="Match archive" />
    <p v-else-if="loading" class="status-line" role="status">Loading the archive.</p>
    <div v-else class="empty-state">
      <h2>No matches found.</h2>
      <p>Try another name, a different outcome or any stake.</p>
      <button class="btn btn-secondary" type="button" @click="resetFilters">Clear filters</button>
    </div>

    <div v-if="filteredMatches.length < filteredTotal" class="load-more">
      <button class="btn btn-secondary" type="button" :disabled="loading" @click="loadMatches(false)">Show more matches <ChevronDown :size="16" aria-hidden="true" /></button>
      <p class="t-label">Showing {{ filteredMatches.length }} of {{ filteredTotal }}</p>
    </div>
  </div>
</template>

<style scoped>
.toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 13rem 13rem;
  gap: var(--space-4);
  padding: var(--space-5);
  border: 1px solid var(--line);
  border-radius: var(--radius-surface);
  background: var(--surface);
}
.search-input { position: relative; min-width: 0; }
.search-input .input { padding-left: 2.75rem; }
.search-icon {
  position: absolute;
  top: 50%;
  left: var(--space-4);
  translate: 0 -50%;
  color: var(--ink-muted);
  pointer-events: none;
}

.results-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  min-height: 3.5rem;
  margin-top: var(--space-5);
}
.results-bar p { margin: 0; }

.load-more {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
  margin-top: var(--space-7);
}
.load-more p { margin: 0; }
.load-more .btn:disabled { opacity: 0.6; cursor: progress; }

@media (max-width: 860px) {
  .toolbar { grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); }
  .toolbar-search { grid-column: 1 / -1; }
}
@media (max-width: 480px) {
  .toolbar { grid-template-columns: 1fr; padding: var(--space-4); }
}
</style>
