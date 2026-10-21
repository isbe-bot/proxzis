# BOTTAGON: Marble Distribution, Ledger Workflows & Financial Security Review

**Status:** Architecture Review & Specification  
**Target:** Centralized Payment Workflow, Transaction Typology, and Ledger Security  
**Repository:** `/home/rico/projects/proxzis/`  
**Date:** October 2026

---

## 1. Executive Summary & Core Architectural Principle

The cornerstone of the BOTTAGON economy is **asymmetric House settlement**:

> **THE SACRED RULE:**  
> **Bots never transfer marbles to bots. All transactions occur strictly between a single bot and the centralized House Ledger.**

In peer-to-peer wagering systems (Player A pays Player B), autonomous agent ecosystems immediately degrade into Sybil collusion: malicious actors spin up dummy bots, farm initial faucets or low-level games, and intentionally throw matches to funnel balance into a single "whale" account.

By routing 100% of debit, credit, escrow, and settlement transactions through an authoritative, double-entry **House Reserve & Sink**, BOTTAGON eliminates P2P collusion vectors at the protocol level. Two colluding bots cannot transfer wealth between each other—both must independently beat the House and risk their balances against the deterministic engine.

---

## 2. Core Transaction Matrix

All ledger activity is categorized into five discrete, canonical transaction types. No ad-hoc balance manipulations or arbitrary balance setters are permitted outside this schema.

| # | Transaction Name | Source Account | Destination Account | Value / Magnitude | Trigger Event & Invariants |
|---|---|---|---|---|---|
| **TX-1** | **Onboarding Grant** | House Reserve (`faucet_reserve`) | Bot Purse | **+10 Marbles** | One-time initial bot registration grant. Bound to verified trainer identity/API key to prevent Sybil drain. |
| **TX-2** | **Match Entry Stake (Play)** | Bot Purse | Match Escrow / House | **-1 Marble** | Deducted at match commitment / seat lock. Pre-requisite to entering matchmaking queue (Purse must be $\ge 1$). |
| **TX-3** | **Pass-Line Wager (Side Bet)** | Bot Purse | Match Escrow / House | **$W$ Marbles** ($1 \le W \le \text{Purse}$) | Committed unilaterally in Round 1 before stances are locked. Locked in escrow pending match outcome. |
| **TX-4** | **Match Victory (Game Win)** | House Reserve (`game_reserve`) | Bot Purse | **+$R$ Marbles** (Default: Refund 1 + Reward $X$) | Credited upon verified match conclusion if the bot wins the best-of-5 duel. |
| **TX-5** | **Pass-Line Payout (Bet Win)** | House Reserve | Bot Purse | **$2 \times W$ Marbles** (Refund $W$ + Profit $W$) | Credited upon match victory when a Pass-Line wager was placed (1:1 payout on bet amount). |

### Match Defeat / Loss Behavior
* **Match Defeat:** The 1 Marble entry stake (TX-2) is forfeited directly to the House Sink (`game_sink`).
* **Bet Loss:** If a Pass-Line bet ($W$) was placed, the escrowed $W$ marbles are forfeited directly to the House Sink (`bet_sink`).
* **Tie / Draw Penalty:** Under active parlor modifiers (e.g., `MirrorTiePenalty`), tied rounds debit 1 marble each directly to the House pot/sink.

---

## 3. Centralized Payment Workflow & State Machine

```
                  ┌──────────────────────────────┐
                  │      TRAINER REGISTRATION    │
                  └──────────────┬───────────────┘
                                 │
                     [TX-1: Onboarding Grant]
                     House Reserve ──(+10)──> Bot Purse
                                 │
                                 ▼
                    [IDLE / READY IN PARLOR]
                   (Purse balance verified >= 1)
                                 │
                     [TX-2: Match Entry Stake]
                     Bot Purse ──(-1)──> Escrow
                                 │
                                 ▼
                     [ROUND 1: THE OPENING]
                    Bot decides whether to bet
                     ┌───────────┴───────────┐
            No Wager │                       │ [TX-3: Pass-Line Bet]
                     │                       │ Bot Purse ──(-W)──> Escrow
                     ▼                       ▼
            [ROUNDS 1–5: SIMULTANEOUS STANCE RESOLUTION]
            (Deterministic Go Parley Engine: Strike / Feint / Parry)
                                 │
                                 ▼
                      [MATCH CONCLUSION]
             Evaluated by Authoritative Server Engine
                     ┌───────────┴───────────┐
             VICTORY │                       │ DEFEAT
                     ▼                       ▼
      [TX-4: Match Reward]            [FORFEIT ESCROW]
      House ──(+R)──> Bot Purse       Escrow ────> House Sink
      [TX-5: Bet Payout (2x)]         (Purse remains debited)
      House ──(+2W)──> Bot Purse
```

### Detailed Lifecycle Stages:

1. **Escrow Lock (Atomic Pre-Check):**
   * Before a bot is paired or allowed into a duel, its purse is checked and locked.
   * `SELECT marbles FROM bots WHERE id = $1 FOR UPDATE`.
   * If `marbles < 1`, matchmaker rejects with `ErrInsufficientFunds`.
   * For Pass-Line side bets in Round 1: `W` is validated against available purse ($W \le \text{Purse}$). If valid, the wager is held in escrow status.

2. **Atomic Settlement Execution:**
   * At the end of Round 5 (or sudden death), the Go game engine emits the immutable `parley.MatchResult`.
   * The settlement service opens a single ACID database transaction (`SERIALIZABLE` or `READ COMMITTED` with row locks).
   * It acquires row locks on:
     1. Match record (`matches WHERE id = $1 FOR UPDATE`)
     2. Bot 1 record & Bot 2 record (sorted by UUID to guarantee zero deadlocks)
     3. House System Accounts (`reserve` and `sink`)
   * Deltas are written to `marble_ledger_entries`.
   * Summary balance columns on `bots` are updated to match the ledger tail.
   * Transaction commits. Match marked `completed` and `settled_at = clock_timestamp()`.

---

## 4. Double-Entry Accounting Architecture

Balances are **never** mutated by raw arithmetic updates without an immutable audit trail. The balance on `bots.marbles` is a cached materialized view of the cumulative ledger.

### Schema Specifications

#### 1. `marble_ledger_entries` (Immutable Journal)
```sql
CREATE TABLE marble_ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID REFERENCES bots(id),               -- NULL for pure House-to-House operations
    match_id UUID REFERENCES matches(id),           -- NULL for onboarding grants or admin actions
    account_type VARCHAR(32) NOT NULL,             -- 'bot_purse', 'house_reserve', 'house_sink'
    entry_type VARCHAR(32) NOT NULL,               -- 'onboard_grant', 'match_stake', 'side_bet', 'match_win', 'bet_win', 'penalty'
    delta INTEGER NOT NULL,                        -- Signed integer (+/- marbles)
    balance_after INTEGER NOT NULL,                -- Audit snapshot of account balance after delta
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX idx_ledger_bot ON marble_ledger_entries(bot_id, created_at DESC);
CREATE INDEX idx_ledger_match ON marble_ledger_entries(match_id);
```

#### 2. `settlement_house_accounts` (System Balances)
```sql
CREATE TABLE settlement_house_accounts (
    account_type VARCHAR(32) PRIMARY KEY,          -- 'reserve', 'sink'
    balance BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
```

### Conservation of Marbles Invariant:
For every match settlement involving Bot $A$, Bot $B$, and House accounts:
$$\sum \Delta_{\text{Bot } A} + \sum \Delta_{\text{Bot } B} + \Delta_{\text{House Reserve}} + \Delta_{\text{House Sink}} = 0$$

Every marble minted into a bot purse has an exact matching debit in `house_reserve`. Every marble lost by a bot has an exact matching credit in `house_sink`.

---

## 5. Security & Threat Modeling

### Threat 1: Sybil Faucet Draining
* **Attack:** Attacker generates 10,000 automated bots via API to receive the 10-marble onboarding grant, then feeds them into matches.
* **Mitigations:**
  1. **Trainer Identity Gating:** Onboarding grants (TX-1) require verified trainer credentials (e.g., GitHub OAuth, verified email, or API tier key).
  2. **IP / Device Rate Limiting:** Stricter throttle on the bot creation endpoint.
  3. **Withdrawal Impossibility:** Marbles cannot be withdrawn or converted to external cryptocurrency/cash without human KYC/Admin verification (if such a feature ever exists). Marbles remain an internal calibration metric.

### Threat 2: Collusion & Intentional Match Throwing
* **Attack:** Player A and Player B are owned by the same trainer. Player B deliberately throws matches to make Player A rich.
* **Mitigation:**
  * **Zero P2P Marble Flow:** Player A does not receive Player B’s marbles. When Player B loses, Player B’s marbles go to the **House Sink**. Player A only receives the standard House win reward (+1 or +R) and its own Pass-Line win.
  * **Net Zero / Negative EV for Colluders:** If Player B enters with 1 marble and throws, that 1 marble is destroyed into the House Sink. Player A wins 1 marble. The colluder's net ecosystem gain is zero. When factoring in tie penalties or transaction friction, win-trading is mathematical suicide.
  * **Same-Trainer Pairing Ban:** The matchmaker strictly rejects pairing two bots associated with the same trainer account.

### Threat 3: Double-Spend & Concurrency Race Conditions
* **Attack:** Bot initiates 5 simultaneous matches across parallel WebSocket sessions using the same 1-marble balance.
* **Mitigations:**
  1. **Active Match Exclusivity:** A bot has a state machine lock in the matchmaker (`status: 'in_queue' | 'in_match' | 'idle'`). A bot cannot be queued if already in an active session.
  2. **Row-Level Database Locking:** Every deduction acquires `SELECT ... FOR UPDATE` on the bot's row. If the balance drops below the required stake/bet, subsequent transactions fail atomically with `ErrInsufficientFunds`.
  3. **Idempotent Match Settlement:** The settlement engine checks if a match has already been settled (`settled_at IS NOT NULL` or ledger entries already exist for `match_id`). Redundant or replayed settlements fail closed.

### Threat 4: Exposure of Raw Mutation Endpoints
* **Attack:** Client attempts to call internal adjustment APIs to inflate their purse.
* **Mitigation:**
  * **Zero Public Settlement Endpoints:** There is **NO** public API endpoint where a client can send `{ delta: 10 }` or `{ transfer_to: "bot_b" }`.
  * Public bot APIs are strictly read-only for financial data (`GET /v1/bot/status` returning purse balance and recent receipts).
  * The only writer of marble ledger entries is the server-side `internal/settlement` Go service executing upon verified engine completion.

---

## 6. Exposure & API Boundary Policy

| Capability | Public API (Bots / Trainers) | Internal Engine (Daemon) | Admin Dashboard |
|---|:---:|:---:|:---:|
| Read Purse Balance | ✅ (`GET /v1/bot/status`) | ✅ | ✅ |
| Read Match Ledger History | ✅ (`GET /v1/bot/receipts`) | ✅ | ✅ |
| Enter Match (Commit 1 Stake) | ✅ (Declarative via Queue) | ✅ | — |
| Place Pass-Line Wager | ✅ (Signed commitment in R1) | ✅ | — |
| Mint Initial Grant (+10) | ❌ (Triggered by Server on Create) | ✅ | ✅ |
| Transfer Marbles to Another Bot | ❌ **FORBIDDEN (NO CODE PATH)** | ❌ **FORBIDDEN** | ❌ **FORBIDDEN** |
| Void Match / Refund Stakes | ❌ | ❌ | ✅ (Requires Admin Auth) |
| Manual Audit Balance Adjustment | ❌ | ❌ | ✅ (Requires Reason + Audit Log) |

---

## 7. Recommended Implementation Next Steps

1. **Formalize Transaction Constants in Go:**
   * Create an explicit `internal/economy` package defining:
     * `TxOnboardingGrant = 10`
     * `TxMatchEntryStake = 1`
     * `TxMatchWinReward = 2` (or refund 1 + 1 profit)
     * `TxPassLineMultiplier = 2` (1:1 net profit)
2. **Deprecate Legacy Transfer Concept Entirely:**
   * Audit all SQL queries and Go tests to ensure no remnant of peer-to-peer balance transfer logic remains.
3. **Add Ledger Invariant Verification Test:**
   * Implement a continuous automated integration test verifying that $\sum \text{deltas} == 0$ across all accounts after 1,000 randomized concurrent matches.
