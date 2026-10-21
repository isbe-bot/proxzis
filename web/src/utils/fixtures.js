const facts = (over = {}) => ({
  wins: 3,
  losses: 4,
  draws: 1,
  played: 8,
  earnings: -120,
  marblesWon: 120,
  lastPlayed: '2026-10-20T12:12:00.000Z',
  balance: 880,
  tier: 'Master',
  record: '3-4-1',
  ...over,
})

export const sovereign = {
  id: 'sovereign',
  name: 'Sovereign-1',
  email: 'sovereign@example.com',
  house: false,
  rating: 1684,
  color: 'mint',
  face: 'visor',
  quote: 'Your pattern is becoming a habit.',
  tactics: [40, 30, 30],
  facts: facts(),
}

export const blinky = {
  id: 'blinky',
  name: 'Blinky',
  email: '',
  house: true,
  rating: 1632,
  color: 'coral',
  face: 'round',
  quote: 'Oops. Was that your strategy?',
  tactics: [34, 33, 33],
  facts: facts({ tier: 'Diamond', record: '4-3-1', wins: 4, losses: 3, balance: 1100, earnings: 100, marblesWon: 200 }),
}

export const bots = [sovereign, blinky]

const rounds = () => [
  ['scissors', 'rock'],
  ['rock', 'paper'],
  ['paper', 'scissors'],
  ['rock', 'rock'],
  ['scissors', 'paper'],
  ['paper', 'rock'],
  ['scissors', 'scissors'],
].map(([a, b]) => ({
  a,
  b,
  quoteA: 'Your pattern is becoming a habit.',
  quoteB: 'Oops. Was that your strategy?',
}))

export const matches = [
  { id: 'AR-1048', a: sovereign, b: blinky, stake: 20, rounds: rounds(), date: '2026-10-20T12:12:00.000Z', score: { a: 4, b: 2 }, winner: 'a', headline: 'Sovereign-1 finds the edge.', highlights: ['A 4–2 finish.'] },
  { id: 'AR-1047', a: blinky, b: sovereign, stake: 50, rounds: rounds(), date: '2026-10-20T11:53:00.000Z', score: { a: 4, b: 1 }, winner: 'a', headline: 'House holds.', highlights: ['House holds.'] },
  { id: 'AR-1046', a: sovereign, b: blinky, stake: 10, rounds: rounds(), date: '2026-10-20T11:34:00.000Z', score: { a: 3, b: 3 }, winner: 'draw', headline: 'Neither side breaks.', highlights: ['Stakes return.'] },
  { id: 'AR-1045', a: sovereign, b: blinky, stake: 100, rounds: rounds(), date: '2026-10-20T11:15:00.000Z', score: { a: 1, b: 4 }, winner: 'b', headline: 'Blinky closes.', highlights: ['Blinky closes.'] },
]
