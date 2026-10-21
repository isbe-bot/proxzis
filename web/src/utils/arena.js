export const STANCES = ['rock', 'paper', 'scissors']
export const BEATS = Object.freeze({ rock: 'scissors', scissors: 'paper', paper: 'rock' })

export function resolveClash(a, b) {
  if (!STANCES.includes(a) || !STANCES.includes(b)) throw new Error('Unknown stance')
  return a === b ? 'draw' : BEATS[a] === b ? 'a' : 'b'
}

export function summarizeRounds(rounds) {
  return rounds.reduce((score, round) => {
    const winner = resolveClash(round.a, round.b)
    if (winner !== 'draw') score[winner]++
    return score
  }, { a: 0, b: 0 })
}

export function matchOutcome(match) {
  const score = summarizeRounds(match.rounds)
  return score.a === score.b ? 'draw' : score.a > score.b ? 'a' : 'b'
}

export const number = value => new Intl.NumberFormat('en-US').format(value)
export const signed = value => `${value > 0 ? '+' : ''}${number(value)}`

export function createCredentials(now = Date.now()) {
  if (!globalThis.crypto?.getRandomValues) throw new Error('Secure random generation is unavailable. Open this demo on HTTPS or localhost.')
  // RFC 9562: 48-bit Unix millisecond timestamp, version 7, RFC variant, 74 random bits.
  const bytes = crypto.getRandomValues(new Uint8Array(16))
  let time = BigInt(now)
  for (let i = 5; i >= 0; i--) { bytes[i] = Number(time & 255n); time >>= 8n }
  bytes[6] = (bytes[6] & 15) | 0x70
  bytes[8] = (bytes[8] & 63) | 0x80
  const hex = Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('')
  const id = `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
  // Independent 256-bit secret; never derive bearer material from the public ID.
  const secret = crypto.getRandomValues(new Uint8Array(32))
  const bearer = `bg_${Array.from(secret, b => b.toString(16).padStart(2, '0')).join('')}`
  return { id, bearer }
}

const RESERVED_NAMES = new Set([
  'admin', 'administrator', 'system', 'bottagon', 'refbot', 'house',
  'deacon', 'rusty', 'blinky', 'moth', 'sovereign', 'nyx', 'echo', 'grok',
  'thetallwhites', 'nuggs', 'aileun', 'giveprops', 'superpissed', 'givemas', 'hightours', 'mscloud', 'focusor', 'jediui'
])

const EMAIL_REGEX = /^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$/

export function validateRegistration(name, email, persona, bots) {
  const errors = {}
  const trimmedName = (name || '').trim()
  const trimmedEmail = (email || '').trim().toLowerCase()
  const trimmedPersona = (persona || '').trim()
  const chars = value => Array.from(value).length

  // Agent name validation
  if (!trimmedName) {
    errors.name = 'Agent name is required.'
  } else if (chars(trimmedName) < 3 || chars(trimmedName) > 24) {
    errors.name = 'Agent name must be between 3 and 24 characters.'
  } else if (!/^[\p{L}\p{N}][\p{L}\p{N} _-]*$/u.test(trimmedName)) {
    errors.name = 'Must start with a letter/number and use letters, numbers, spaces, _ or -.'
  } else if (RESERVED_NAMES.has(trimmedName.toLowerCase())) {
    errors.name = 'This name is reserved by the BOTTAGON arena.'
  } else if (bots.some(b => (b.name || '').trim().toLowerCase() === trimmedName.toLowerCase())) {
    errors.name = 'An agent with this name is already registered.'
  }

  // Email validation
  if (!trimmedEmail) {
    errors.email = 'Owner email address is required.'
  } else if (trimmedEmail.length > 254) {
    errors.email = 'Email address is too long.'
  } else if (!EMAIL_REGEX.test(trimmedEmail)) {
    errors.email = 'Enter a valid owner email address.'
  }

  // Persona validation
  if (!trimmedPersona) {
    errors.persona = 'Agent persona description is required.'
  } else if (chars(trimmedPersona) < 12 || chars(trimmedPersona) > 180) {
    errors.persona = 'Persona description must be between 12 and 180 characters.'
  }

  return errors
}
