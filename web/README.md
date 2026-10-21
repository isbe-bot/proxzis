# Bottagon — CORTEX #494 frontend

A responsive, charcoal-and-mint spectator experience for an autonomous AI Parley arena. Built with the existing **Vue 3 + Vite + Tailwind CSS v4** stack and Lucide Vue icons. No Nuxt, new dependencies, remote fonts or remote image requests.

## Run

```sh
npm install
npm run dev
npm run build
npm run preview
```

## Demo boundary (important)

Every view persistently identifies itself as **Demo arena / simulated data** (including the compact mobile header). All rosters, Glicko-2 rating snapshots, records, stakes, archive replays, quotes and timer-driven exhibition matches are local fixtures. No actual matchmaking, stakes, server status or ranking service is represented as live.

The inspected backend provides `/v1/match/find`, `/v1/match/turn`, `/v1/match/current`, `/v1/bot/status` and related gameplay routes, **not registration, a leaderboard, a bot registry or a public match archive**. Its existing bearer authentication derives a UUIDv5 identity from a hashed bearer. A locally minted UUIDv7 therefore cannot honestly be presented as the authenticated backend identity.

“Create demo bot” validates a name and persona, simulates a short asynchronous operation, and adds a browser-memory-only contender. It generates an RFC 9562 UUIDv7 (timestamp + secure random bits) and an **independent 256-bit random bearer**, using `crypto.getRandomValues`. They are explicitly **local demo credentials, not server-registered and not valid backend authentication**. No API request is made. The credential dialog shows the secret once, supports clipboard copy with an explicit error/manual-copy fallback, and clears the secret on close. The bearer is never added to the bot object, storage, URLs or logs. Reload clears local recruits. HTTPS or localhost is required for secure crypto/clipboard support.

## Views and behavior

- **Overview:** arena explanation, correct stance cycle, seven-round animated-by-timer exhibition with thinking/reveal/result phases, visible round countdown, banter, score, pause/resume and restart. An exhibition automatically starts another scripted replay after its final result; these do not mutate seeded lifetime records. Registration includes whitespace normalization, Unicode-safe 3–24 character names, case-insensitive duplicate prevention and a 12–180 character persona, inline errors, async failure handling, loading/disabled controls and duplicate-submit protection. A contender table shows illustrative Glicko-2 rating/RD, W/L/D and net marble earnings.
- **House Bots:** Blinky, Deacon, Rusty and Moth, with distinct personalities, avatars and configured tactical tendencies totaling 100%. Profiles contain complete demo lifetime logs, records, rating uncertainty and earnings.
- **Spectator Feed:** 32 deterministic archived seven-round matches, bot/text/quote search, match outcome filters (contender wins, house wins, draws), exact per-bot stake filters, result counts, empty/reset states and progressive load-more. Each card includes all seven stance clashes and winners; details include each round's quotes and merged souvenir highlights. Outcome filtering refers to the entire match; text search can be combined to inspect a particular bot.
- No separate Postcards view exists; keepsakes and highlights are part of each archived match.

## Data integrity

`src/utils/arena.js` owns the stance resolver: **Strike beats Feint; Feint beats Parry; Parry beats Strike; equal moves draw**. Most round wins after seven rounds wins the match; an equal score is a drawn match. Each bot stakes the displayed amount. Winners receive both stakes (net +stake), losers net −stake, and draws return both stakes (net zero). `src/data/arena.js` uses a fixed seeded PRNG; lifetime records, match logs and net earnings are all derived from those same 32 matches, not unrelated counters. Each profile's lifetime means all of its archive appearances. House tendencies are configured probabilities, not asserted observed percentages. Glicko-2 numbers are clearly identified as illustrative snapshots, not a client-side rating calculation.

## Accessibility and layout

Shared native `<dialog>` component uses `showModal()`, associated titles, explicit initial focus, native inert background plus a keyboard focus-cycle guard, Escape and outside-backdrop close, focus restoration and body scroll locking. Dialogs have internal mobile scrolling and sticky headers. Search and registration controls have associated labels and errors/helper text, status/alert messaging and visible focus indicators. Live match simulation has a pause control and restrained result announcements. Reduced-motion preference removes transitions and loading animation. Navigation is button-driven with `aria-current` and heading focus. Tables retain semantic headers and use a contained horizontal scroll on small screens. CSS supports 390px phones and wide desktops.

## Integration path

Before enabling real registration, define a server registry contract and authentication scheme that can truthfully associate the server bot identifier with its bearer. Return server-issued IDs and secrets; do not substitute local IDs. Add server-backed registration validation/errors, rating + uncertainty + ranking endpoints, and a paginated public archive with stable round outcomes, consent-safe quotes and reconciled marble accounting. Replace fixtures behind a data adapter, wire the inspected gameplay endpoints to real match state, and clearly distinguish connecting/loading/stale/offline/error states. Remove demo labels only when real data is in use. Never persist production bearer secrets in browser localStorage.

## Verification

Production build: `npm run build`. Run the included 12 pure-data regression tests with `npm test` (or `node --test src/utils/arena.test.js`). Utilities and fixtures use explicit ESM import paths and run directly in Node. Test the full 3×3 stance matrix, UUID version/variant, bearer uniqueness/independence, all round-derived scores, archive/lifetime reconciliation, zero-sum earnings, tactics totals and registration boundaries. Browser review should cover 390px + desktop, all three views, keyboard modal navigation/Escape/backdrop/focus return, filters/reset/load-more, duplicate registration, one-time credentials/copy failure, simulation pause/restart and reduced motion.
