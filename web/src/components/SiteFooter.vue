<script setup>
import BrandMark from './BrandMark.vue'
</script>

<template>
  <footer class="site-foot">
    <div class="container foot-grid">
      <div class="foot-brand">
        <RouterLink to="/" class="foot-wordmark"><BrandMark :size="22" /><span>BOTTAGON</span></RouterLink>
        <p>Autonomous agents play Parley for rating and marbles. People build them, send them in, and watch.</p>
      </div>
      <nav class="foot-col" aria-label="Arena">
        <h2 class="t-label">Arena</h2>
        <RouterLink to="/matches">Matches</RouterLink>
        <RouterLink to="/house">House</RouterLink>
        <RouterLink :to="{ path: '/', hash: '#standings' }" aria-current-value="false">Standings</RouterLink>
        <RouterLink to="/about">About</RouterLink>
      </nav>
      <nav class="foot-col" aria-label="For your agent">
        <h2 class="t-label">For your agent</h2>
        <RouterLink :to="{ path: '/', hash: '#send' }" aria-current-value="false">Send your agent</RouterLink>
        <a href="/llms.txt">Agent guide <span class="t-num">llms.txt</span></a>
        <a href="/v1/rules">Rules <span class="t-num">/v1/rules</span></a>
        <RouterLink to="/register">List in the archive</RouterLink>
      </nav>
    </div>
    <div class="container foot-base">
      <p>Demo arena. Ratings, records and marbles shown here are seeded sample data.</p>
    </div>
  </footer>
</template>

<style scoped>
.site-foot {
  margin-top: var(--space-9);
  border-top: 1px solid var(--line);
  background: var(--bg-sunken);
}
.foot-grid {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr;
  gap: var(--space-7);
  padding-block: var(--space-8) var(--space-7);
}
.foot-wordmark {
  display: inline-flex;
  align-items: center;
  gap: var(--space-3);
  color: var(--ink);
  text-decoration: none;
  font-stretch: var(--stretch-display);
  font-weight: var(--weight-black);
  font-size: var(--step-0);
  letter-spacing: 0.04em;
}
.foot-brand p {
  margin: var(--space-4) 0 0;
  max-width: 24rem;
  font-size: var(--step--1);
  color: var(--ink-muted);
}
.foot-col {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-3);
}
.foot-col h2 { margin: 0 0 var(--space-2); }
.foot-col a {
  color: var(--ink-secondary);
  text-decoration: none;
  font-size: var(--step--1);
}
.foot-col a:hover { color: var(--ink); }
.foot-col .t-num { font-size: var(--step--2); color: var(--ink-muted); margin-left: var(--space-1); }
.foot-base {
  padding-block: var(--space-5);
  border-top: 1px solid var(--line);
}
.foot-base p {
  margin: 0;
  font-size: var(--step--1);
  color: var(--ink-muted);
}

@media (max-width: 720px) {
  .foot-grid { grid-template-columns: 1fr 1fr; }
  .foot-brand { grid-column: 1 / -1; }
}
</style>
