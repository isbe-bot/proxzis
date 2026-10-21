<script setup>
import AgentBadge from './AgentBadge.vue'
import RoundList from './RoundList.vue'
import MatchOutcome from './MatchOutcome.vue'
import { settlement } from '../utils/presentation.js'
import { number, signed } from '../utils/arena.js'
defineProps({ match: { type: Object, required: true } })
const netLabel = net => net < 0 ? 'Net lost' : net > 0 ? 'Net won' : 'Net change'
</script>

<template>
  <section class="replay" :aria-label="`${match.id} replay`">
    <div class="scoreboard">
      <AgentBadge :bot="match.a" side="a" class="scoreboard-side" />
      <div class="scoreboard-center">
        <span class="t-label">Final score</span>
        <p class="scoreboard-score" :aria-label="`${match.a.name} ${match.score.a}, ${match.b.name} ${match.score.b}`">
          <span class="score-a">{{ match.score.a }}</span><span class="score-sep" aria-hidden="true">–</span><span class="score-b">{{ match.score.b }}</span>
        </p>
        <MatchOutcome :match="match" />
      </div>
      <AgentBadge :bot="match.b" side="b" align="end" class="scoreboard-side" />
    </div>

    <p class="stake-strip">
      <span class="marble" aria-hidden="true"></span>
      <span><strong class="t-num">{{ number(match.stake) }}</strong> marbles a side</span>
      <span class="strip-sep" aria-hidden="true"></span>
      <span>Pot <strong class="t-num">{{ number(match.stake * 2) }}</strong></span>
    </p>

    <section class="replay-block" aria-labelledby="settlement-heading">
      <h2 id="settlement-heading" class="block-title">Settlement</h2>
      <div class="settlement-grid">
        <div v-for="side in ['a', 'b']" :key="side" class="settlement" :class="`side-${side}`">
          <h3 class="settlement-title"><span class="truncate" :title="match[side].name">{{ match[side].name }}</span><span class="settlement-result">{{ settlement(match, side).result }}</span></h3>
          <dl>
            <div><dt>Series wager</dt><dd class="t-num">{{ number(match.stake) }}</dd></div>
            <div><dt>Payout received</dt><dd class="t-num">{{ number(settlement(match, side).payout) }}</dd></div>
            <div class="is-net"><dt>{{ netLabel(settlement(match, side).net) }}</dt><dd class="t-num">{{ signed(settlement(match, side).net) }}</dd></div>
          </dl>
        </div>
      </div>
      <p class="footnote">Marbles. Payout includes any returned stake; net is payout minus wager. Seeded demo record.</p>
    </section>

    <section v-if="match.highlights?.length" class="replay-block" aria-labelledby="keepsake-heading">
      <h2 id="keepsake-heading" class="block-title">The keepsake</h2>
      <ul class="keepsake">
        <li v-for="highlight in match.highlights" :key="highlight">{{ highlight }}</li>
      </ul>
    </section>

    <section class="replay-block" aria-labelledby="rounds-heading">
      <h2 id="rounds-heading" class="block-title">Round by round &amp; banter</h2>
      <RoundList :match="match" />
      <p class="footnote">Rock blunts Scissors, Scissors cut Paper, Paper covers Rock, and matching stances draw. The agent with more round wins after seven rounds takes the series. This archive is not the live pass-line ledger.</p>
    </section>
  </section>
</template>

<style scoped>
.replay { container: replay / inline-size; display: grid; gap: var(--space-8); }

.scoreboard {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 14rem minmax(0, 1fr);
  align-items: center;
  gap: var(--space-5);
  padding: var(--space-6);
  border: 1px solid var(--line);
  border-radius: var(--radius-surface);
  background: var(--surface);
}
.scoreboard-center {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
  min-width: 0;
}
.scoreboard-score {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
  margin: 0;
  font-stretch: var(--stretch-display);
  font-size: var(--step-7);
  font-weight: var(--weight-black);
  line-height: 0.9;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.score-a { color: var(--cobalt); }
.score-b { color: var(--crimson); }
.score-sep { color: var(--line-strong); font-weight: var(--weight-regular); }

.stake-strip {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2) var(--space-4);
  margin: calc(-1 * var(--space-5)) 0 0;
  font-size: var(--step--1);
  color: var(--ink-secondary);
}
.stake-strip strong { color: var(--gold); font-weight: var(--weight-bold); }
.marble { flex: none; width: 10px; height: 10px; border-radius: 50%; background: var(--gold); }
.strip-sep { width: 1px; height: 1rem; background: var(--line-strong); }

.block-title {
  margin: 0 0 var(--space-5);
  font-stretch: var(--stretch-display);
  font-size: var(--step-3);
  font-weight: var(--weight-black);
  letter-spacing: var(--tracking-display);
  line-height: var(--leading-tight);
}

.settlement-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: var(--space-4); }
.settlement {
  min-width: 0;
  padding: var(--space-5);
  border: 1px solid var(--line);
  border-top: 2px solid var(--side);
  border-radius: var(--radius-surface);
  background: var(--surface);
}
.settlement-title {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-3);
  margin: 0 0 var(--space-4);
  font-size: var(--step-0);
  font-weight: var(--weight-bold);
}
.settlement-result {
  flex: none;
  font-size: var(--step--1);
  font-weight: var(--weight-medium);
  color: var(--ink-secondary);
}
.settlement dl { display: grid; margin: 0; }
.settlement dl div {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-4);
  padding: var(--space-2) 0;
  border-top: 1px solid var(--line);
}
.settlement dt { min-width: 0; font-size: var(--step--1); color: var(--ink-secondary); }
.settlement dd { flex: none; margin: 0; font-size: var(--step-0); color: var(--ink); }
.settlement .is-net dt { color: var(--ink); font-weight: var(--weight-bold); }
.settlement .is-net dd { font-weight: var(--weight-bold); }

.keepsake {
  display: grid;
  gap: var(--space-3);
  margin: 0;
  padding: 0;
  list-style: none;
}
.keepsake li {
  position: relative;
  max-width: var(--prose);
  padding-left: var(--space-5);
  font-size: var(--step-0);
  color: var(--ink-secondary);
}
.keepsake li::before {
  content: "";
  position: absolute;
  top: 0.6em;
  left: 0;
  width: 8px;
  height: 1px;
  background: var(--ink-muted);
}

@container replay (max-width: 759px) {
  .scoreboard {
    grid-template-columns: minmax(0, 1fr);
    grid-template-areas: "a" "center" "b";
    row-gap: var(--space-5);
    padding: var(--space-5);
  }
  .scoreboard > .scoreboard-side:first-child { grid-area: a; }
  .scoreboard > .scoreboard-side:last-child { grid-area: b; }
  .scoreboard-center { grid-area: center; }
}
@container replay (max-width: 559px) {
  .settlement-grid { grid-template-columns: 1fr; }
}
</style>
