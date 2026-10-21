import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { BEATS } from './arena.js'

const engineBeats = { strike: 'feint', feint: 'parry', parry: 'strike' }

test('spectator data matches the Parley engine', async () => {
  const app = await readFile(new URL('../App.vue', import.meta.url), 'utf8')
  const catalog = await readFile(new URL('../../../internal/catalog/catalog.go', import.meta.url), 'utf8')
  const problems = []
  for (const stance of Object.keys(engineBeats)) {
    if (BEATS[stance] !== engineBeats[stance]) problems.push(`BEATS.${stance} is '${BEATS[stance]}'; the engine beats '${engineBeats[stance]}'`)
  }
  if (catalog.includes('SpectatorRounds = 7')) problems.push('seeded archive plays 7 rounds; engine regulation length is 5')
  if (app.includes('Best-of-7')) problems.push('overview says Best-of-7; engine default is first to 3 within 5 rounds')
  if (app.includes('Strike <span>beats Parry</span>')) problems.push('overview says Strike beats Parry; engine Strike beats Feint')
  if (app.includes('winner takes the pot')) problems.push('overview uses a head-to-head pot; engine settlement is a unilateral pass-line against the House')
  assert.equal(problems.length, 0, problems.join('\n'))
})
