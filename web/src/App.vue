<script setup>
import { nextTick, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import SiteHeader from './components/SiteHeader.vue'
import SiteFooter from './components/SiteFooter.vue'
import { ensureCatalog, reloadCatalog, useCatalog } from './catalogStore.js'

const route = useRoute()
const { state } = useCatalog()

onMounted(ensureCatalog)

watch(() => route.fullPath, async (to, from) => {
  if (typeof document === 'undefined') return
  document.title = route.meta.title ? `${route.meta.title} · BOTTAGON` : 'BOTTAGON'
  if (!from || route.hash || to.split('#')[0] === from.split('#')[0]) return
  await nextTick()
  document.querySelector('#view-heading')?.focus({ preventScroll: true })
})
</script>

<template>
  <a class="skip-link" href="#main">Skip to content</a>
  <SiteHeader />
  <main id="main" class="site-main">
    <div v-if="state.error" class="container">
      <p class="load-error" role="alert">{{ state.error }} <button class="btn-text" type="button" @click="reloadCatalog">Try again</button></p>
    </div>
    <RouterView />
  </main>
  <SiteFooter />
</template>

<style scoped>
.skip-link {
  position: absolute;
  left: var(--space-4);
  top: -100px;
  z-index: 30;
  padding: var(--space-2) var(--space-4);
  background: var(--action-bg);
  color: var(--action-fg);
  border-radius: var(--radius-control);
  font-weight: var(--weight-bold);
}
.skip-link:focus { top: var(--space-3); }
.site-main { min-height: 60vh; }
.load-error {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  margin: var(--space-5) 0 0;
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--line-strong);
  border-radius: var(--radius-surface);
  background: var(--surface);
  font-size: var(--step--1);
  color: var(--ink-secondary);
}
</style>
