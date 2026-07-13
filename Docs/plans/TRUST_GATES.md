# Trust Gates

## Purpose

The Trust Gates phase adds a deterministic candidate gatekeeper for deciding whether a structurally complete, evidence-scored trade candidate can move to risk review.

This phase does not approve trades and does not make anything executable.

## What The Gatekeeper Does

- Checks structured candidate completeness.
- Checks evidence status and evidence readiness.
- Blocks stale, blocked, contradictory-only, weak, mixed, or missing evidence from progressing.
- Rejects missing trade plan fields such as entry, stop loss, catalyst summary, and invalidation reason.
- Rejects leverage requests above `1.0`.
- Rejects any premature approval, execution instruction, trade id, or broker-execution flag.
- Produces a deterministic gate result with reasons and next phase guidance.

## What The Gatekeeper Does Not Do

- It does not approve candidates.
- It does not create execution instructions.
- It does not allow broker execution.
- It does not call broker, IB, execution, or order-placement code.
- It does not perform final risk sizing.
- It does not decide BUY, SELL, or HOLD.
- It does not bypass structural validation or evidence scoring.

## Gate Inputs

The first implementation evaluates:

- `Candidate` from `internal/modules/candidates`.
- `EvidenceScoreSummary` from the evidence scoring layer.
- Evaluation time.

The gate composes existing candidate fields and statuses where possible:

- symbol
- setup type
- direction
- catalyst summary
- proposed entry price
- stop loss price
- invalidation reason
- structural reject reasons
- evidence status
- evidence readiness
- contradictory evidence counts
- stale evidence counts
- risk status
- approval status
- candidate approval/execution identifiers
- metadata safety flags, including leverage and broker execution flags

## Gate Result Fields

The gate result contains:

- `candidate_id`
- `evaluated_at`
- `gate_status`
- `gate_ready`
- `hard_reject`
- `reject_reasons`
- `warning_reasons`
- `next_required_phase`
- `broker_execution_allowed`
- `execution_instruction_created`
- `approval_granted`

The final three fields are always `false` in this phase.

## Gate Statuses

- `blocked`
- `incomplete`
- `evidence_missing`
- `evidence_weak`
- `evidence_mixed`
- `evidence_stale`
- `risk_pending`
- `approval_pending`
- `ready_for_risk_review`

The existing `not_evaluated` and `ready` statuses remain for older candidate lifecycle code. The gatekeeper returns `ready_for_risk_review` for a candidate that is safe and complete enough for the next phase.

## Hard Reject Rules

The gate hard rejects when:

- the candidate is structurally incomplete
- symbol is missing
- setup type is missing
- direction is missing
- catalyst summary is missing
- proposed entry price is missing or invalid
- stop loss price is missing or invalid
- invalidation reason is missing
- evidence status is `blocked`
- evidence status is `stale`
- only contradictory evidence exists
- leverage is requested or implied above `1.0`
- broker execution is flagged as allowed
- an execution instruction already exists
- a trade id already exists
- approval was already granted before this phase should grant it

## Pending And Blocking Rules

The gate blocks progression without hard rejection when:

- evidence is missing
- evidence is weak
- evidence is mixed
- evidence is marked sufficient but not evidence-ready

Risk review remains the next required phase for candidates that pass the gate. Human approval and broker execution remain out of scope for this phase.

## Example Gate Result JSON

```json
{
  "candidateId": "00000000-0000-0000-0000-000000000001",
  "evaluatedAt": "2026-07-12T09:30:00Z",
  "gateStatus": "ready_for_risk_review",
  "gateReady": true,
  "hardReject": false,
  "warningReasons": ["risk_review_pending", "approval_not_started"],
  "nextRequiredPhase": "risk_review",
  "brokerExecutionAllowed": false,
  "executionInstructionCreated": false,
  "approvalGranted": false
}
```

## Safety Notes

- The implementation is domain-only and does not add persistence.
- No migration is required for this phase.
- Gate readiness means ready for risk review only.
- Approval, paper execution, live trading, leverage, and broker execution remain blocked by design.
- Evidence scoring still does not approve candidates or create execution instructions.

## Deferred Work

- Persist gate results if later phases need an auditable gate history.
- Add strategy-specific gates after the shared gatekeeper is stable.
- Add slippage-adjusted risk and sizing checks in the risk engine phase.
- Integrate gate results into approval queues only after risk review semantics are implemented.
- Add broader replay/golden coverage when gatekeeper output becomes part of external API responses.

## What's Left

- Wire this domain result into a later risk-review workflow.
- Decide whether gate results should be persisted in a future additive migration.
- Add API or frontend exposure only after risk review and approval boundaries are finalized.
