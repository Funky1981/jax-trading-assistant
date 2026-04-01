# Phase 0 — Truth Path Hardening

## Objective
Make the research and paper-trading paths trustworthy before more features are added.

## Problems this phase solves
- fake/synthetic paths can still contaminate research conclusions
- provenance is not enforced as a hard platform rule
- deterministic replay is not yet a sign-off rule
- artifact promotion is stronger than execution truth, which is backwards

## Required deliverables

### 0.1 No-fake-data enforcement
Implement:
- runtime mode policy:
  - `dev`
  - `test`
  - `research`
  - `paper`
  - `live`
- startup validation that blocks fake providers in `research`, `paper`, `live`
- explicit kill switch if any synthetic provider is active in restricted modes

Files to add or modify:
- `cmd/trader/main.go`
- `cmd/research/main.go`
- `config/providers.json`
- provider registry or mode validation layer
- new `internal/modules/platform/mode_policy.go`

### 0.2 Provenance model
Add provenance to:
- research runs
- artifacts
- datasets
- candidate trades
- paper executions

Minimum fields:
- `source_provider`
- `data_source_type`
- `dataset_id`
- `dataset_hash`
- `is_synthetic`
- `synthetic_reason`
- `provenance_verified_at`

### 0.3 Deterministic replay enforcement
Backtest/research runs must store:
- config snapshot
- dataset reference
- determinism seed
- code/runtime version
- artifact hash linkage where relevant

### 0.4 Gate creation
Create first gates:
- Gate 0: config/schema integrity
- Gate 1: data/provenance integrity
- Gate 2: deterministic replay reproducibility
