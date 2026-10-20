# CORTEX #492 — engine/conduct/rating audit (2026-10-20)

Write scope: `internal/game/parley/**`, `internal/conduct/**`, `internal/rating/**` only. No commits, pushes, or CORTEX changes.

## Fixes

- Commitments are immutable until resolution. Unknown players, invalid cards/configurations, negative/excessive wagers, and wager changes after round one fail before mutation. `PlayRound` prevalidates both players and rejects mixed pending commitments.
- Default rules now match `BOTTAGON_SPEC.md`/README: first to 3 points, 5 regulation rounds. Custom best-of-seven tests remain explicit. Sudden death is bounded to 3 rounds, then a draw, following source spec §7.7 and finite-match invariant §17. Custom regulation is limited to 1000 rounds, target to 2000 points. Constructor preallocation is capped independently of caller-provided round count.
- Zero purse is no longer interpreted as unlimited funds. Frozen round-one wagers can be omitted (zero) or repeated on later requests, but not changed.
- Settlement arithmetic cannot wrap debits into credits. `Match.Settle` uses new checked arithmetic and rejects invalid winners/negative inputs/overflow. Legacy `SettlePassLine` clamps negative inputs and saturates integer extremes; it is not the ledger-writing API.
- Banter sanitization has bounded input/output work, entity decoding, markup/script/style removal, whitespace folding, invalid UTF-8/control/bidi filtering, and no per-call regexp compilation or full-input rune slice. Input above 16 KiB is muted. Output remains at most 140 Unicode code points.
- `QuarantineTag` sanitizes and XML-escapes its own input rather than trusting the caller. Raw, encoded, malformed, and repeated wrapper-breaking inputs remain a single untrusted text node.
- Moderation compiles rules once, normalizes HTML entities/NFKC/whitespace/invisible controls, checks role markers before tag removal and split tokens after removal, and filters email/phone/handle contact details. Approved output is sanitized. Clean requests no longer serialize behind the infraction mutex. Counters cannot overflow; expired bans reset their infraction window on the next offense.
- Glicko-2 has finite/range/score validation, bounded root searches, logistic endpoint protection, and checked errors. `UpdateChecked` is the persistence-facing API. Legacy `Update` preserves a finite valid player on invalid batches and repairs corrupt player fields to defaults. The Glickman worked example remains passing.

## Verification

Failing adversarial regression tests were run before fixes. Final owned-package race/coverage checks:

```
go test -race -count=1 -cover ./internal/game/parley ./internal/conduct ./internal/rating/...
parley: 85.9%; conduct: 100.0%; glicko2: 94.5%; all PASS
go vet ./internal/game/parley ./internal/conduct ./internal/rating/...
PASS
```

Fuzz runs passed: quarantine 88,260 executions / 8-second budget; round resolution 265,739 / 5 seconds; checked/legacy finite ratings 607,406 / 8 seconds. Two workers each. These are bounded smoke campaigns, not exhaustive proofs.

Full repository integration suite also passed on the integrated Go 1.26.9 toolchain:

```
go test -p 2 -timeout 60s ./...
PASS (all packages, including doctor, HTTP, queue, settlement, and stress tests)
```

Benchmarks on this host (no race instrumentation; concurrent audit/build activity makes timings noisy; allocations are the more useful signal):

| Operation | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| Sanitize 140 ASCII chars | 1921 | 144 | 1 |
| Sanitize 16 KiB ASCII (140-char output) | 2119 | 576 | 1 |
| Reject 1 MiB banter | 1.94 | 0 | 0 |
| Round resolution | 22.2 | 0 | 0 |
| Three-round match, one short banter per round | 1125 | 944 | 4 |
| Quarantine short text including ampersand | 522 | 160 | 4 |
| Approved moderation, parallel | 20005–42519 | 64 | 2 |
| Glicko-2 one-result update | 400 | 0 | 0 |
| House stance + banter selection | 8.9–14.4 | 0 | 0 |

## Integration obligations / residual risks

- **Supply authoritative `P1Purse`/`P2Purse` before round-one wager validation.** The queue owner has now populated both using explicitly named `DemoPurse`; production still needs authoritative balances. Economy must separately reserve/debit base loss and mirror penalties; validating wager alone cannot guarantee sufficient final funds.
- **Use `UpdateChecked` in persistence callers** to propagate bad persisted rating/batch errors rather than silently taking the legacy finite fallback. The settlement owner has now adopted it for both players; other legacy callers still compile against `Update`.
- `Match` is intentionally not concurrency safe; session owners must serialize reads/writes and own sequence-ID idempotency. Exported fields and JSON snapshots are not a persistence format for pending private commitments.
- Rule moderation is heuristic, not proof against multilingual/novel semantic prompt injection. Recipient system instructions must explicitly treat quarantined dialogue as untrusted flavor, never game commands. Structural isolation does not make an LLM immune to persuasion.
- Conduct state remains process-local; no persistence, per-visit reset hook, or capacity/TTL eviction for never-banned identities is added here. Wider honor/report/ban enforcement is outside this scope.
- Current summary spec does not state a sudden-death cap; the 3-round draw fallback is from the detailed source spec. Parent should align outward-facing rules with that finite-match choice.
- Initial full-suite run found doctor expecting `<opponent_banter>` instead of documented `<opponent_banter untrusted="true">`; its concurrent owner corrected that expectation. The final bounded-parallelism full-suite rerun passed. An earlier rerun hit its 120-second process limit during concurrent toolchain rebuilds; this was not an owned-package test failure.

## Cross-cutting lesson

Security wrappers must own escaping even when callers promise sanitized input. Keep backward-compatible total math helpers separate from checked persistence APIs; ledger/rating writers must propagate checked failures rather than persist clamped or repaired results.

Tooling note: `rg` is absent on this host; use `grep`/`find` rather than assuming ripgrep availability.
