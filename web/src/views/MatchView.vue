<script setup>
import { ref, watch } from 'vue'
import { ArrowLeft } from 'lucide-vue-next'
import MatchReplay from '../components/MatchReplay.vue'
import { getMatch } from '../api/catalog.js'
import { dateLabel } from '../utils/presentation.js'

const props = defineProps({ id: { type: String, required: true } })
const detail = ref(null)
const detailError = ref('')
let controller = null

watch(() => props.id, async id => {
  controller?.abort()
  const current = new AbortController()
  controller = current
  detail.value = null
  detailError.value = ''
  try {
    const match = await getMatch(id, current.signal)
    if (current !== controller) return
    detail.value = match
    document.title = `${match.a.name} vs ${match.b.name} · BOTTAGON`
  } catch (error) {
    if (error.name === 'AbortError' || current !== controller) return
    detailError.value = error.status === 404 ? `There is no match ${id} in the archive.` : (error.message || 'The match record is unavailable.')
  }
}, { immediate: true })
</script>

<template>
  <div class="page container page-reading">
    <RouterLink to="/matches" class="back-link"><ArrowLeft :size="16" aria-hidden="true" /> All matches</RouterLink>
    <header class="page-head">
      <div class="page-head-copy">
        <p class="page-kicker t-label">{{ id }}<template v-if="detail"> · {{ dateLabel(detail.date) }}</template></p>
        <h1 id="view-heading" class="page-title t-display" tabindex="-1">
          <template v-if="detail">{{ detail.a.name }} <span class="versus">vs</span> {{ detail.b.name }}</template>
          <template v-else>Match {{ id }}</template>
        </h1>
      </div>
    </header>
    <p v-if="detailError" class="notice" role="alert">{{ detailError }} <RouterLink to="/matches" class="btn-text">Browse the archive</RouterLink></p>
    <p v-else-if="!detail" class="status-line" role="status">Loading the match record.</p>
    <MatchReplay v-if="detail" :match="detail" />
  </div>
</template>

<style scoped>
.page-reading { max-width: calc(60rem + 2 * var(--gutter)); }
.page-reading .page-title { font-size: var(--step-5); }
.versus { color: var(--ink-muted); font-weight: var(--weight-medium); }
</style>
