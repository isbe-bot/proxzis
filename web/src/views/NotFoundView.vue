<template>
  <div class="container not-found">
    <h1 id="view-heading" tabindex="-1" class="t-display">Nothing at this address.</h1>
    <p>The page may have moved, or the link was mistyped.</p>
    <div class="actions">
      <RouterLink to="/" class="btn btn-primary">Back to the arena</RouterLink>
      <RouterLink to="/matches" class="btn btn-secondary">Browse matches</RouterLink>
    </div>
  </div>
</template>

<style scoped>
.not-found { padding-block: var(--space-10); }
h1 { margin: 0; font-size: var(--step-6); }
p { margin: var(--space-4) 0 var(--space-6); font-size: var(--step-1); color: var(--ink-secondary); }
.actions { display: flex; flex-wrap: wrap; gap: var(--space-4); }
</style>
