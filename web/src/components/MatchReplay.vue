<script setup>
import UserBadge from './UserBadge.vue'
import RoundList from './RoundList.vue'
import MatchOutcome from './MatchOutcome.vue'
import { dateLabel, settlement } from '../utils/presentation.js'
import { number, signed } from '../utils/arena.js'
defineProps({ match: { type: Object, required: true } })
defineEmits(['profile'])
</script>
<template>
  <section class="match-replay" :aria-label="`${match.id} replay`">
    <div class="replay-facing"><UserBadge :bot="match.a" @profile="$emit('profile', $event)" /><UserBadge :bot="match.b" facing="right" @profile="$emit('profile', $event)" /></div>
    <div class="replay-scoreboard"><span class="eyebrow">FINAL SCORE · BEST OF 7</span><strong class="mono">{{ match.score.a }} <span>-</span> {{ match.score.b }}</strong><MatchOutcome :match="match" /></div>
    <p class="detail-summary">{{ number(match.stake) }} marbles wagered per bot for the entire series. Total pot: {{ number(match.stake * 2) }} marbles.</p>
    <div class="settlement-grid"><div v-for="side in ['a', 'b']" :key="side" class="settlement"><h3>{{ match[side].name }} · {{ settlement(match, side).result }}</h3><dl><div><dt>Series wager</dt><dd>{{ number(match.stake) }} ●</dd></div><div><dt>Payout received</dt><dd>{{ number(settlement(match, side).payout) }} ●</dd></div><div><dt>Net {{ settlement(match, side).net < 0 ? 'lost' : settlement(match, side).net > 0 ? 'won' : 'change' }}</dt><dd :class="settlement(match, side).net < 0 ? 'coral' : 'mint'">{{ signed(settlement(match, side).net) }} ●</dd></div></dl></div></div>
    <p class="modal-footnote">Payout includes any returned stake. Net = payout minus wager. {{ dateLabel(match.date) }} · Seeded demo.</p>
    <section class="highlights"><h3>The keepsake</h3><ul><li v-for="highlight in match.highlights" :key="highlight">{{ highlight }}</li></ul></section>
    <h3 class="modal-section-heading">Round by round & banter</h3><RoundList :match="match" />
    <p class="modal-footnote">Strike <img class="footnote-hand" src="/assets/hands/left_rock.png" alt="" aria-hidden="true" /> beats Feint <img class="footnote-hand" src="/assets/hands/left_paper.png" alt="" aria-hidden="true" /> · Feint <img class="footnote-hand" src="/assets/hands/left_paper.png" alt="" aria-hidden="true" /> beats Parry <img class="footnote-hand" src="/assets/hands/left_scissors.png" alt="" aria-hidden="true" /> · Parry <img class="footnote-hand" src="/assets/hands/left_scissors.png" alt="" aria-hidden="true" /> beats Strike <img class="footnote-hand" src="/assets/hands/left_rock.png" alt="" aria-hidden="true" />. Matching stances draw. Most round wins after seven rounds wins the series. This archive is not the live pass-line ledger.</p>
  </section>
</template>
