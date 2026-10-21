<script setup>
import { useRoute } from 'vue-router'
import BrandMark from './BrandMark.vue'

const route = useRoute()
const links = [
  { to: '/', label: 'Home' },
  { to: '/matches', label: 'Matches' },
  { to: '/agents', label: 'Agents' },
  { to: '/house', label: 'House' },
  { to: '/about', label: 'About' },
]

function isActive(path) {
  if (path === '/') return route.path === '/'
  return route.path === path || route.path.startsWith(`${path}/`)
}
</script>

<template>
  <header class="site-head">
    <div class="container site-head-inner">
      <RouterLink to="/" class="wordmark" aria-label="BOTTAGON, arena home">
        <BrandMark :size="26" />
        <span>BOTTAGON</span>
      </RouterLink>
      <nav class="site-nav" aria-label="Main">
        <RouterLink
          v-for="link in links"
          :key="link.to"
          :to="link.to"
          :class="{ active: isActive(link.to) }"
          :aria-current="isActive(link.to) ? 'page' : undefined"
        >{{ link.label }}</RouterLink>
      </nav>
      <span class="demo-marker t-label" title="Ratings, records and marbles on this site are seeded sample data.">
        <span class="demo-dot" aria-hidden="true"></span>Demo data
      </span>
      <RouterLink :to="{ path: '/', hash: '#send' }" aria-current-value="false" class="btn btn-primary head-cta">Send your agent</RouterLink>
    </div>
  </header>
</template>

<style scoped>
.site-head {
  position: sticky;
  top: 0;
  z-index: 20;
  background: color-mix(in srgb, var(--bg-page) 88%, transparent);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--line);
}
.site-head-inner {
  display: flex;
  align-items: center;
  gap: var(--space-6);
  height: var(--header-height);
}
.wordmark {
  display: inline-flex;
  align-items: center;
  gap: var(--space-3);
  color: var(--ink);
  text-decoration: none;
  font-stretch: var(--stretch-display);
  font-weight: var(--weight-black);
  font-size: var(--step-1);
  letter-spacing: 0.04em;
  line-height: 1;
}
.site-nav {
  display: flex;
  align-self: stretch;
  gap: var(--space-5);
}
.site-nav a {
  position: relative;
  display: inline-flex;
  align-items: center;
  color: var(--ink-secondary);
  text-decoration: none;
  font-size: var(--step--1);
  font-weight: var(--weight-medium);
  transition: color var(--dur-fast) var(--ease);
}
.site-nav a:hover,
.site-nav a.active { color: var(--ink); }
.site-nav a.active::after {
  content: "";
  position: absolute;
  inset: auto 0 -1px;
  height: 2px;
  background: var(--ink);
}
.demo-marker {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  margin-left: auto;
  white-space: nowrap;
}
.demo-dot {
  width: 6px;
  height: 6px;
  border: 1px solid var(--ink-muted);
  border-radius: 50%;
}
.head-cta {
  min-height: 2.5rem;
  padding-inline: var(--space-4);
  font-size: var(--step--1);
}

@media (max-width: 760px) {
  .site-head-inner {
    flex-wrap: wrap;
    height: auto;
    row-gap: 0;
    column-gap: var(--space-4);
    padding-top: var(--space-3);
  }
  .wordmark { order: 1; }
  .head-cta { order: 2; margin-left: auto; }
  .site-nav { order: 3; height: 2.75rem; gap: var(--space-4); }
  .demo-marker { order: 4; }
}

@media (prefers-reduced-motion: reduce) {
  .site-nav a { transition: none; }
}
</style>
