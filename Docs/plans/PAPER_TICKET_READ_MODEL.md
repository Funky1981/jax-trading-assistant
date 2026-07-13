# Paper Ticket Read Model

## Purpose

This phase lets Jax store and review candidates that have reached the paper-ticket boundary. It is a review model only: it preserves the human-approved, paper-only trade plan after the prior candidate checks pass.

## What Persisted Paper Tickets Are

Persisted paper tickets are durable records in `candidate_paper_tickets` for candidates that passed:

- structural completeness
- evidence scoring
- trust gate review
- risk review
- approval eligibility
- explicit human approval
- paper ticket boundary validation

They capture the candidate trade plan, review statuses, risk values, source approval reference, rejection or warning reasons, and paper-only safety flags.

## What They Are Not

Persisted paper tickets are not broker orders, execution instructions, live-trading permissions, leverage permissions, or BUY/SELL/HOLD decisions made by an LLM. They do not enable automatic order placement.

## Required Prior Phase Checks

Persistence is allowed only when the candidate is structurally complete, evidence is sufficient, the gate is ready for risk review, risk is ready for approval review, approval eligibility passes, human approval is granted, and the paper ticket boundary returns `CanCreateTicket=true`.

The boundary also rejects leverage above 1.0, broker execution flags, existing execution instructions, live-trading flags, submitted or filled candidates, and trade identifiers.

## Table And Read Model Fields

The table is `candidate_paper_tickets`.

Core fields:

- `paper_ticket_id`, `candidate_id`, `created_at`, `updated_at`, `status`
- `source_approval_id`, `approval_decision_ref`
- `symbol`, `direction`, `setup_type`, `catalyst_summary`
- `entry_price`, `stop_loss_price`, `target_price`, `position_size`
- `max_normal_loss`, `max_slippage_adjusted_loss`, `reward_risk_ratio`
- `evidence_status`, `gate_status`, `risk_status`, `approval_status`
- `paper_only`, `broker_execution_allowed`, `execution_instruction_created`, `live_trading_allowed`, `leverage_allowed`
- `reject_reasons`, `warning_reasons`

The API-facing review projection is exposed as `paperTicket` on approval detail responses. It intentionally omits broker/live/leverage/execution-control fields and exposes only review-safe trade plan, risk, status, and reason fields.

## Status Meanings

- `paper_ticket_ready`: boundary result before durable creation.
- `paper_ticket_created`: persisted review record exists.
- `paper_ticket_reviewed`: operator reviewed the paper-ticket record.
- `paper_ticket_cancelled`: review record was cancelled before further paper workflow.
- `paper_ticket_blocked`: persistence or review is blocked by a safety condition.

No execution statuses are introduced in this phase.

## Safety Defaults

Persisted rows are constrained to:

- `paper_only = true`
- `broker_execution_allowed = false`
- `execution_instruction_created = false`
- `live_trading_allowed = false`
- `leverage_allowed = false`

Database check constraints enforce these defaults.

## Example Persisted Paper Ticket JSON

```json
{
  "paperTicketId": "pt_2f4b7d27-67a5-4575-85fb-463bdf0fd99b",
  "candidateId": "2f4b7d27-67a5-4575-85fb-463bdf0fd99b",
  "status": "paper_ticket_created",
  "sourceApprovalId": "99e8e985-0f55-47a2-b1b4-572c942e21a1",
  "approvalDecisionRef": "99e8e985-0f55-47a2-b1b4-572c942e21a1",
  "symbol": "SOXX",
  "direction": "long",
  "setupType": "event_driven_swing",
  "catalystSummary": "Semiconductor ETF momentum confirmed by fresh evidence.",
  "entryPrice": 240.0,
  "stopLossPrice": 236.0,
  "targetPrice": 252.0,
  "positionSize": 12.0,
  "maxNormalLoss": 48.0,
  "maxSlippageAdjustedLoss": 54.0,
  "rewardRiskRatio": 3.0,
  "evidenceStatus": "sufficient",
  "gateStatus": "ready_for_risk_review",
  "riskStatus": "ready_for_approval_review",
  "approvalStatus": "paper_ticket_ready",
  "paperOnly": true,
  "brokerExecutionAllowed": false,
  "executionInstructionCreated": false,
  "liveTradingAllowed": false,
  "leverageAllowed": false,
  "rejectReasons": [],
  "warningReasons": []
}
```

## Example Review Read Model JSON

```json
{
  "paperTicketId": "pt_2f4b7d27-67a5-4575-85fb-463bdf0fd99b",
  "candidateId": "2f4b7d27-67a5-4575-85fb-463bdf0fd99b",
  "status": "paper_ticket_created",
  "symbol": "SOXX",
  "direction": "long",
  "setupType": "event_driven_swing",
  "catalystSummary": "Semiconductor ETF momentum confirmed by fresh evidence.",
  "entryPrice": 240.0,
  "stopLossPrice": 236.0,
  "targetPrice": 252.0,
  "positionSize": 12.0,
  "maxNormalLoss": 48.0,
  "maxSlippageAdjustedLoss": 54.0,
  "rewardRiskRatio": 3.0,
  "evidenceStatus": "sufficient",
  "gateStatus": "ready_for_risk_review",
  "riskStatus": "ready_for_approval_review",
  "approvalStatus": "paper_ticket_ready",
  "paperOnly": true,
  "rejectReasons": [],
  "warningReasons": []
}
```

## Deferred Work

- Operator actions for marking paper tickets reviewed or cancelled.
- Dedicated list endpoint for paper-ticket review queues.
- UI panels that display the paper-ticket review model independently from legacy execution activity.
- Archival/reporting workflows for reviewed paper tickets.

## Remaining Legacy Execution Risks

Legacy execution-instruction tables, workers, and UI surfaces still exist. This phase does not remove them. The current approval path does not call `buildInstruction`, and persisted paper tickets are constrained to paper-only defaults, but adjacent legacy execution-chain views may still display historical execution concepts until a later cleanup phase separates paper review from execution activity more fully.
