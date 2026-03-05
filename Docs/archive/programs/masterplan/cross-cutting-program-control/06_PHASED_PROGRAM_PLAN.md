# Phased Program Plan

## Phase 0 Baseline hardening
Stabilize work branch runtime, CI, migrations, environment matrix.

## Phase 1 No-fake-data enforcement (first)
Disable fabricated paths, add provenance, startup policy, provenance gate.

## Phase 2 Event data foundation
Event ingestion, normalization, dedupe, symbol mapping, event APIs.

## Phase 3 Intraday data + session correctness
1m/5m candles, VWAP, calendar utilities, data recon.

## Phase 4 Strategy framework + instances
Strategy types metadata, instance CRUD/validation, UI forms.

## Phase 5 IBM strategy pack
Implement event strategies (safe -> risky) with tests.

## Phase 6 Research engine completion
Real deterministic backtests, run storage, sweeps, walk-forward, reproducibility.

## Phase 7 Execution hardening
Intents/orders/fills, duplicate prevention, flatten-by-close, reconciliations.

## Phase 8 AI audit + replay
ai_decisions, acceptance logging, schema outputs, decision inspector.

## Phase 9 UI operations completion
Research/Analysis/Testing route wiring and API integration.

## Phase 10 Trust gates + shadow validation
Automated gates, proof artifacts, parity checks, incident workflow.

## Phase 11 Controlled live readiness (later)
Feature flags, caps, approvals, runbooks, pilot sign-off.
