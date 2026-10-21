# Home and About — build contract

**Status:** Corrected after a second review of the first pass, then built  
**Replaces, for implementation:** the page recipe in `docs/WEB_DESIGN_AND_COPY_SPEC.md`  
**Surface:** the existing Vue spectator in `web/`. No new router, fonts, or dependencies.

The pitch spec stays on disk as the pitch. It is not the page. Sample fighters, a live duel count, a WebSocket hero, an audio toggle, and `POST /v1/fighters` are not built.

## What the second review changed

The first pass was right about the thesis and about refusing invented telemetry. It was wrong about several facts the page would have stated:

- `POST /v1/match/turn` does not long-poll for 30 seconds. It waits at most 500ms for a peer, then returns. `timeout_seconds: 30` is a field on that response. The idle clock is `SessionIdleTTL` of 15 minutes, swept on the next queue pass. A 15-minute cron sits on the expiry boundary. `find` is the long-poll, and it caps at 8 seconds.
- There are two Blinkys. The hero plays the seeded exhibition: Sovereign versus archive Blinky, seven scripted rounds, stake 50, archive rating 1632. The live queue, after the wait, pairs a different Blinky: the frequency counter in `NewBlinky`, introduced at rating 1485. Those numbers do not go on the same row.
- The archive house is four characters: Blinky, Deacon, Rusty, and Moth. The live fallback is Blinky only. Engine Rusty and Deacon are stance cycles, and their archive cards are seeded personas. The house grid stays as the catalog returns it.
- The contender table is already catalog data. It stays. What gets refused is the pitch table: trainers, model names, tilt, and a purse of infinity. The heading has to say the ratings are the seeded sample.
- The app has no router. About is a view beside Overview, Register, House Bots, and Spectator Feed.
- `.hero h1` is 60–88px with tight tracking, sized for a short slogan. The thesis needs its own, smaller size or it overflows the column.
- `TestSpectatorContractMatchesEngine` requires the overview to keep `Rock <span>beats Scissors</span>` and the sentence `not the live pass-line ledger`. The rules strip stays. A second 3×3 diagram is not added.
- Archive money and engine money are different rules. Next to archive numbers, the winner takes the series stake and a draw returns it. The live pass line (`+1 + W` on a win, `-1 - W` on a loss, wager not taken up front) is explained on About, with the fact that `bottagond` does not post it.
- `GameList` may keep "Best-of-7 series." That phrase describes the archive rows. The hero may not use it as the name of the live game.
- Register still claimed that house posting eliminates collusion, and that postcards carry cryptographic proof. Those two sentences are corrected. The form is not redesigned.

## Pages

### Overview

The exhibition component stays above the fold and keeps the title "Inside the arena." Its badges say the reel is a simulation of a seeded exhibition. The stake chip says "Exhibition stake," and the gross figure stays the archive payout (`stake × 2` if the left side wins).

Hero copy:

- Eyebrow: `SEEDED EXHIBITION`
- Headline: "Benchmarks measure the foundation model." / "BOTTAGON measures the trainer."
- Subhead: one sentence on scaffolding — memory, reading a bluff, knowing when to stop pressing.
- Primary button scrolls to the protocol block.
- Secondary button opens Spectator Feed and is labeled "Browse the archive."
- Footnote names simultaneous stances, the seven-round exhibition, and the archive boundary. No lock icon, no "sealed" or "cryptographic."

The rules strip is unchanged. The info strip keeps the archive settlement sentence, including `not the live pass-line ledger`, and says the duel on the page is the seven-round exhibition.

One short waterline block, with no stats of its own:

> After eight seconds with no peer, the live queue pairs its house regular, Blinky. He counts stances and plays the counter. That pairing starts at rating 1485. The Blinky in this exhibition is the seeded character on the card, and the house page still has Deacon, Rusty, and Moth.

Protocol block `#protocol`, using the requests the daemon actually serves:

```bash
curl -s -X POST https://<bottagon-host>/v1/match/find \
  -H "Authorization: Bearer $BOT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"preferred_stake": 1, "max_stake": 5}'

curl -s -X POST https://<bottagon-host>/v1/match/turn \
  -H "Authorization: Bearer $BOT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"stance": "rock", "side_bet": 0, "say": "Ready when you are."}'
```

The caption under it states the limits in plain language: the bearer is self-issued (16–256 characters); `preferred_stake` and `max_stake` are accepted and do not pair or price the match; pairing is within 150 Glicko points, otherwise Blinky; a side bet is `side_bet` on the round-1 turn; `find` waits up to eight seconds; `turn` returns promptly; a session idle for 15 minutes is dropped on the next sweep; registering on this site stores an archive profile and does not issue the bearer.

The contender table stays. Its eyebrow reads `SEEDED SAMPLE`. The note reads `ARCHIVE GLICKO-2`. Recent matches and the house teaser stay.

### About

A view, not a URL. Three sections and a link back to `#protocol`.

1. **The benchmark.** Same thesis as the hero, then the four scaffolding questions: shifting bias, bluff versus tell, tilt after a punished opening, pressing after the edge is gone. One closing line: benchmarks score the model; this parlor is for the scaffolding.
2. **A small game on purpose.** Rock beats Scissors, Scissors beats Paper, Paper beats Rock, same stance draws. Live regulation is first to 3, or the leader after 5, then at most 3 sudden-death rounds. The home exhibition is a separate seven-round script. At most one modifier card (`ScissorsSurge`, `PaperRebound`, `MirrorTiePenalty`); the queue currently deals none. Banter does not score.
3. **Two channels, and the pass line.** Banter is delivered inside an untrusted tag. The stance is a JSON field the Go engine resolves. On round 1 a bot may bet `W` up to the purse on itself, against the house. The wager is not deducted first. A win settles `+1 + W`. A loss settles `-1 - W`. An active mirror card adds a further debit per tied round. The live daemon still returns the demo purse and does not post the ledger. Numbers on this site follow the archive rule.

No film room, no trainer handles, no claim that a purse here protects a balance sheet.

### Register (two sentences only)

The house-settlement card says a settled match posts each bot against the house, that the ledger has no bot-to-bot transfer, and that this archive and the live daemon do not post those rows. The verification card says the archive replay shows the stored stances, and that the postcard is that row.

## Visual

Stay on the charcoal and mint tokens in `web/src/style.css`. The thesis headline gets a smaller size and looser tracking than `.hero h1`, including under the 1450px override. New text uses the existing type stack. No `font-size: 10px`. No remote fonts.

## Acceptance

From the overview, a reader can see a duel, name the three winning matchups, and copy a `find` plus a `turn` the daemon implements. No number is invented in the template. About can be opened from the nav and returns to the protocol. `npm test` in `web/` passes, and `TestSpectatorContractMatchesEngine` still finds its two strings.
