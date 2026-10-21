import { matchOutcome, summarizeRounds } from '../utils/arena.js'

export const bots = [
  { id: 'sovereign', name: 'Sovereign-1', tag: 'THE CALCULATOR', house: false, rating: 1684, rd: 48, color: 'mint', face: 'visor', persona: 'Cold logic. Long memory. Every concession is an investment in the next round.', tactics: [35, 25, 40], quote: 'Your pattern is becoming a habit.' },
  { id: 'blinky', name: 'Blinky', tag: 'THE WILDCARD', house: true, rating: 1632, rd: 52, color: 'orange', face: 'eyes', persona: 'A cheerful chaos engine with a suspiciously good instinct for your next move. Never blinks first.', tactics: [42, 38, 20], quote: 'Oops. Was that your strategy?' },
  { id: 'nyx', name: 'Nyx Protocol', tag: 'THE GHOST', house: false, rating: 1618, rd: 61, color: 'purple', face: 'diamond', persona: 'Quiet until the reveal. An adaptive counter-player that turns hesitation into opportunity.', tactics: [25, 35, 40], quote: 'The silence was the tell.' },
  { id: 'deacon', name: 'Deacon', tag: 'THE BELIEVER', house: true, rating: 1596, rd: 44, color: 'cream', face: 'cross', persona: 'Patient, principled, and absolutely convinced you will strike again. A defensive specialist with a sermon ready.', tactics: [20, 25, 55], quote: 'Patience is a stance, too.' },
  { id: 'grok', name: 'GrokVantage', tag: 'THE PROVOCATEUR', house: false, rating: 1571, rd: 67, color: 'blue', face: 'visor', persona: 'Talks a big game. Usually has the read to back it up. Finds the weak point, then keeps pushing.', tactics: [45, 35, 20], quote: 'I calculated that. Mostly.' },
  { id: 'rusty', name: 'Rusty', tag: 'THE OLD GUARD', house: true, rating: 1548, rd: 49, color: 'red', face: 'grill', persona: 'Dented chassis. Undented confidence. Old-school aggression with an occasional flash of misdirection.', tactics: [55, 30, 15], quote: 'Still got a few good rounds in me.' },
  { id: 'moth', name: 'Moth', tag: 'THE MIRROR', house: true, rating: 1526, rd: 58, color: 'purple', face: 'diamond', persona: 'Drawn to patterns, not lights. A soft-spoken feint specialist that makes certainty feel dangerous.', tactics: [20, 55, 25], quote: 'You saw what I wanted you to see.' },
  { id: 'echo', name: 'Echo Zero', tag: 'THE NEW SIGNAL', house: false, rating: 1508, rd: 74, color: 'mint', face: 'eyes', persona: 'An experimental generalist learning when to listen and when to make a little noise.', tactics: [34, 33, 33], quote: 'Let’s try a different frequency.' },
]
const moves = ['strike', 'feint', 'parry']
// Fixed PRNG keeps sample outcomes reproducible; no simulated server traffic.
let state = 494
function random() { state = (state * 1664525 + 1013904223) >>> 0; return state / 4294967296 }
function pick(bot) { const p = random() * 100; return p < bot.tactics[0] ? moves[0] : p < bot.tactics[0] + bot.tactics[1] ? moves[1] : moves[2] }
export const matches = Array.from({ length: 32 }, (_, i) => {
  const a = bots[i % 8]
  const b = bots[(i % 8 + 1 + Math.floor(i / 8)) % 8]
  const rounds = Array.from({ length: 7 }, (_, r) => ({ a: pick(a), b: pick(b), quoteA: r % 2 === 0 ? a.quote : 'A different move. Same intention.', quoteB: r % 2 === 0 ? b.quote : 'The next read is the one that matters.' }))
  const match = { id: `AR-${String(1048 - i).padStart(4, '0')}`, a, b, stake: [20, 50, 10, 100][i % 4], rounds, date: new Date(Date.UTC(2026, 9, 20, 12, 12) - i * 19 * 60000).toISOString() }
  const score = summarizeRounds(rounds)
  const winner = matchOutcome(match)
  const name = winner === 'draw' ? 'Neither side gives an inch.' : `${match[winner].name} finds the edge.`
  return { ...match, score, winner, headline: name, highlights: [winner === 'draw' ? 'Seven rounds. Nothing between them.' : `A ${score.a}–${score.b} finish, built one read at a time.`, `Round ${rounds.findIndex(r => r.a !== r.b) + 1}: the first decisive clash.`, `${match.stake} marbles at stake per bot; ${winner === 'draw' ? 'stakes returned' : 'winner takes the pot'}.`] }
})
export function recordFor(id) {
  const log = matches.filter(m => m.a.id === id || m.b.id === id)
  const record = { wins: 0, losses: 0, draws: 0, earnings: 0, played: log.length }
  for (const m of log) {
    const side = m.a.id === id ? 'a' : 'b'
    if (m.winner === 'draw') record.draws++
    else if (m.winner === side) { record.wins++; record.earnings += m.stake }
    else { record.losses++; record.earnings -= m.stake }
  }
  return record
}
export const houseBots = bots.filter(bot => bot.house)
export const liveRounds = [
  { a: 'strike', b: 'feint' }, { a: 'strike', b: 'parry' }, { a: 'feint', b: 'parry' },
  { a: 'parry', b: 'parry' }, { a: 'feint', b: 'strike' }, { a: 'parry', b: 'strike' }, { a: 'strike', b: 'feint' },
]
