# Paper Approval Loop

## Purpose

The paper approval loop decides whether a trade candidate is safe and complete enough to show to a human for paper approval review.

It answers one question:

```text
Can this candidate be shown to a human for paper approval review?
```

It does not decide whether a trade should execute.

## What Approval-Review Eligibility Does

Approval-review eligibility composes the prior deterministic phases:

- structured candidate completeness
- evidence scoring
- trust gate readiness
- risk review readiness
- safety flags that must remain false before human action

A candidate is approval-review eligible only when all required prior phases pass and no premature approval, execution, broker, leverage, or live-trading signal is present.

## What It Does Not Do

Approval-review eligibility does not:

- approve a candidate
- create a paper ticket by itself
- create execution instructions
- allow broker execution
- allow live trading
- allow leverage above `1.0`
- place orders automatically
- let an LLM decide BUY, SELL, or HOLD

## Approval Eligibility Inputs

The eligibility check uses:

- `Candidate` from `internal/modules/candidates`
- latest `EvidenceScoreSummary`
- latest `GateResult`
- latest `RiskReviewResult`

For persisted approval queues and approved decisions, the implementation uses existing tables and columns:

- `candidate_trades.status`
- `candidate_trades.gate_status`
- `candidate_trades.risk_status`
- `candidate_trades.approval_status`
- `candidate_trades.human_approval_required`
- latest `candidate_evidence_scores`
- absence of `execution_instructions`
- absence of an approved `candidate_approvals` record

No new persistence table is required for this phase.

## Approval Result Fields

`ApprovalEligibilityResult` returns:

- `candidate_id`
- `evaluated_at`
- `approval_eligible`
- `approval_status`
- `reject_reasons`
- `warning_reasons`
- `next_required_phase`
- `broker_execution_allowed`
- `execution_instruction_created`
- `live_trading_allowed`

The final three fields are always `false` in this phase.

## Status Meanings

- `blocked`: unsafe or premature approval/execution state was detected.
- `structure_incomplete`: required candidate fields are missing.
- `evidence_not_ready`: evidence is missing, weak, mixed, stale, blocked, or not marked ready.
- `gate_not_ready`: trust gate did not produce `ready_for_risk_review`.
- `risk_not_ready`: risk review did not produce `ready_for_approval_review`.
- `approval_review_ready`: candidate can be shown to a human for paper approval review.
- `human_approved_paper`: human approved the paper workflow boundary.
- `human_rejected`: human rejected the candidate.
- `human_snoozed`: human deferred review.

## Required Prior Phase Checks

A candidate must satisfy all of these before it can be approval-review ready:

- structurally complete
- `evidence_status = sufficient`
- `evidence_ready = true`
- `gate_status = ready_for_risk_review`
- `gate_ready = true`
- `risk_status = ready_for_approval_review`
- `risk_ready = true`
- slippage-adjusted loss is within allowed risk
- requested leverage is not above `1.0`
- `broker_execution_allowed = false`
- `execution_instruction_created = false`
- `approval_granted = false`

## Human Approval Boundaries

Human approval can only approve paper review/workflow progression.

Human approval does not imply:

- live trading allowed
- broker execution allowed
- leverage allowed
- execution instruction created by eligibility
- automatic order placement

Existing approval code may still have later paper-workflow behavior after an explicit human approval. This phase adds an eligibility boundary before that action and does not expand execution behavior.

## Example JSON

```json
{
  "candidateId": "00000000-0000-0000-0000-000000000001",
  "evaluatedAt": "2026-07-13T15:00:00Z",
  "approvalEligible": true,
  "approvalStatus": "approval_review_ready",
  "rejectReasons": [],
  "warningReasons": [],
  "nextRequiredPhase": "approval_review",
  "brokerExecutionAllowed": false,
  "executionInstructionCreated": false,
  "liveTradingAllowed": false
}
```

## Safety Notes

- Approval eligibility is deterministic.
- Approval eligibility is domain-first and does not require a new table.
- Approval queues fail closed when latest evidence scores are missing.
- Approved human decisions are blocked unless the candidate is approval-review eligible.
- Live trading, broker execution, leverage, automatic order placement, and execution-instruction creation remain outside this phase.

## Deferred Work

- Persist a first-class `candidate_approval_reviews` table only if later audit or UI workflows need durable eligibility snapshots.
- Add frontend read models once the approval-review UI contract is finalized.
- Split existing post-human-approval paper execution behavior behind a later explicit paper-ticket phase if the roadmap requires a stricter separation.
