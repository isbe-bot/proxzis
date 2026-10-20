# PostgreSQL settlement contract

Apply migration `00004_settlement_accounts.sql` before using `Service`.

`SettleMatch` requires an existing match row and both existing bot rows. The
engine's UUID must match the supplied match UUID, and the supplied ordered
participants must match `matches.p1_id` / `matches.p2_id`. Callers must serialize
mutation of the engine while settlement reads it.

The transaction locks the match, then both bots in UUID order, then the house
reserve and sink in fixed order. `matches.settled_at`, not `status`, is the durable
financial completion marker. A completed engine may settle an existing pending,
in-progress, or completed database row. An identical retry returns success
without changing balances, ratings, timestamps, or ledger entries. A retry with
a different winner, score, wager, mirror count, or sudden-death flag fails with
`ErrSettlementConflict`.

## Accounting

Side bets are unilateral against the house and are not escrowed by this service.
Preserved net player deltas are:

- Winner: `1 + wager - mirror_ties`.
- Loser: `-1 - wager - mirror_ties`.
- Draw: `-mirror_ties` for each player, with no wager movement.

Each player gets one `bot` ledger entry and one opposite system-account entry.
Positive/zero net player deltas debit `house_reserve`; negative net player deltas
credit `house_sink`. Thus every new settled match has four entries whose deltas
sum to zero, even with unequal wagers or mirror penalties. These are **net**
postings, not separate postings for each base reward, wager, and penalty.
System entries have `bot_id = NULL`; playable `is_house` bots remain contestants,
not system counterpart accounts.

System balances start at zero on migration and use signed PostgreSQL BIGINT.
The reserve may go negative, preserving the existing unlimited-house payout
semantics. Contestant balances retain PostgreSQL INTEGER bounds; only playable
house bots may go negative. Deltas must also have a representable opposite in
INTEGER. Input, balance, house overflow, checked-rating, and SQL failures roll
back all postings, bot changes, and the settlement marker.

## Historical data and limits

The migration does not rewrite or pretend to reconcile historical ledger rows.
An unmarked match with any existing ledger rows fails closed for manual
reconciliation: older settlement could already have paid multiple times.
Existing bot opening balances/grants are not backfilled. The schema's old
ledger remains directly mutable by sufficiently privileged SQL clients; this
change does not add a general ledger immutability or deferred balance trigger.
Balance and replay guarantees apply to transactions through this service.

Migration 00004 deliberately refuses Down to avoid silently discarding financial
history and replay protection. Plan an explicit data-preserving downgrade if one
is required.

## Verification

```sh
go test -race ./internal/settlement -count=1 -v
go vet ./internal/settlement
```

Integration tests discover local PostgreSQL binaries (`PATH`, then Debian's
`/usr/lib/postgresql/*/bin`), create a fresh temporary cluster, disable its TCP
listener, apply the actual Goose migrations, and use only its unique Unix socket.
They never accept a database URL or connect to an existing database. Cleanup
stops only that owned cluster and removes its temporary directory. Integration
tests explicitly skip if binaries are unavailable or the test user is root;
validation-only tests still run. No container, package install, or server service
configuration is needed.
