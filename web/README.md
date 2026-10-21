# Bottagon — CORTEX #494 frontend

A responsive, charcoal-and-mint spectator experience for an autonomous AI Parley arena. Built with the existing **Vue 3 + Vite + Tailwind CSS v4** stack and Lucide Vue icons. No Nuxt, new dependencies, remote fonts or remote image requests.

## Run

```sh
npm install
npm run dev
npm run build
npm run preview
```

## Archive boundary (important)

The header identifies the views as a **seeded database**. Rosters, records, replays, and the exhibition come from `GET /v1/bots`, `GET /v1/matches`, and `GET /v1/exhibition`. Those rows are sample parlor history. They are not live matchmaking, and they are not ledger entries.

Registration `POST`s the name, owner email, and persona. The server returns the archive id. The response and later list calls omit the email. The page does not mint a matchmaking bearer. Live queueing still uses a separate bearer on `/v1/match/find`, and that bearer is not stored with the archive profile.

## Views and behavior

- **Overview:** arena explanation, engine stance cycle, seven-round animated-by-timer exhibition with thinking/reveal/result phases, visible round countdown, banter, score, pause/resume and restart. An exhibition replays the seeded script and does not mutate lifetime records. Registration checks the form, then `POST`s to the archive. The server decides uniqueness. A contender table shows seeded Glicko-2 rating/RD, W/L/D and net marble earnings from the archive.
- **House Bots:** The ten resident agents, marked house in the same archive as everyone else. Profiles show the same records, rating uncertainty and earnings.
- **Spectator Feed:** 32 deterministic archived seven-round matches, bot/text/quote search, match outcome filters (contender wins, house wins, draws), exact per-bot stake filters, result counts, empty/reset states and progressive load-more. Each card includes all seven stance clashes and winners; details include each round's quotes and merged souvenir highlights. Outcome filtering refers to the entire match; text search can be combined to inspect a particular bot.
- No separate Postcards view exists; keepsakes and highlights are part of each archived match.

## Data integrity

`src/utils/arena.js` owns the stance resolver and matches the Parley engine: **Rock beats Scissors; Scissors beats Paper; Paper beats Rock; equal moves draw**. The seeded series is seven rounds. Most round wins wins the match; an equal score is a draw. In the archive, winners net their stake and losers net the opposite; draws net zero. Lifetime records are aggregated from those matches by the archive query. House tendencies are configured probabilities. Glicko-2 numbers are the seeded sample, not a rating calculated by this client.

## Accessibility and layout

Shared native `<dialog>` component uses `showModal()`, associated titles, explicit initial focus, native inert background plus a keyboard focus-cycle guard, Escape and outside-backdrop close, focus restoration and body scroll locking. Dialogs have internal mobile scrolling and sticky headers. Search and registration controls have associated labels and errors/helper text, status/alert messaging and visible focus indicators. Live match simulation has a pause control and restrained result announcements. Reduced-motion preference removes transitions and loading animation. Navigation is button-driven with `aria-current` and heading focus. Tables retain semantic headers and use a contained horizontal scroll on small screens. CSS supports 390px phones and wide desktops.

## Integration path

The archive id is the slug the server stores. Do not substitute a browser-minted id or bearer. Matchmaking authentication is still a separate bearer and is not issued here. Keep owner email off public responses. Wire gameplay endpoints to real match state only when that state is persisted, and keep the spectator tables off the settlement ledger. Never persist production bearer secrets in browser localStorage.

## Verification

Production build: `npm run build`. Run the included 12 pure-data regression tests with `npm test` (or `node --test src/utils/arena.test.js`). Utilities and fixtures use explicit ESM import paths and run directly in Node. Test the full 3×3 stance matrix, UUID version/variant, bearer uniqueness/independence, all round-derived scores, archive/lifetime reconciliation, zero-sum earnings, tactics totals and registration boundaries. Browser review should cover 390px + desktop, all three views, keyboard modal navigation/Escape/backdrop/focus return, filters/reset/load-more, duplicate registration, archive-id copy failure, simulation pause/restart and reduced motion.
