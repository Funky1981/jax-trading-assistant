# Structured Trade Candidate Model

> **HISTORICAL / SUPPORTING IMPLEMENTATION NOTE** — Current roadmap authority is
> `Docs/ROADMAP.md`. This document records an earlier candidate-model increment;
> its implementation status does not imply current-phase acceptance.

## Status

Implemented as the next phase after the Safety Baseline.

Historical roadmap reference:

`Docs/plans/jax-trading-roadmap-pack`

This phase defines the first-class trade candidate contract only. It does not implement evidence scoring, trust gates, risk sizing, broker execution, live trading, leverage, automatic order placement, or strategy logic.

## Implemented Contract

The existing `internal/modules/candidates.Candidate` model now carries explicit fields for:

- Identity and source: candidate id, timestamps, status, source, provenance.
- Instrument context: symbol, instrument type, venue, currency.
- Setup context: setup type, direction, time horizon, strategy family, candidate reason summary.
- Catalyst context: catalyst type, summary, source, timestamp, confidence.
- Evidence summary: supporting summary, contradictory summary, source count, contradictory evidence flag.
- Trade plan: proposed entry price, stop loss, target, invalidation reason, expected reward/risk ratio.
- Risk placeholders: slippage allowance, max normal loss, max slippage-adjusted loss, position size, risk status.
- Approval readiness: human approval required, approval status, reject reasons, gate status.
- Audit references: model version, generator version, raw source ref, source payload ref, decision log ref.

Existing fields are reused where appropriate:

- `EntryPrice` remains the proposed entry price.
- `StopLoss` remains the stop-loss price.
- `TakeProfit` remains the target price.
- `SignalType` remains available for legacy producers, while `Direction` is first-class for the new contract.
- `Reasoning` remains available, while `CandidateReasonSummary` is the concise review-facing summary.

## Structural Validation

`ValidateStructuralCompleteness` marks a candidate structurally incomplete unless all of these are present:

- `symbol`
- `setup_type`
- `direction`
- `catalyst_summary`
- `proposed_entry_price`
- `stop_loss_price`
- `invalidation_reason`

Contradictory evidence is allowed to exist in the model, but a candidate with contradictory evidence is not gate-ready.

Risk fields may remain unset during this phase. When risk is unset, the candidate reports:

`risk_status = pending`

## Safety Boundary

Structured candidate validation is not an execution authorization.

The validation result explicitly keeps:

`brokerExecutionAllowed = false`

Candidate qualification now blocks candidates that are not gate-ready before they can move to `awaiting_approval`.

No broker execution paths were added or expanded.

## Persistence

Migration added:

- `db/postgres/migrations/000043_structured_trade_candidate_model.up.sql`
- `db/postgres/migrations/000043_structured_trade_candidate_model.down.sql`

The migration is additive and defaulted for existing rows.

## Tests

Added tests cover:

- Core structural fields are required.
- Missing stop loss is incomplete.
- Missing catalyst is incomplete.
- Contradictory evidence is not gate-ready.
- Risk fields can be unset while risk status remains pending.
- Structural validation does not allow broker execution.
- The migration contains the expected structured candidate columns.

## Next Phase

The next roadmap phase is evidence scoring.

Do not proceed to scoring until this contract is accepted and the candidate producers are reviewed for which structured fields they can safely populate from real data.
