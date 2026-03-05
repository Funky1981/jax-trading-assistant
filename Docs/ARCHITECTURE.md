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
  - `services/hindsight` on `8888`

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
  ib-bridge/, agent0-service/, hindsight/
frontend/
db/postgres/migrations/
scripts/
```

## Guardrails

- Trader must stay deterministic and avoid research-only imports.
- Artifact loading/promotion must remain approval-state driven.
- Trust-gate evidence (Gate2 replay + Gate3 promotion) is required for validation transitions.
- External Python services remain explicit boundaries unless changed by ADR.

## Verification Baseline

- Go code changes: `scripts/go-verify.ps1 -Mode quick|standard|full`.
- Golden/replay-sensitive changes: `scripts/golden-check.ps1 -Mode verify`.
- Knowledge ingest flow: `scripts/knowledge-cycle.ps1 -Mode all` (dry-run first).
