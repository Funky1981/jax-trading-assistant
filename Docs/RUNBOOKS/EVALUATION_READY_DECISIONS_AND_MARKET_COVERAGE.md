# Evaluation-ready decisions, asset resolution, and market coverage

This runbook covers the evaluation-only path introduced by database schema version 53. It does not approve candidates, create execution instructions, submit orders, or relax any candidate threshold.

## Durable decision contract

`genuine-event-decision-v2` writes one immutable initial decision for each World Monitor inbox event and ruleset. The row records its origin as `live_origin`, `historical_backfill`, or `historical_replay`. Re-evaluation may update the separate current projection, but it cannot rewrite or delete a genuine initial label. Historical backfills are always disclosed and must not be presented as event-time live latency.

The continuous pull worker resolves and decides in the same database transaction as inbox persistence. Its origin is `live_origin`. The bounded replay CLI uses `historical_backfill` and reuses an existing initial v2 decision on identical reruns.

## Deterministic asset resolution

`event-asset-resolution-v1` applies this precedence:

1. one trusted structured symbol;
2. an exact, date-valid issuer alias in the headline;
3. one explicitly declared category or official-source proxy;
4. ambiguous or unresolved.

Multiple entities remain ambiguous. Unmatched events remain unresolved. There is no broad-market fallback and no AI mapping. Each result is append-only and records source fields, source values, rule version, relationship, confidence class, effective dates, decision origin, temporal-availability flags, and a deterministic fingerprint. Accepted results are also inserted through the existing `event_symbol_map` model.

## Operator workflow

Keep the runtime in paper mode with live trading, execution, the execution worker, and broker execution disabled, and leverage capped at 1x.

```powershell
go run ./cmd/event-decision-replay --ruleset genuine-event-decision-v2 --limit 250 --dry-run
go run ./cmd/event-decision-replay --ruleset genuine-event-decision-v2 --limit 250
.\scripts\collect-evaluation-market-coverage.ps1
.\scripts\evaluate-evidence-quality.ps1 -RulesetFile config/historical-evidence-quality-v2.json -OutputDirectory .runtime/evidence-quality-v2
```

The collector obtains its bounded symbol set only from accepted v2 resolutions and their predeclared benchmarks. It permits at most 25 symbols, requests only 1h and 1d history, uses two attempts with a 45-second timeout per attempt, rejects synthetic/test providers, deduplicates timestamps, and records timestamp semantics, provider, UTC normalization, regular-hours knowledge, classification, and adjustment state. `unknown` is retained when the provider abstraction cannot prove an adjustment choice.

## Evaluator rules

`historical-evidence-quality-v2` selects immutable initial `historical_backfill` decisions only. It excludes later projections, test/proof records, invalid time order, duplicates, and events beyond candle coverage. A live asset mapping created after its initial decision is rejected; a disclosed backfill mapping is accepted only if its inputs were knowable at the receipt anchor. Outcome windows use the first valid persisted candle at or after the anchor and never a preceding candle.

## 2026-07-31 proof

- First replay: 220 selected, 215 eligible, 215 decisions created, 146 `NO_TRADE`, 69 `WATCH`, zero `CANDIDATE`, zero failures.
- Identical replay: zero decisions created and all 215 reused.
- Mapping review: 81 resolved, seven ambiguous, 127 unresolved before deterministic exclusions; no forced fallback.
- Bounded market collection: 21 symbols including benchmarks, 42 successful timeframe requests, zero errors, 1,246 genuine candles persisted.
- Evaluation: 190 included events, 60 mapped, 130 unresolved; 60 `WATCH`, 130 `NO_TRADE`; both 1h and 1d comparisons met the five-per-group gate, while 1w did not.
- Two final evaluator runs had input fingerprint `41cce439f9a267e47b11876a52e2ab30f9b21937aa18801f6e78afbe49ff0d4f` and byte-identical Markdown, JSON, population CSV, and outcomes CSV artefacts.
- Product recommendation: `REFINE SPECIFIC RULES`.
- Final verdict: `PASS WITH LIMITATIONS`.

The principal limitation is that all evaluated v2 labels are historical backfills, not live-origin event-time decisions. The result is evaluation-pipeline evidence, not profitability evidence. Most included events also remain conservatively unresolved, and 1w coverage is insufficient.
