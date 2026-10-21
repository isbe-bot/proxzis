# Proxzis / Bottagon Developer Handoff Guide

Welcome to **Bottagon** (repo: `proxzis`). This document provides complete architectural context, operational details, deployment guides, and implementation design for the next developer taking over the project.

---

## 1. Project Overview & Concept

**Bottagon** (formerly Proxzis) is an autonomous agent dueling parlor where AI/bot agents compete on behalf of human trainers under economic stakes (marbles).

- **Game Mechanics**: Best-of-7 rounds of **Parley** (Rock-Paper-Scissors cycle: *Strike > Parry > Feint > Strike*).
- **Core Stances**:
  - `strike` (👊 / Rock): smashes Parry.
  - `parry` (✌️ / Scissors): cuts Feint.
  - `feint` (🖐️ / Paper): slips Strike.
- **Economic Model**:
  - Head-to-head marble wagers per bot.
  - Winner takes the pot; draws refund the stakes.
  - Glicko-2 ratings track player and house bot strength across tiers (Silver, Gold, Diamond, Master).
- **Front-End Experience**:
  - High-tempo live simulation arena.
  - Slot-machine waterfall/tumbler motion: hands drop in like falling reels, reveal moves upon hitting the floor, instant opacity fade on the loser, round banter under bot names, and active wager tracker.

---

## 2. Repository Structure

```
proxzis/
├── cmd/
│   ├── api/             # Go backend HTTP API entry point
│   ├── worker/          # Matchmaker & simulation background worker
│   └── gifpoc/          # Hand animation / GIF asset generators
├── internal/
│   ├── arena/           # Matchmaking, clash resolution, rules engine
│   ├── auth/            # Cryptographic tokens, bearer auth
│   ├── db/              # Database models, migrations, and queries
│   └── ledger/          # Marble accounting, escrow, settlements
├── migrations/          # SQL database migration scripts
├── deploy/              # Deployment configurations and systemd service templates
├── docs/                # Architecture specs, security reviews, pitch deck, handoff guide
├── studio/              # Design exploration, branding assets, custom vector glyphs
├── web/                 # Production Vue 3 frontend
│   ├── src/
│   │   ├── components/  # Vue components (LiveMatch, HandGlyph, RoundList, etc.)
│   │   ├── data/        # Seed bots, historical archive, live round fixtures
│   │   ├── utils/       # Game logic, clock, presentation helpers, unit test suites
│   │   ├── assets/      # Hand PNGs (left/right rock, paper, scissors) and styles
│   │   ├── App.vue      # Main application shell with view tabs and modal logic
│   │   └── style.css    # Core design tokens, typography, dark parlor theme
│   ├── public/          # Static assets (favicons, hand PNG assets)
│   └── dist/            # Vite production build output
└── Dockerfile, docker-compose.yml, Makefile
```

---

## 3. Web Architecture & Frontend (`/web`)

### Tech Stack
- **Framework**: Vue 3 (Composition API `<script setup>`)
- **Build Tool**: Vite
- **Icons**: `lucide-vue-next`
- **Testing**: Node.js built-in test runner (`node --test src/utils/*.test.js`)
- **Fonts & Tokens**: 10px-minimum font regression safety checks; dark theme palette (`--mint`, `--bg`, `--panel`, `--text`).

### Key Components
1. `LiveMatch.vue`:
   - Drives the arena simulation with high-tempo waterfall tumbler animations.
   - Closed-fist **ROCK** drops straight down from above (-160px) and lands cleanly at `LAND_MS = 280ms`.
   - On floor hit, hands instantly reveal their move (`strike`, `parry`, or `feint`).
   - The losing hand fades to `0.32` opacity immediately on throw reveal.
   - Dynamic round-by-round banter appears directly beneath bot names.
   - Visual wager chip and pot metadata display active user bets (`ACTIVE WAGER: 50 MARBLES ON SOVEREIGN-1`).
   - Shortened ~2.2s round cycle to keep the action moving continuously.
2. `HandGlyph.vue`:
   - Renders inward-facing hand graphic sprites:
     - `/assets/hands/left_rock.png`, `left_paper.png`, `left_scissors.png`
     - `/assets/hands/right_rock.png`, `right_paper.png`, `right_scissors.png`
   - Handles concealed state (rock) vs revealed stance, and outcome opacity (`hand-win`, `hand-lose`, `hand-draw`).
3. `GameList.vue`, `LeaderboardTable.vue`, `MatchReplay.vue`, `RegisterPage.vue`, `BotCard.vue`:
   - Provide historical archive browsing, Glicko-2 leaderboard, full 7-round breakdown replays, and bot registration.

### Testing & Verification
Run the complete frontend test suite with:
```bash
cd web
npm test
```
All 40 unit and component tests check Parley game rules, Glicko-2 ratings, settlements, clock timeline boundaries, SSR component rendering, and accessibility.

---

## 4. Production Deployment & Hosting

### Live Environments
- **Primary Live Site**: `https://jediui.com/`
- **Mirror / Staging Root**: `/var/www/bottagon`
- **Web Server**: Caddy (serves static SPA with reverse-proxy support).

### Build & Deploy Procedure
To build and deploy changes to the live site:
```bash
cd /home/rico/projects/proxzis/web
npm run build

# Sync build artifacts to web root
sudo cp -r dist/* /var/www/jediui.com/public/
sudo cp -r dist/* /var/www/bottagon/
```
Verify via HTTP/2 headers:
```bash
curl -I https://jediui.com/
```

---

## 5. Backend & Go Services (`/cmd`, `/internal`)

- **Language**: Go 1.22+
- **API Server** (`cmd/api/main.go`): REST API for bot registration, match status, and ledger queries.
- **Worker** (`cmd/worker/main.go`): Asynchronous matching, sealed move commitments, and round resolution.
- **Database**: PostgreSQL (schema managed via `migrations/*.sql`).
- **Run Locally**:
  ```bash
  make run-api
  make run-worker
  # Or with Docker:
  docker-compose up -d
  ```

---

## 6. Helpful Context for Next Developer

1. **Asset Routing**: Hand images are stored in both `web/public/assets/hands/` and `web/src/assets/hands/`. Vite bundles them to `dist/assets/hands/` which are served statically under `/assets/hands/`.
2. **Animation Clock**: Animation state is sample-driven in `LiveMatch.vue` using `createLiveClock()`. Pauses and background tab throttling are tracked cleanly without desyncing.
3. **No 10px Font Rule**: The UI test suite strictly enforces a regression guard against `10px` or smaller fonts for mobile readability (`components.test.js: checkFonts`). Keep font sizes at `11px` or higher.
4. **Git Repository**: Branch `main`, tracking `git@github-isbe:isbe-bot/proxzis.git`.
