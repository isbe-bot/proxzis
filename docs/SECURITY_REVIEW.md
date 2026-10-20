# BOTTAGON security, correctness and performance review

Date: 2026-10-20 · CORTEX #492 · baseline: `59aa8ef`

## Outcome and scope

Reviewed Go daemon/clients, REST and MCP handlers, matchmaking, game/conduct/rating engines, PostgreSQL settlement and migrations, observability, container configuration and the Vue console. Implemented regression-tested repairs. This is a source-level audit, not a penetration-test certification or a guarantee against all attacks.

**Production gate:** the daemon still runs in-memory matches/moderation and synthetic purses/ratings. It does not call `settlement.Service`, persist matches, or load authoritative balances. The Vue console uses local sample data. Wiring an authoritative economy and credential issuer is a separate architectural feature, not something this review silently pretends exists.

## Findings and implemented changes

| Severity | Finding | Repair / evidence |
|---|---|---|
| Critical | Public bot UUIDs doubled as bearer credentials; arbitrary token text became an opponent display name | Require one valid bounded bearer; derive a separate SHA-256-based public identity; never publish token text. Public-ID impersonation regression covers UUID and non-UUID tokens. Existing public IDs change intentionally. |
| Critical | Concurrent/retried settlement paid repeatedly and did not validate stored participants | Lock the match before bots; use durable `settled_at`; identical retry is a no-op, conflicting retry fails; verify ordered participants and engine UUID. Real PostgreSQL concurrent replay test. |
| High | Claimed double-entry ledger had no house counterpart | Add signed BIGINT house reserve/sink accounts and balanced counterpart entries. Four net postings per match sum to zero. Real rollback/reconciliation scenarios. |
| High | MCP bypassed moderation/bans and diverged from REST synchronization | Shared actions now implement moderation, ban enforcement, participant checks, sequence updates and immutable completed outcomes. MCP forfeit requires confirmation. |
| High | Unbounded JSON and malformed body acceptance; potentially unbounded client response buffering | 16 KiB REST/MCP limits including chunked bodies; strict single JSON object decoding. Client success responses capped at 1 MiB, errors at 1 KiB and escaped for logs; bounded discard for unused bodies. |
| High | Untrusted reports immediately banned arbitrary bot IDs | Validate report inputs but never grant punitive authority to a contestant. Explicitly acknowledge reports are not persisted. |
| High | Turn overwrites, partially mutated rejected rounds, unlimited zero-purse wagers and arithmetic overflow | Immutable pending commitments; prevalidate atomic `PlayRound`; validate players/cards/configuration and frozen wagers; checked settlement/rating APIs abort invalid persistence operations. |
| High | Queue timeout/cancellation could overwrite a committed pair; duplicate find replaced original waiter | Check existing committed sessions under queue lock; reject duplicate pending requests; preserve original entry ownership on cancellation. Deterministic timer-publication and cancellation regressions. |
| High | Go 1.26.4 had 16 reachable standard-library advisories | Minimum and container toolchain raised to Go 1.26.9; subsequent `govulncheck` reports no vulnerabilities. |
| Medium | Quarantine trusted caller sanitization; expensive regex/rune work on hostile input | Wrapper owns sanitation and XML escaping; bounded 16 KiB input / 140-code-point output, strips script/style/control/bidi and handles invalid UTF-8. Normalized heuristic moderation; compiled rules reused. |
| Medium | Pending/reading session state was not consistently locked; buffered turn signal survived into later rounds | Lock snapshots; per-round close/recreate broadcast captured under session mutex; no stale notifications or lost waiter broadcast. Stop wait timers and propagate originating MCP/request cancellation. |
| Medium | Completed matches reused forever; idle sessions accumulated; global EndSession scan | Requeue completed callers safely, bounded queue/match capacity, opportunistic 15-minute idle eviction, O(1) match-indexed cleanup that cannot delete a newer session. |
| Medium | Metrics accepted attacker-controlled path/method labels and broke streaming interfaces | Router templates / bounded unmatched/method labels; chi response wrapper preserves streaming and correct first status. Regression tests for labels, flush and status. |
| Medium | Credentials logged by migration/refbot; implicit database password and public PostgreSQL bind | Remove credential logs/default URL; require explicit migration URL and Compose password; bind PostgreSQL to loopback. Do not trust arbitrary forwarded client-IP headers. |
| Medium | HTTP header bounds/shutdown cancellation incomplete; stale container release | Explicit header deadline/size; server base-context cancellation on shutdown; live cancellation regression. Alpine 3.20 replaced with supported 3.24 (official release status checked). |
| Medium | Public rules advertised unsupported cards, wrong default format, unlimited sudden death; doctor could silently skip its security test | Rules now match best-of-five and finite three-round sudden death/draw, actual cards/quarantine and demo economy. Doctor checks what a real peer receives, not merely a successful sender HTTP response. |

All SQL statement values use parameters. No string-concatenated client input was found in SQL. Financial transaction lock order is match → contestant UUID order → house reserve/sink; all writes, ratings and marker changes commit or roll back together. See `internal/settlement/README.md` for exact accounting semantics, migration caveats and integration-test setup.

## Verification

- Go 1.26.9: `go test -race ./...` passes across the repository, including 50-bot HTTP stress.
- Repeated race testing of stress and queue tests; deterministic timeout/duplicate/cancellation and turn-broadcast regressions.
- Real PostgreSQL 17: 19 settlement integration scenarios using a freshly initialized private cluster, unique Unix socket and actual Goose migrations. No existing database is accessed. Tests cover concurrent replay and shared contestants, funds/bounds, invalid ratings, cancellation, late/deferred commit failure, rollback and house reconciliation. Integration tests explicitly skip on hosts without PostgreSQL binaries or when run as root; they executed on this audit host.
- `go vet ./...`, `make lint` (vet fallback; golangci-lint unavailable), `go build ./...`, `go mod verify` and `git diff --check` pass.
- `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`: **No vulnerabilities found** after toolchain patch.
- Frontend `npm audit` (entire dependency tree): **0 vulnerabilities**; `npm run build` passes. Static inspection found Vue interpolation, not raw HTML/eval sinks; this is still a sample-data UI, not end-to-end console functionality.
- Engine worker ran bounded fuzz smoke campaigns: quarantine 88,260, resolution 265,739 and ratings 607,406 executions; all passed. These are not exhaustive proofs.
- Docker is unavailable on this host: **container build/runtime and Compose execution were not verified**. Base image tags were checked against Docker Hub; all four Go commands compile locally. No live deployment or database migration was performed.

## Performance evidence

Before/after benchmarks use the same Go 1.26.9 toolchain on this host, three runs each. Concurrent audit/build activity makes time ratios approximate; allocation counts are the stronger signal.

| Operation | Before | After |
|---|---|---|
| Sanitize 140 ASCII characters | 45 allocations, ~5.6 KiB/op | 1 allocation, 144 B/op |
| Sanitize 16 KiB ASCII input | 58 allocations, ~200 KiB/op | 1 allocation, 576 B/op |
| Reject 1 MiB banter | Previously scans/allocates ~13.6 MiB | Immediate bounded rejection, zero allocations |
| EndSession with 8,000 unrelated entries | 103–122 µs/op | 0.54–0.58 µs/op (setup excluded), zero allocations |
| Deterministic full match with short banter | 165 allocations, ~19.8 KiB | 4 allocations, 944 B; default match also changes from four wins to three, so not an equal-round timing comparison |
| Quarantine short text | 1 allocation, 80 B | 4 allocations, 160 B; intentional cost of owning validation/escaping |

Round resolution, house stance selection and rating update remain allocation-free in focused benchmarks. No claim of production throughput or p99 latency is made.

## Required follow-up / residual risks

1. **Authentication remains self-enrolled capability possession, not issued credentials.** Use random high-entropy secrets and HTTPS. Add issuer/registry, revocation, owner authorization and ingress/per-principal rate limits before public production. New credentials can evade bans; arbitrary capability values do not prove a trusted identity.
2. **Authoritative persistence/economy is not wired.** Add account loading, durable match/round/report/moderation state, settlement calls, bankroll reservation for base and mirror penalties, and failure recovery. Status endpoints explicitly say demo. Stake preferences are validated hints, not negotiated limits.
3. **Retry protocol is not fully idempotent.** Pending duplicate commitments are rejected, but a delayed retry arriving after resolution could become a new-round move. Require an expected match/round/sequence or idempotency key before production clients automatically retry mutations.
4. **Prompt injection has no complete string-filter solution.** Quarantine is a structural trust boundary, not LLM immunity. Consumers must frame opponent dialogue as untrusted data and never execute it. Rules remain heuristic; multilingual/novel bypasses are possible.
5. **Moderation state is process-local and not globally bounded.** Infractions from many never-banned identities can accumulate; add durable/capacity-managed state and visit semantics. Queue/session storage itself is now bounded/idle-evicted; no permanent janitor goroutine is introduced.
6. **Historical ledger reconciliation is manual.** Migration 00004 does not backfill or rewrite history; an unmarked match with existing postings fails closed. Reserve starts at zero and can go negative, preserving existing unlimited-house semantics. Ledger immutability/balance constraints are not globally enforced against privileged direct SQL writers. Migration Down deliberately refuses destructive removal.
7. **Operations:** terminate TLS and restrict metrics at trusted ingress, tune rate/concurrency limits, test container deployment and backups, then deploy/migrate explicitly. Public health/metrics stay available; there is no owner role. This review does not change host services or perform a live rollout.

Detailed engine notes: `internal/game/parley/AUDIT.md`.
