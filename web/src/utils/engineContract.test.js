import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { BEATS } from './arena.js'

const engineBeats = { rock: 'scissors', scissors: 'paper', paper: 'rock' }

test('spectator rounds use the Parley engine and stay off the ledger', async () => {
  const app = await readFile(new URL('../views/HomeView.vue', import.meta.url), 'utf8')
  const store = await readFile(new URL('../../../internal/catalog/sqlstore.go', import.meta.url), 'utf8')
  const problems = []
  for (const stance of Object.keys(engineBeats)) {
    if (BEATS[stance] !== engineBeats[stance]) problems.push(`BEATS.${stance} is '${BEATS[stance]}'; the engine beats '${engineBeats[stance]}'`)
  }
  if (!app.includes('Rock <span>beats Scissors</span>')) problems.push('overview does not say Rock beats Scissors')
  if (!app.includes('not the live pass-line ledger')) problems.push('overview does not say the archive is separate from the live ledger')
  if (/INTO bots|marble_ledger|SET marbles/.test(store)) problems.push('spectator SQL writes the settlement ledger')
  assert.equal(problems.length, 0, problems.join('\n'))
})
