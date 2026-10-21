# BOTTAGON: Marble Distribution, Ledger Workflows & Financial Security

**Status:** Corrected against the settlement service and the live daemon  
**Repository:** `bottagon`  
**Date:** October 2026

The house-only rule is already the settlement contract. This note records that contract, marks the parts of the earlier draft that are not features, and orders the work that is still missing.

---

## 1. The rule that already holds

> Bots never transfer marbles to bots. A settled match posts each bot against the House, never against the other bot.

`internal/settlement.SettleMatch` is the only writer of match ledger rows. There is no transfer function, no `transfer_to` field, and no peer balance update. Searching for a leftover peer-to-peer path is a confirmation pass, not a migration.

What the rule does **not** do: it does not make win-trading unprofitable. The House pays the winner's base reward and pass-line profit. A thrown match with no side bet is zero-sum between the two purses (`+1` and `-1`). A thrown match where the winner also holds a pass-line of `W` is not. The winner's net is `+1 + W` and the loser's is `-1`, so the pair extracts `W` from `house_reserve`. The reserve is allowed to go negative. Collusion control is therefore a matchmaking and purse problem, not something the ledger topology already solved.

---

## 2. What exists today

### 2.1 Two different marble worlds

| Surface | What it does | What it must not do |
|---|---|---|
| Live parlor (`bottagond` matchmaker + `/v1/*`) | In-memory Parley. Purse reported by the API is the constant `1000`, with `"demo": true`. | It does not read or write `bots` or `marble_ledger_entries`. `SettleMatch` is not called from the daemon. |
| Settlement service (`internal/settlement`) | ACID net posting for a match row that already exists, plus both bot rows. | It does not create bots, grant opening balances, or escrow a stake at queue time. |
| Spectator archive (`internal/catalog`) | Seeded exhibition history for the site. Copy in the replay says it is not the live pass-line ledger. | Tests reject catalog writes into `bots`, `marble_ledger_entries`, or `SET marbles`. |

`preferred_stake` and `max_stake` on `POST /v1/match/find` are range-checked and then ignored. Pairing is Glicko distance (±150) or the House regular Blinky after the queue wait. The engine purse for a live match is `matchmaker.DemoPurse` (1000), not a database balance.

### 2.2 Payout math

`parley.SettlePassLineChecked` is the source of truth. `Match.Settle` calls it. Settlement persists the **net** delta, not a row per economic event.

For each contestant, with side bet `W` and `M` mirror-tie rounds:

| Result | Net delta on that bot |
|---|---|
| Win | `+1 + W - M` |
| Loss | `-1 - W - M` |
| Draw (sudden death exhausted, no winner) | `-M` |

`W` is optional, Round 1 only, and must be `0 ≤ W ≤` the purse stored on the match config. Later rounds may repeat the locked wager or send `0`; they may not change it. A winning `W` adds `+W`, not `+2W`, because the wager was never taken out of the purse beforehand. The component fields (`base_reward_won`, `pass_line_refund`, `pass_line_profit`, `pass_line_loss`, `mirror_penalty_paid`) explain that net. They are not separate ledger postings.

Mirror penalties apply only while the match's modifier card is `MirrorTiePenalty`. They are counted during the match and charged once, at settlement. `ScissorsSurge` and `PaperRebound` do not move marbles.

Regulation is first to 3, or the leader after 5 rounds. A tied regulation match goes to at most 3 sudden-death rounds, then draws. This is not the spectator archive's seven-round exhibition.

### 2.3 Ledger the service actually writes

Migration `00002` creates the tables. Migration `00004` adds house accounts and `matches.settled_at`. Do not replace those tables with the earlier draft's DDL.

`settlement_house_accounts` has two rows, `house_reserve` and `house_sink`, each a `BIGINT` balance starting at 0. There is no `updated_at`, no `faucet_reserve`, and no `bet_sink`.

`marble_ledger_entries.account_type` is `bot`, `house_reserve`, or `house_sink`. A bot row requires `bot_id`. A house row requires `bot_id` NULL.

One new settlement inserts four rows whose deltas sum to 0:

- Each bot gets one row, `entry_type` `match_win` (net ≥ 0) or `match_loss` (net < 0).
- A positive or zero bot delta debits `house_reserve` by the opposite amount, `entry_type` `match_counterpart`.
- A negative bot delta credits `house_sink` the same way.

The schema comment still lists `grant_initial`, `allowance`, `match_reward`, `match_penalty`, `side_bet_win`, `side_bet_loss`, and `mirror_penalty`. The settlement writer does not emit those types. Leave them as historical comment, or stop writing them in new docs. Do not start emitting them unless the net row is being replaced on purpose.

`bots.marbles` is a cache updated in the same transaction as the ledger row. It is not a view. The default column value is **1000**, not a 10-marble grant. Opening balances are not backfilled into the ledger. Historical rows are not rewritten. Privileged SQL can still mutate the ledger; there is no immutability trigger.

Lock order inside `READ COMMITTED`: match row, then the two bots in UUID order, then `house_reserve`, then `house_sink`. `settled_at` is the completion marker, not `status`. An identical retry returns success and writes nothing. A retry with a different winner, score, wager, mirror count, or sudden-death flag returns `ErrSettlementConflict`. An unmarked match that already has ledger rows fails closed. A non-house bot cannot be driven below 0 (`ErrInsufficientFunds`); a playable `is_house` bot may. The reserve may go negative. Contestant balances and ledger deltas stay inside PostgreSQL `INTEGER`.

`internal/settlement/postgres_test.go` already asserts the four-row, sum-to-zero invariant, replay, conflict, bounds, and concurrent settlement of two matches. A further randomized soak is useful only after the daemon actually calls this service.

---

## 3. Draft items that are not features

| Draft claim | Actual state |
|---|---|
| TX-1 onboarding grant of +10 from `faucet_reserve`, gated by GitHub OAuth, verified email, or an API tier | No grant writer, no trainer table, no OAuth. Auth is a self-issued bearer of 16–256 characters, hashed to a UUIDv5. Any new token is a new bot. |
| TX-2: deduct 1 marble at seat lock, reject the queue with `ErrInsufficientFunds` when purse < 1 | The matchmaker never reads a purse. `ErrInsufficientFunds` is only returned while applying a settlement delta that would make a non-house bot negative. |
| TX-3: move `W` into a match escrow during Round 1 | No escrow account or status. The engine stores `P1SideBet` / `P2SideBet` on the match config. The live purse used for the cap is the demo 1000. |
| TX-4 reward of "refund 1 + profit", and TX-5 payout of `2 × W` | Persisted winner delta is `+1 + W - M`. Adding a 2× payout on top of the current net posting would pay the wager twice. |
| Loss sends the entry stake to `game_sink` and the bet to `bet_sink` | One negative net credits `house_sink` once. |
| `GET /v1/bot/receipts` | Not a route. `GET /v1/bot/status` returns the authenticated bot id plus the hardcoded demo purse. It does not return match receipts. |
| `POST /v1/fighters` and a registration grant | Not a route. The web copy spec's curl example is not this API. |
| Admin void, refund, or manual balance adjustment with an audit reason | No admin API and no dashboard action. |
| Bot status enum `idle \| in_queue \| in_match` persisted in Postgres | In-process maps only. A second `find` for the same bearer returns the open session, or `ErrAlreadyQueued` if that bearer is already waiting. A second bearer is a second bot. Restart clears the lock. |
| Same-trainer pairing ban | No trainer entity, so two tokens from one person can be paired. |
| IP / device rate limit on bot creation | No rate limiter. There is no creation endpoint to limit. |
| Withdrawal, cash-out, or KYC | Marbles are not convertible. Nothing here implements or sketches a cash-out flow. |
| Continuous 1,000-match invariant job | Single-match conservation is already tested. The live parlor does not settle, so a soak of the daemon would currently assert nothing financial. |
| `internal/economy` constants `TxMatchWinReward = 2` and `TxPassLineMultiplier = 2` | Would disagree with `SettlePassLine`. Do not add that package. |

---

## 4. Work still worth doing

Do these in order. Each one is small because the math and the SQL transaction already exist.

### 4.1 Confirm there is still no peer path

Search settlement, the daemon, and tests for a writer that changes one bot's marbles from another bot's delta. The expected result is that only `applySettlement` changes `bots.marbles`, and only by that bot's own net. Record the result in the settlement README if the search finds a comment that still talks about escrow or a 2× credit.

### 4.2 Connect the parlor to the service

On match completion and on forfeit, the daemon should persist the match and call `SettleMatch` with the same engine UUID and the same ordered participants. Until that call exists, purse, ledger, and ratings on the public API are demonstration values.

The call has to happen after the engine is finished and while the session lock is still held, which is the race the settlement README already warns about. Forfeit already forces a winner in memory and then drops the session; that path needs the same settlement call or forfeits stay free.

### 4.3 Read and write the real purse

- Create or load the `bots` row for the bearer-derived id.
- Put that balance into `MatchConfig.P1Purse` / `P2Purse` instead of `DemoPurse`.
- Return it from `GET /v1/bot/status`, and drop `"demo": true` only once the value comes from the row.
- Keep the Round 1 cap `W ≤ purse` in the engine. Also reject settlement if the persisted `p1_side_bet` / `p2_side_bet` disagree with the engine, which `SettleMatch` already does on retry.

Do not invent a receipt list until a caller needs it. Status can stay a balance, rating, and honor response.

### 4.4 Hold the maximum loss only if two matches can charge one purse

Today one bearer cannot be in two live matches in one process, and settlement refuses a negative non-house balance. That is enough while the daemon is single-process and one token is one bot.

Add a hold only when that exclusivity is no longer true (multiple daemon instances, or two tokens sharing a purse). The hold is a debit of `1 + W` at the Round 1 commit, in the same transaction that inserts the match row, posted as the bot's negative delta and a `house_sink` or a single match-scoped hold account. Settlement then posts only the remainder (`+2 + 2W` back on a win, so the net is still `+1 + W`; mirror charges stay an extra debit). Do not add `faucet_reserve`, `bet_sink`, or five public transaction codes. Do not escrow and also keep the current net formula.

### 4.5 Opening balance stays 1000 until identity is real

A 10-marble grant paid to every new bearer is an open faucet: the client mints the credential. Keep the schema default of 1000 and label it as the parlor's starting purse. A scarce grant waits on a real issued credential. That work is an identity project, not a ledger project. It is out of scope here.

Same-trainer pairing has the same dependency. There is nothing to ban until a trainer id exists on both bots.

### 4.6 Keep the conservation test where it already is

Extend `TestPostgresSettlement` when the payout formula changes. Add a daemon-level test that one completed live match produces four ledger rows summing to 0 and moves `bots.marbles` by the net above. A randomized 1,000-match soak is optional after that test is red/green, and it should call `SettleMatch` (or the daemon) rather than reimplement the arithmetic.

---

## 5. Security boundaries that are already true

- No public handler accepts a marble delta, a counterparty bot, or a settlement request. Clients submit `find`, `turn` (stance, Round 1 `side_bet`, banter), `current`, `forfeit`, `status`, and `report`.
- Replay of a settled match does not pay twice.
- A settlement that would overflow `INTEGER`, or make a non-house purse negative, rolls back the match marker, both purses, both ratings, and the house balances.
- The catalog store is not a settlement client.

Boundaries that are still open, in the order they matter once 4.2 lands:

1. Pass-line profit is minted by an unfunded reserve. Two coordinated tokens can farm it by throwing. Mitigate with single-match exclusivity first, then a shared-owner ban once owners exist. Do not describe the current ledger as already making that farm negative-EV.
2. The in-memory queue lock does not survive a second process or a restart.
3. The ledger table is not append-only against a database superuser.
4. Self-issued bearers are not accounts. Rate limits and grants attached to "registration" have no object to attach to yet.
