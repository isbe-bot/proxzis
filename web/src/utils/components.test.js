import test from 'node:test'
import assert from 'node:assert/strict'
import { createServer } from 'vite'
import { createSSRApp } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { fileURLToPath } from 'node:url'
import { readFile, readdir } from 'node:fs/promises'
import { bots, matches } from '../data/arena.js'

const root = fileURLToPath(new URL('../../', import.meta.url))

test('Vue components render real badges, fixed rows, facts and replay settlements', async t => {
  const server = await createServer({ root, server: { middlewareMode: true }, appType: 'custom' })
  const render = async (name, props) => {
    const component = await server.ssrLoadModule(`/src/components/${name}.vue`)
    return renderToString(createSSRApp(component.default, props))
  }
  try {
    await t.test('badge shows identity, tier, rating and seeded demo balance', async () => {
      const html = await render('UserBadge', { bot: bots[0] })
      assert.match(html, /Sovereign-1/)
      assert.match(html, /Master/)
      assert.match(html, /1,684 rating/)
      assert.match(html, /demo marbles/)
      assert.match(html, /<button/)
      assert.doesNotMatch(await render('UserBadge', { bot: bots[0], interactive: false }), /<button/)
    })
    await t.test('games share identical center-column structure despite name lengths', async () => {
      const html = await render('GameList', { matches: matches.slice(0, 4) })
      assert.equal((html.match(/class="game-center"/g) || []).length, 4)
      assert.equal((html.match(/class="game-row"/g) || []).length, 4)
      assert.equal((html.match(/BEST OF 7/g) || []).length, 4)
      assert.match(html, /Series wager/)
      assert.match(html, /faces-left/)
      assert.match(html, /per series — not per round/)
      const css = await readFile(`${root}src/style.css`, 'utf8')
      assert.match(css, /grid-template-columns:minmax\(0,1fr\) 144px minmax\(0,1fr\)/)
      assert.match(css, /grid-template-columns:minmax\(0,1fr\) 120px minmax\(0,1fr\)/)
      assert.match(css, /grid-template-columns:minmax\(0,1fr\) 20px minmax\(0,1fr\)/)
      const unequalScores = await render('GameList', { matches: [{ ...matches[0], score: { a: 10, b: 1 } }] })
      assert.match(unequalScores, /class="game-score-value">10<\/span><span class="game-clash" aria-hidden="true">-<\/span><span class="game-score-value">1<\/span>/)
    })
    await t.test('leaderboard records have ASCII hyphens and gross winning payouts', async () => {
      const html = await render('LeaderboardTable', { bots })
      assert.match(html, /W-L-D/)
      assert.match(html, /Marbles won/)
      assert.match(html, /\d+-\d+-\d+/)
      assert.doesNotMatch(html, /NET MARBLES|\/ RD|W \/ L/)
    })
    await t.test('house and player bots use the same universal facts', async () => {
      for (const bot of bots.slice(0, 2)) {
        const html = await render('BotCard', { bot })
        for (const label of ['Marble balance', 'Matches played', 'Record · W-L-D', 'Marbles won · gross', 'Net earnings', 'Last played', 'Tactical tendencies']) assert.ok(html.includes(label), label)
        assert.match(html, /1,000 seeded opening marbles/)
        assert.ok(html.includes(bot.house ? 'HOUSE BOT' : 'PLAYER BOT'))
      }
    })
    await t.test('replay shows winner and loser accounting, seven rounds and banter', async () => {
      for (const match of [matches.find(m => m.winner === 'a'), matches.find(m => m.winner === 'b'), matches.find(m => m.winner === 'draw')]) {
        const html = await render('MatchReplay', { match })
        assert.match(html, /FINAL SCORE/)
        assert.match(html, /Payout received/)
        assert.match(html, /Series wager/)
        assert.match(html, /Round by round &amp; banter/)
        assert.equal((html.match(/class="round-item"/g) || []).length, 7)
        assert.ok(html.includes(match.rounds[0].quoteA))
        if (match.winner === 'draw') assert.match(html, /stake returned/)
        else { assert.match(html, /Net won/); assert.match(html, /Net lost/) }
      }
    })
    await t.test('Overview omits the registration card but keeps live play, registration navigation and games', async () => {
      const { default: App } = await server.ssrLoadModule('/src/App.vue')
      const html = await renderToString(createSSRApp(App))
      assert.doesNotMatch(html, /id="registration"|id="bot-email"|id="register-heading"|<form/)
      assert.match(html, /Enter the arena/)
      assert.match(html, /Register/)
      assert.match(html, /Inside the arena/)
      assert.match(html, /class="game-list panel"/)
      assert.match(html, /Marbles won/)
      assert.doesNotMatch(html, /\(RD\)|\/ RD|W \/ L \/ D/)
    })
    await t.test('dedicated RegisterPage remains compatible with email validation', async () => {
      const html = await render('RegisterPage', { allBots: bots })
      assert.match(html, /type="email"/)
      assert.match(html, /<form/)
      const source = await readFile(`${root}src/components/RegisterPage.vue`, 'utf8')
      assert.equal((source.match(/validateRegistration\(name.value, email.value, persona.value, props.allBots\)/g) || []).length, 2)
    })
    await t.test('App wires universal cards, profile log, archive and replay', async () => {
      const source = await readFile(`${root}src/App.vue`, 'utf8')
      assert.match(source, /<BotCard v-for="bot in houseBots"/)
      assert.match(source, /<BotCard :bot="profile"/)
      assert.match(source, /<GameList v-if="profileLog.length"/)
      assert.match(source, /<GameList v-if="filteredMatches.length"/)
      assert.match(source, /<MatchReplay :match="detail"/)
      assert.equal((source.match(/validateRegistration\(name.value, email.value, persona.value, allBots.value\)/g) || []).length, 2)
    })
    await t.test('shared hand glyphs render clean without transforms, conceal moves, and fade loser', async () => {
      const source = await readFile(`${root}src/components/HandGlyph.vue`, 'utf8')
      assert.doesNotMatch(source, /transform:\s*rotate/)
      assert.doesNotMatch(source, /transform:\s*scaleX/)
      assert.match(source, /\.hand-win\s*\{\s*opacity:\s*1;/)
      assert.match(source, /\.hand-lose\s*\{\s*opacity:\s*0\.32;/);
      for (const outcome of ['win', 'lose', 'draw']) {
        const html = await render('HandGlyph', { stance: 'parry', side: 'b', owner: 'Right bot', outcome })
        assert.ok(html.includes(`hand-${outcome}`))
        assert.ok(html.includes('side-b'))
        assert.match(html, /Right bot: Parry, round/)
      }
      const concealed = await render('HandGlyph', { stance: 'parry', side: 'a', owner: 'Left bot', concealed: true, outcome: 'win' })
      assert.match(concealed, /closed fist, stance concealed/)
      assert.match(concealed, /✊/)
      assert.doesNotMatch(concealed, /✌️|hand-win|round won/)
      const live = await render('LiveMatch', {})
      assert.equal((live.match(/stance concealed/g) || []).length, 2)
      assert.doesNotMatch(live, /hand-win|hand-lose|hand-draw/)
      const liveSource = await readFile(`${root}src/components/LiveMatch.vue`, 'utf8')
      for (const side of ['a', 'b']) assert.ok(liveSource.includes(`handOutcome(phase, winner, '${side}')`))
      assert.match(liveSource, /prefers-reduced-motion: reduce/)
      assert.equal((liveSource.match(/timeline.handOffset/g) || []).length, 1)
    })
    await t.test('winning badges identify either bot with its avatar; draws are visually distinct', async () => {
      for (const winner of ['a', 'b', 'draw']) {
        const match = matches.find(m => m.winner === winner)
        const badge = await render('MatchOutcome', { match })
        const games = await render('GameList', { matches: [match] })
        const replay = await render('MatchReplay', { match })
        for (const html of [badge, games, replay]) {
          if (winner === 'draw') {
            assert.match(html, /outcome-draw/)
            assert.match(html, /aria-label="Draw: both stakes returned"/)
          } else {
            assert.ok(html.includes(`aria-label="Winner: ${match[winner].name}"`))
            assert.ok(html.includes(`${match[winner].name} won`))
          }
        }
        assert.equal((badge.match(/class="bot-avatar/g) || []).length, winner === 'draw' ? 0 : 1)
        if (winner !== 'draw') assert.ok(badge.includes(`bot-avatar ${match[winner].color} ${match[winner].face}`))
      }
    })
    await t.test('every seeded round renders two labelled, inward-facing actual hand glyphs in archive and replay', async () => {
      const glyphs = { strike: '👊', feint: '🖐️', parry: '✌️' }
      for (const match of matches) {
        const rounds = await render('RoundList', { match, compact: true })
        assert.equal((rounds.match(/role="img"/g) || []).length, 14)
                for (const round of match.rounds) for (const side of ['a', 'b']) {
          assert.ok(rounds.includes(glyphs[round[side]]))
          const name = round[side][0].toUpperCase() + round[side].slice(1)
          assert.ok(rounds.includes(`aria-label="${match[side].name}: ${name}"`))
        }
      }
      const html = await render('GameList', { matches: [matches[0]] })
      assert.match(html, /<details class="game-rounds"><summary aria-label="Round stances and results for AR-1048">7 rounds/)
      assert.equal((html.match(/class="round-item"/g) || []).length, 7)
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
      const css = await readFile(`${root}src/style.css`, 'utf8')
      assert.match(css, /\.record\s*\{\s*font-size:14px/)
      assert.match(css, /\.bot-facts dd\.record\{font-size:18px/)
      assert.match(css, /\.badge-record\{font-size:14px/)
      assert.match(css, /\.match-replay \.modal-section-heading\{font-size:17px/)
      assert.match(css, /\.match-replay \.round-quotes,\.match-replay \.round-quotes strong\{font-size:13px/)
      // These non-font dimensions deliberately remain 10px.
      assert.match(css, /padding:10px 16px/)
      assert.match(css, /gap:10px 20px/)
      assert.match(css, /width:10px/)
      const badge = await render('UserBadge', { bot: bots[0] })
      assert.match(badge, /class="badge-record mono" aria-label="\d+ wins, \d+ losses, \d+ draws"/)
      const card = await render('BotCard', { bot: bots[0] })
      assert.match(card, /<dd class="record">\d+-\d+-\d+<\/dd>/)
    })
  } finally { await server.close() }
})
