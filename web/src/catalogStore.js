import { computed, reactive } from 'vue'
import { getExhibition, listBots, listMatches } from './api/catalog.js'

const state = reactive({
  contenders: [],
  houseBots: [],
  recent: [],
  exhibition: null,
  archiveTotal: 0,
  error: '',
  loaded: false,
})

let pending = null

async function load() {
  state.error = ''
  try {
    const [players, house, latest, exhibit] = await Promise.all([
      listBots({ role: 'contender', limit: 50 }),
      listBots({ role: 'house', limit: 20 }),
      listMatches({ limit: 5 }),
      getExhibition(),
    ])
    state.contenders = players.items
    state.houseBots = house.items
    state.recent = latest.items
    state.archiveTotal = latest.total
    state.exhibition = exhibit
    state.loaded = true
  } catch (error) {
    state.error = error.message || 'The arena archive is unavailable.'
  } finally {
    pending = null
  }
}

export function ensureCatalog() {
  if (state.loaded) return Promise.resolve()
  pending ??= load()
  return pending
}

export function reloadCatalog() {
  state.loaded = false
  return ensureCatalog()
}

export async function refreshContenders() {
  const page = await listBots({ role: 'contender', limit: 50 })
  state.contenders = page.items
}

export function useCatalog() {
  return {
    state,
    allBots: computed(() => [...state.contenders, ...state.houseBots]),
  }
}
