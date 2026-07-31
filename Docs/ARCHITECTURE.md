# Architecture

The active platform is an ADR-0012 modular-monolith topology with two Go runtimes and explicit external service boundaries.

## Runtime Topology

- `cmd/trader`
  - Deterministic runtime and execution path.
  - Frontend-facing API on `8081`.
  - Runtime server on `8100`.
- `cmd/research`
  - Orchestration, research/backtest, memory tool paths.
  - HTTP port `8091`.
- External boundaries
  - `services/ib-bridge` on `8092`
  - `services/agent0-service` on `8093`

## Repository Layout (Current)

```text
cmd/
  trader/
  research/
  artifact-approver/
  shadow-validator/
internal/
  modules/, domain/, integrations/, providers/
libs/
  auth/, marketdata/, utcp/, agent0/, dexter/, ...
services/
  ib-bridge/, agent0-service/
frontend/
db/postgres/migrations/
scripts/
```

## Guardrails

- Trader must stay deterministic and avoid research-only imports.
- Artifact loading/promotion must remain approval-state driven.
- Trust-gate evidence (Gate2 replay + Gate3 promotion) is required for validation transitions.
- External Python services remain explicit boundaries unless changed by ADR.

## Genuine event decision boundary

`cmd/event-decision-replay` is a bounded operator CLI over `internal/modules/eventdecisions`. It reads existing World Monitor inbox, normalized-event, raw-provenance, symbol-mapping, candidate, evidence, trust, and risk state. The deterministic evaluator produces a structured `NO_TRADE`, `WATCH`, or `CANDIDATE` result under an explicit versioned ruleset.

Schema version 53 adds an immutable initial-decision contract and append-only deterministic asset-resolution provenance. `genuine-event-decision-v2` keeps later projections separate from the evaluation label, while `event-asset-resolution-v1` reuses `event_symbol_map` for accepted mappings without forcing unresolved or ambiguous events to an asset. Historical origin and temporal availability are first-class fields so the evaluator cannot mistake replay delay or a later mapping for live event-time state.

`cmd/evidence-quality-evaluation` is an isolated retrospective reader over `internal/modules/evidencequality`. It evaluates current genuine event decisions against existing provenance-qualified candles inside a repeatable-read, read-only transaction. Versioned population, mapping, benchmark, timestamp, and no-look-ahead rules produce Markdown, JSON, and CSV artefacts without modifying decisions, projections, candidates, approvals, or execution-side state.

Migrations 49 and 50 persist append-only input and ruleset versions in `genuine_event_decisions` with a unique replay identity and exactly one current projection per source event. Persisted batches use a serializable transaction. The module never writes candidates or approval/execution-side records; a `CANDIDATE` result may only link an already-complete safe candidate. API handlers expose the persisted projection and history as read models for Home and Evidence Inbox.

## Verification Baseline

- Go code changes: `scripts/go-verify.ps1 -Mode quick|standard|full`.
- Golden/replay-sensitive changes: `scripts/golden-check.ps1 -Mode verify`.
- Knowledge ingest flow: `scripts/knowledge-cycle.ps1 -Mode all` (dry-run first).
