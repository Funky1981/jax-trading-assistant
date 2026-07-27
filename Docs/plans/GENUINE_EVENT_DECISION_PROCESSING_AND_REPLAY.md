# Genuine Event Decision Processing and Replay

## Scope and safety boundary

This phase deterministically processes already-persisted, provenance-confirmed World Monitor events into one current `NO_TRADE`, `WATCH`, or `CANDIDATE` decision per event and ruleset. It does not call AI, collect news, schedule work, create candidates, approve candidates, create paper tickets, or write to execution, order, trade, or fill records.

The operator command fails closed unless all six runtime facts are explicit: paper mode, live trading off, execution off, execution-instruction worker off, broker execution off, and maximum leverage no greater than `1x`.

## Persisted contract

Migration 49 adds `genuine_event_decisions`; migration 50 hardens the projection to exactly one current decision per source event across ruleset versions. Important decision facts are structured columns: source and normalized identities, decision and version, ruleset and processor identities, processing mode, distinct publication/collection/receipt/decision timestamps, provenance, evidence and confidence, assets and explicit unknown-asset state, reasons, blockers, missing evidence, trust and risk states, optional candidate linkage, replay identity, input fingerprint, current projection, and audit timestamps.

The table is append-only by input version with exactly one current projection per event. A materially different persisted input or ruleset produces a new decision version and retires the prior projection without deleting history. An identical input/ruleset reuses its unique replay identity. Database constraints allow a candidate ID only on `CANDIDATE` and require one there.

## Ruleset: genuine-event-decision-v1

Configuration: `config/genuine-event-decision-v1.json`.

- Processor: `jax-genuine-event-decision-processor`.
- Mode: deterministic.
- Material severities: `medium`, `high`, and `critical`.
- Minimum persisted confidence for `WATCH`: `0.50`, matching the existing deterministic ingestion boundary.
- Minimum persisted candidate evidence score: `0.60`, matching the existing candidate evidence gate.
- Candidate product: catalog-allowed ETF only.
- Maximum leverage: `1x`.

`NO_TRADE` is the safe default. An immaterial event, confidence below the watch boundary, an unsafe product, leverage above `1x`, or a blocked/expired/rejected legacy candidate finishes as `NO_TRADE`.

A material event at or above the watch boundary finishes as `WATCH` when a complete candidate contract is absent. Unknown assets are explicitly persisted and may be watched, but can never create or link a candidate. Reasons, missing evidence, and blockers say what prevented a stronger decision.

`CANDIDATE` is possible only when an already-linked persisted candidate has a truthful mapped ETF, all required structured fields, sufficient persisted evidence, ready trust and risk states, an active lifecycle, no execution-side linkage, and safe leverage. This phase reuses that candidate; it never invokes a candidate writer or fabricates trade levels.

These thresholds are conservative system rules, not validated strategy parameters.

## Operator replay

Build or run `./cmd/event-decision-replay` with an explicit ruleset:

```powershell
go run ./cmd/event-decision-replay --ruleset genuine-event-decision-v1 --limit 30 --dry-run
go run ./cmd/event-decision-replay --ruleset genuine-event-decision-v1 --event <inbox-uuid-or-source-event-id> --limit 1
go run ./cmd/event-decision-replay --ruleset genuine-event-decision-v1 --limit 30
```

The command accepts a bounded limit from 1 through 250. It reads `DATABASE_URL` unless `--database-url` is supplied. Dry-run evaluates and returns the complete summary without opening a write transaction. Persisted batch writes use one serializable transaction; any evaluation or persistence failure rolls the whole selected batch back.

## Runtime proof: 2026-07-27

The first proof used only existing Jax data. No World Monitor process was run or modified, and no new news was collected.

Pre-replay population:

- 30 inbox rows: 25 eligible genuine, 2 explicit synthetic, and 3 rejected/missing linked provenance.
- 28 raw events and 28 normalized events.
- 20 inbox rows linked to candidates; only four candidate records were structurally complete and only three were evidence/trust/risk ready. The historical approved QQQ paper proof was explicitly synthetic and excluded. The remaining legacy candidate compatible with the rules was expired and therefore not linkable.
- 24 rows had persisted affected assets and 6 explicitly had unknown assets.
- Six genuine RSS rows had deterministic analysis identity `jax-live-ingestion-keywords-v1`. No persisted row claimed an AI analysis provider or model.
- No genuine-event decision rows existed.

Safety baseline was paper mode, live trading false, execution false, execution worker false, broker execution false, and maximum leverage `1`. Selected historical QQQ execution counts were all zero.

Proof results:

| Step | Eligible | NO_TRADE | WATCH | CANDIDATE | Excluded | Errors | Decisions created | Decisions reused | Candidates created | Candidates reused |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Dry-run | 25 | 18 | 7 | 0 | 5 | 0 | 0 | 0 | 0 | 0 |
| Persisted replay | 25 | 18 | 7 | 0 | 5 | 0 | 25 | 0 | 0 | 0 |
| Identical second replay | 25 | 18 | 7 | 0 | 5 | 0 | 0 | 25 | 0 | 0 |

After replay there were 25 current decisions over 25 distinct events, no duplicate replay identities, and no invalid candidate links. The excluded rows were the two controlled synthetic records and three rejected records without complete provenance.

Before/after safety counts were unchanged except for the new decisions:

| Record type | Before | After |
|---|---:|---:|
| Genuine-event decisions | 0 | 25 |
| Candidates | 21 | 21 |
| Approvals | 2 | 2 |
| Paper tickets | 1 | 1 |
| Execution instructions | 1 | 1 |
| Order intents | 0 | 0 |
| Derived broker orders | 0 | 0 |
| Trades | 0 | 0 |
| Fills | 0 | 0 |

The authenticated runtime API and built frontend displayed the persisted `18 / 7 / 0 / 0` Home and Evidence Inbox counts. Automated persisted-data browser checks inspected Home, one `WATCH`, one `NO_TRADE`, and System Safety at 320, 768, and 1280 pixels with no primary overflow or mutation controls. There was no natural `CANDIDATE`, so none was fabricated.

## Statistical caution

This replay proves system behaviour only. It does not validate decision quality, thresholds, natural candidate quality, strategy profitability, statistical significance, Shadow Mode, AI analysis, or broker execution. Zero candidates is a valid conservative result, not a failure.
