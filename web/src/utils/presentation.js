import { matches } from '../data/arena.js'
import { number } from './arena.js'

export const DEMO_OPENING_BALANCE = 1000
export const STANCE_GLYPHS = Object.freeze({ strike: '👊', feint: '🖐️', parry: '✌️' })
export const stanceName = stance => ({ strike: 'Strike', feint: 'Feint', parry: 'Parry' }[stance])
// Outcome styling starts with the scored result, never during concealed/reveal cues.
export function handOutcome(phase, winner, side) {
  if (phase !== 'result' && phase !== 'complete') return null
  return winner === 'draw' ? 'draw' : winner === side ? 'win' : 'lose'
}
export function humanTier(rating) {
  if (rating >= 1650) return 'Master'
  if (rating >= 1600) return 'Diamond'
  if (rating >= 1550) return 'Gold'
  return 'Silver'
}
export const recordLabel = record => `${record.wins}-${record.losses}-${record.draws}`
export function dateLabel(date) {
  return date ? `${new Intl.DateTimeFormat('en-GB', { month: 'short', day: '2-digit', hour: '2-digit', minute: '2-digit', timeZone: 'UTC' }).format(new Date(date))} UTC` : 'Not played yet'
}
export function settlement(match, side) {
  const draw = match.winner === 'draw'
  const won = match.winner === side
  return { wager: match.stake, payout: draw ? match.stake : won ? match.stake * 2 : 0, net: draw ? 0 : won ? match.stake : -match.stake, result: draw ? 'Draw · stake returned' : won ? 'Won' : 'Lost' }
}
export function botFacts(bot, archive = matches) {
  const log = archive.filter(m => m.a.id === bot.id || m.b.id === bot.id)
  const facts = { wins: 0, losses: 0, draws: 0, played: log.length, earnings: 0, marblesWon: 0, lastPlayed: null }
  for (const match of log) {
    const side = match.a.id === bot.id ? 'a' : 'b'
    const outcome = settlement(match, side)
    if (match.winner === 'draw') facts.draws++
    else if (match.winner === side) { facts.wins++; facts.marblesWon += outcome.payout }
    else facts.losses++
    facts.earnings += outcome.net
    if (!facts.lastPlayed || match.date > facts.lastPlayed) facts.lastPlayed = match.date
  }
  return { ...facts, balance: DEMO_OPENING_BALANCE + facts.earnings, tier: humanTier(bot.rating), record: recordLabel(facts) }
}
export const marbleLabel = value => `${number(value)} marbles`
