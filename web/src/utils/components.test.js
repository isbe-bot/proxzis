import test from 'node:test'
import assert from 'node:assert/strict'
import { createServer } from 'vite'
import { createSSRApp } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { createMemoryHistory } from 'vue-router'
import { fileURLToPath } from 'node:url'
import { readFile, readdir } from 'node:fs/promises'
import { bots, matches } from './fixtures.js'

const root = fileURLToPath(new URL('../../', import.meta.url))

test('Vue components render real badges, fixed rows, facts and replay settlements', async t => {
  const server = await createServer({ root, server: { middlewareMode: true }, appType: 'custom' })
  const render = async (name, props) => {
    const component = await server.ssrLoadModule(`/src/components/${name}.vue`)
    return renderToString(createSSRApp(component.default, props))
  }
  try {
    await t.test('badge shows identity, tier, rating and seeded demo balance', async () => {
      const html = await render('AgentBadge', { bot: bots[0] })
      assert.match(html, /Sovereign-1/)
      assert.match(html, /Master/)
      assert.match(html, /1,684<\/span> rating/)
      assert.match(html, /marbles/)
      assert.match(html, /<a href="\/agents\/sovereign"/)
      assert.doesNotMatch(await render('AgentBadge', { bot: bots[0], link: false }), /<a /)
    })
    await t.test('games share identical center-column structure despite name lengths', async () => {
      const html = await render('GameList', { matches: matches.slice(0, 4), note: true })
      assert.equal((html.match(/class="game-center"/g) || []).length, 4)
      assert.equal((html.match(/class="game-row"/g) || []).length, 4)
      assert.equal((html.match(/href="\/matches\/AR-\d+"/g) || []).length, 4)
      assert.match(html, /per agent, per series, not per round/)
      const source = await readFile(`${root}src/components/GameList.vue`, 'utf8')
      assert.match(source, /grid-template-columns: 9\.5rem minmax\(0, 1fr\) 6rem minmax\(0, 1fr\)/)
      assert.match(source, /grid-template-columns: minmax\(0, 1fr\) 5\.5rem minmax\(0, 1fr\)/)
      assert.match(source, /grid-template-columns: minmax\(0, 1fr\) 4\.5rem minmax\(0, 1fr\)/)
      const unequalScores = await render('GameList', { matches: [{ ...matches[0], score: { a: 10, b: 1 } }] })
      assert.match(unequalScores, /<span class="score-a"[^>]*>10<\/span><span class="game-clash" aria-hidden="true"[^>]*>–<\/span><span class="score-b"[^>]*>1<\/span>/)
    })
    await t.test('standings records have ASCII hyphens, gross winning payouts and fixed columns', async () => {
      const html = await render('StandingsTable', { bots })
      assert.match(html, /W-L-D/)
      assert.match(html, /Marbles won/)
      assert.match(html, /\d+-\d+-\d+/)
      assert.doesNotMatch(html, /NET MARBLES|\/ RD|W \/ L/)
      const source = await readFile(`${root}src/components/StandingsTable.vue`, 'utf8')
      assert.match(source, /table-layout: fixed/)
    })
    await t.test('house and contender bots use the same universal facts', async () => {
      for (const bot of bots.slice(0, 2)) {
        const html = await render('BotCard', { bot })
        for (const label of ['Marble balance', 'Matches played', 'Record · W-L-D', 'Marbles won', 'gross winning payouts', 'Net earnings', 'Last played', 'Tactical tendencies']) assert.ok(html.includes(label), label)
        assert.match(html, /1,000 seeded opening marbles/)
        assert.ok(html.includes(bot.house ? 'House bot' : 'Contender'))
      }
    })
    await t.test('replay shows winner and loser accounting, seven rounds and banter', async () => {
      for (const match of [matches.find(m => m.winner === 'a'), matches.find(m => m.winner === 'b'), matches.find(m => m.winner === 'draw')]) {
        const html = await render('MatchReplay', { match })
        assert.match(html, /Final score/)
        assert.match(html, /Payout received/)
        assert.match(html, /Series wager/)
        assert.match(html, /Round by round &amp; banter/)
        assert.equal((html.match(/class="round-item"/g) || []).length, 7)
        assert.ok(html.includes(match.rounds[0].quoteA))
        if (match.winner === 'draw') assert.match(html, /stake returned/)
        else { assert.match(html, /Net won/); assert.match(html, /Net lost/) }
      }
    })
    await t.test('Home omits the registration form and shows the duel, agent handoff, protocol, standings and links', async () => {
      const { default: App } = await server.ssrLoadModule('/src/App.vue')
      const { createAppRouter } = await server.ssrLoadModule('/src/router.js')
      const router = createAppRouter(createMemoryHistory())
      await router.push('/')
      await router.isReady()
      const html = await renderToString(createSSRApp(App).use(router))
      assert.doesNotMatch(html, /id="registration"|id="bot-email"|id="register-heading"|<form/)
      assert.match(html, /Send your agent/)
      assert.match(html, /\/llms\.txt/)
      assert.match(html, /\/v1\/match\/find/)
      assert.match(html, /\/v1\/match\/turn/)
      assert.doesNotMatch(html, /\/v1\/fighters/)
      assert.match(html, /href="\/matches"/)
      assert.match(html, /href="\/register"/)
      assert.match(html, /href="\/about"/)
      assert.match(html, /Match chat/)
      assert.doesNotMatch(html, /Inside the arena/)
      assert.match(html, /Marbles won/)
      assert.match(html, /not the live pass-line ledger/)
      assert.doesNotMatch(html, /\(RD\)|\/ RD|W \/ L \/ D/)
    })
    await t.test('dedicated RegisterPage remains compatible with email validation', async () => {
      const html = await render('RegisterPage', { allBots: bots })
      assert.match(html, /type="email"/)
      assert.match(html, /<form/)
      const source = await readFile(`${root}src/components/RegisterPage.vue`, 'utf8')
      assert.equal((source.match(/validateRegistration\(name.value, email.value, persona.value, props.allBots\)/g) || []).length, 2)
    })
    await t.test('Routes wire universal cards, profile log, archive and replay', async () => {
      const view = name => readFile(`${root}src/views/${name}.vue`, 'utf8')
      assert.match(await view('HouseView'), /<BotCard v-for="bot in houseBots"/)
      const agent = await view('AgentView')
      assert.match(agent, /<BotCard :bot="profile"/)
      assert.match(agent, /<GameList v-if="profileLog.length"/)
      assert.match(await view('MatchesView'), /<GameList v-if="filteredMatches.length"/)
      assert.match(await view('MatchView'), /<MatchReplay v-if="detail" :match="detail"/)
      assert.match(await view('HomeView'), /not the live pass-line ledger/)
      const router = await readFile(`${root}src/router.js`, 'utf8')
      for (const path of ["'/'", "'/matches'", "'/matches/:id'", "'/agents/:id'", "'/house'", "'/about'", "'/register'"]) assert.ok(router.includes(`path: ${path}`), path)
      const store = await readFile(`${root}src/catalogStore.js`, 'utf8')
      assert.match(store, /from '\.\/api\/catalog\.js'/)
      for (const file of ['App.vue', 'views/HomeView.vue']) {
        const source = await readFile(`${root}src/${file}`, 'utf8')
        assert.equal((source.match(/validateRegistration\(/g) || []).length, 0, file)
      }
    })
    await t.test('shared hand glyphs render clean without transforms, conceal moves, and fade loser', async () => {
      const source = await readFile(`${root}src/components/HandGlyph.vue`, 'utf8')
      assert.doesNotMatch(source, /transform:\s*rotate/)
      assert.doesNotMatch(source, /transform:\s*scaleX/)
      assert.match(source, /\.hand-win\s*\{\s*opacity:\s*1;/)
      assert.match(source, /\.hand-lose\s*\{\s*opacity:\s*0\.32;/);
      for (const outcome of ['win', 'lose', 'draw']) {
        const html = await render('HandGlyph', { stance: 'paper', side: 'b', owner: 'Right bot', outcome })
        assert.ok(html.includes(`hand-${outcome}`))
        assert.ok(html.includes('side-b'))
        assert.match(html, /Right bot: Paper, round/)
      }
      const concealed = await render('HandGlyph', { stance: 'paper', side: 'a', owner: 'Left bot', concealed: true, outcome: 'win' })
      assert.match(concealed, /closed fist, stance concealed/)
      assert.match(concealed, /✊/)
      assert.doesNotMatch(concealed, /✌️|hand-win|round won/)
      const live = await render('LiveMatch', { left: bots[0], right: bots[1], rounds: [{ a: 'rock', b: 'paper', quoteA: 'a', quoteB: 'b' }], stake: 50 })
      assert.equal((live.match(/stance concealed/g) || []).length, 2)
      assert.doesNotMatch(live, /hand-win|hand-lose|hand-draw/)
      const toggle = live.match(/<button class="chat-toggle"[^>]*>/)?.[0] || ''
      assert.match(toggle, /aria-expanded="false"/)
      assert.match(toggle, /aria-label="Show match chat"/)
      const chatId = toggle.match(/aria-controls="([^"]+)"/)?.[1]
      assert.ok(chatId && live.includes(`id="${chatId}"`), 'chat toggle controls the chat log')
      const liveSource = await readFile(`${root}src/components/LiveMatch.vue`, 'utf8')
      for (const side of ['a', 'b']) assert.ok(liveSource.includes(`handOutcome(phase, winner, '${side}')`))
      assert.match(liveSource, /prefers-reduced-motion: reduce/)
      assert.equal((liveSource.match(/timeline\.lift/g) || []).length, 1)
      assert.doesNotMatch(live, /Choosing/)
    })
    await t.test('winning badges identify either bot with its avatar; draws are visually distinct', async () => {
      for (const winner of ['a', 'b', 'draw']) {
        const match = matches.find(m => m.winner === winner)
        const badge = await render('MatchOutcome', { match })
        const games = await render('GameList', { matches: [match] })
        const replay = await render('MatchReplay', { match })
        for (const html of [badge, replay]) {
          if (winner === 'draw') {
            assert.match(html, /outcome-draw/)
            assert.match(html, /aria-label="Draw: both stakes returned"/)
          } else {
            assert.ok(html.includes(`aria-label="Winner: ${match[winner].name}"`))
            assert.ok(html.includes(`${match[winner].name} won`))
          }
        }
        assert.ok(games.includes(winner === 'draw' ? 'Draw · stakes returned' : `${match[winner].name} won`))
        assert.equal((badge.match(/class="agent-mark/g) || []).length, winner === 'draw' ? 0 : 1)
        if (winner !== 'draw') assert.ok(badge.includes(`agent-mark side-${winner} on-side`))
      }
    })
    await t.test('every seeded round renders two labelled, inward-facing actual hand glyphs in archive and replay', async () => {
      const glyphs = { rock: '✊', paper: '✋', scissors: '✌️' }
      const art = { rock: 'rock.png', paper: 'paper.png', scissors: 'scissors.png' }
      for (const stance of Object.keys(art)) {
        const hand = await render('HandGlyph', { stance, side: 'a', owner: 'Left bot' })
        assert.ok(hand.includes(`/assets/hands/left_${art[stance]}`), `${stance} shows the ${stance} hand`)
      }
      for (const match of matches) {
        const rounds = await render('RoundList', { match, compact: true })
        assert.equal((rounds.match(/role="img"/g) || []).length, 14)
                for (const round of match.rounds) for (const side of ['a', 'b']) {
          assert.ok(rounds.includes(glyphs[round[side]]))
          const name = round[side][0].toUpperCase() + round[side].slice(1)
          assert.ok(rounds.includes(`aria-label="${match[side].name}: ${name}"`))
        }
      }
      const html = await render('GameList', { matches: [matches[0]], expandable: true })
      assert.match(html, /<details class="game-rounds"[^>]*><summary aria-label="Round stances and results for AR-1048"[^>]*>7 rounds/)
      assert.equal((html.match(/class="round-item"/g) || []).length, 7)
    })
    await t.test('design-system files take every color and type size from tokens', async () => {
      const files = ['styles/base.css', 'App.vue']
      for (const dir of ['components', 'views']) {
        for (const entry of await readdir(`${root}src/${dir}`)) if (entry.endsWith('.vue')) files.push(`${dir}/${entry}`)
      }
      assert.ok(files.length > 20)
      for (const file of files) {
        const source = await readFile(`${root}src/${file}`, 'utf8')
        const style = source.includes('<style') ? source.slice(source.indexOf('<style')) : source
        assert.doesNotMatch(style, /#[0-9a-fA-F]{3,8}\b|rgba?\(|hsla?\(/, `${file} uses a raw color`)
        assert.doesNotMatch(style, /font-size:\s*[\d.]+(px|rem)\b/, `${file} uses a fixed font size instead of a step token`)
      }
      const tokens = await readFile(`${root}src/styles/tokens.css`, 'utf8')
      const steps = [...tokens.matchAll(/--step-(-?\d):/g)].map(m => m[1])
      assert.deepEqual(steps, ['-2', '-1', '0', '1', '2', '3', '4', '5', '6', '7'])
      assert.match(tokens, /--step--2: 0\.75rem;/)
    })
    await t.test('font-only 10px migration and larger records/replay text have regression guards', async () => {
      async function checkFonts(dir) {
        for (const entry of await readdir(dir, { withFileTypes: true })) {
          if (entry.isDirectory()) await checkFonts(`${dir}/${entry.name}`)
          else if (/\.(vue|css)$/.test(entry.name)) {
            const source = await readFile(`${dir}/${entry.name}`, 'utf8')
            assert.doesNotMatch(source, /\bfont(?:-size)?\s*:\s*10px\b|text-\[10px\]/, entry.name)
          }
        }
      }
      await checkFonts(`${root}src`)
      const source = file => readFile(`${root}src/components/${file}.vue`, 'utf8')
      assert.match(await source('BotCard'), /\.card-facts dd\.record \{ font-size: var\(--step-2\); \}/)
      assert.match(await source('AgentBadge'), /\.badge-meta \{ font-size: var\(--step--1\);/)
      assert.match(await source('MatchReplay'), /\.block-title \{[^}]*font-size: var\(--step-3\);/)
      assert.match(await source('RoundList'), /\.round-quote \{[^}]*font-size: var\(--step--1\);/)
      const badge = await render('AgentBadge', { bot: bots[0] })
      assert.match(badge, /class="badge-record" aria-label="\d+ wins, \d+ losses, \d+ draws"/)
      const card = await render('BotCard', { bot: bots[0] })
      assert.match(card, /<dd class="record"[^>]*>\d+-\d+-\d+<\/dd>/)
    })
  } finally { await server.close() }
})
