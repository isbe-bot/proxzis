<script setup>
import { computed } from 'vue'
import { stanceName } from '../utils/presentation.js'
import { resolveClash } from '../utils/arena.js'

const props = defineProps({
  match: { type: Object, required: true },
})

const log = computed(() => {
  const match = props.match
  const items = [{ id: 'begin', kind: 'system', text: 'Game begins.' }]
  for (let i = 0; i < match.rounds.length; i++) {
    const played = match.rounds[i]
    if (played.quoteA) items.push({ id: `a-${i}`, kind: 'a', name: match.a.name, text: played.quoteA })
    if (played.quoteB) items.push({ id: `b-${i}`, kind: 'b', name: match.b.name, text: played.quoteB })
    const side = resolveClash(played.a, played.b)
    const n = i + 1
    if (side === 'draw') items.push({ id: `r-${i}`, kind: 'system', text: `Round ${n} is a draw.` })
    else if (side === 'a') items.push({ id: `r-${i}`, kind: 'system', text: `${match.a.name} wins round ${n} with ${stanceName(played.a)}.` })
    else items.push({ id: `r-${i}`, kind: 'system', text: `${match.b.name} wins round ${n} with ${stanceName(played.b)}.` })
  }
  const closing = match.winner === 'draw' ? 'Game complete. Draw.' : `Game complete. ${match[match.winner].name} wins.`
  items.push({ id: 'done', kind: 'system', text: closing })
  return items
})
</script>

<template>
  <div class="transcript" tabindex="0">
    <ol class="chat-log" aria-label="Match chat">
      <li v-for="item in log" :key="item.id" class="chat-line" :class="`is-${item.kind}`">
        <span v-if="item.kind === 'system'" class="chat-system">{{ item.text }}</span>
        <template v-else>
          <span class="chat-name">{{ item.name }}</span>
          <span class="chat-text">{{ item.text }}</span>
        </template>
      </li>
    </ol>
  </div>
</template>

<style scoped>
.transcript {
  min-height: 0;
  overflow: auto;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
  border: 1px solid var(--line);
  border-radius: var(--radius-control);
  background: var(--bg-sunken);
  padding: var(--space-5) var(--space-4) var(--space-5) var(--space-5);
}
.transcript:focus-visible { outline: 2px solid var(--focus-ring); outline-offset: -2px; }
.transcript::-webkit-scrollbar { width: 6px; }
.transcript::-webkit-scrollbar-track { background: transparent; }
.transcript::-webkit-scrollbar-thumb { background: var(--line-strong); border-radius: 3px; }
.transcript::-webkit-scrollbar-button { display: none; }
@supports not selector(::-webkit-scrollbar) {
  .transcript { scrollbar-width: thin; scrollbar-color: var(--line-strong) transparent; }
}

.chat-log {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin: 0;
  padding: 0;
  list-style: none;
}
.chat-line { margin: 0; min-width: 0; }
.chat-line.is-system + .chat-line:not(.is-system),
.chat-line:not(.is-system) + .chat-line.is-system { margin-top: var(--space-1); }
.chat-system {
  display: block;
  font-size: var(--step--2);
  line-height: 1.4;
  text-align: center;
  color: var(--ink-muted);
}
.chat-name {
  display: block;
  margin-bottom: 2px;
  font-size: var(--step--2);
  font-weight: var(--weight-bold);
  line-height: 1.3;
}
.is-a .chat-name { color: var(--cobalt); }
.is-b .chat-name { color: var(--crimson); }
.chat-text {
  display: block;
  font-size: var(--step--1);
  line-height: 1.4;
  color: var(--ink);
  overflow-wrap: anywhere;
}
</style>
