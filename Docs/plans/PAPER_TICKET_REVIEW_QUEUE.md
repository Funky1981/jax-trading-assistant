# Paper Ticket Review Queue

## Purpose

Expose persisted `candidate_paper_tickets` as a safe paper-only review queue and allow boring review actions that do not create order intent.

## What The Review Queue Does

- Lists persisted paper tickets for operator review.
- Shows only the review-safe trade plan, status, risk, evidence, and reason fields.
- Lets an operator mark a paper ticket reviewed.
- Lets an operator cancel a paper ticket.
- Lets an operator attach an internal review note without adding it to the public review projection.

## What It Does Not Do

- It does not submit orders.
- It does not create execution instructions.
- It does not permit broker routing.
- It does not permit live trading.
- It does not permit leverage.
- It does not turn position size into an executable order.
- It does not let an LLM decide BUY, SELL, or HOLD.

## Safe Read Model Fields

- `paper_ticket_id`
- `candidate_id`
- `created_at`
- `updated_at`
- `status`
- `symbol`
- `direction`
- `setup_type`
- `catalyst_summary`
- `entry_price`
- `stop_loss_price`
- `target_price`
- `position_size`
- `max_normal_loss`
- `max_slippage_adjusted_loss`
- `reward_risk_ratio`
- `evidence_status`
- `gate_status`
- `risk_status`
- `approval_status`
- `paper_only`
- `reject_reasons`
- `warning_reasons`

## Forbidden Execution Fields And Actions

Forbidden fields in API/UI review projections:

- `broker_execution_allowed`
- `execution_instruction_created`
- `live_trading_allowed`
- `leverage_allowed`
- `execution_ready`
- `auto_execution_enabled`

Forbidden actions:

- broker routing
- order placement
- live-mode enablement
- leverage enablement
- execution instruction creation
- automatic order placement

## Allowed Review Actions

- `mark_reviewed`
- `cancel`
- `add_note`

`add_note` writes internal `review_notes`; the safe review DTO still omits notes and execution-control fields.

## Status Transitions

- `paper_ticket_created` -> `paper_ticket_reviewed`
- `paper_ticket_ready` -> `paper_ticket_reviewed`
- `paper_ticket_created` -> `paper_ticket_cancelled`
- `paper_ticket_ready` -> `paper_ticket_cancelled`

Cancelled tickets do not transition into reviewed or executable states. No execution statuses are added.

## Example Queue Item JSON

```json
{
  "paperTicketId": "pt_2f4b7d27-67a5-4575-85fb-463bdf0fd99b",
  "candidateId": "2f4b7d27-67a5-4575-85fb-463bdf0fd99b",
  "createdAt": "2026-07-13T08:00:00Z",
  "updatedAt": "2026-07-13T08:05:00Z",
  "status": "paper_ticket_created",
  "symbol": "SPY",
  "direction": "long",
  "setupType": "pullback_continuation",
  "catalystSummary": "Broad-market ETF holding above support.",
  "entryPrice": 101.25,
  "stopLossPrice": 98.75,
  "targetPrice": 107.0,
  "positionSize": 4.0,
  "maxNormalLoss": 10.0,
  "maxSlippageAdjustedLoss": 12.0,
  "rewardRiskRatio": 2.3,
  "evidenceStatus": "sufficient",
  "gateStatus": "ready_for_risk_review",
  "riskStatus": "ready_for_approval_review",
  "approvalStatus": "paper_ticket_ready",
  "paperOnly": true,
  "rejectReasons": [],
  "warningReasons": []
}
```

## Example Review Action JSON

```json
{
  "note": "reviewed by operator"
}
```

POST paths:

- `/api/v1/paper-tickets/{paperTicketId}/mark-reviewed`
- `/api/v1/paper-tickets/{paperTicketId}/cancel`
- `/api/v1/paper-tickets/{paperTicketId}/notes`

## Safety Notes

The database continues to enforce:

- `paper_only = true`
- `broker_execution_allowed = false`
- `execution_instruction_created = false`
- `live_trading_allowed = false`
- `leverage_allowed = false`

Review actions update only review status, `updated_at`, and internal review notes. They do not insert into `execution_instructions`.

## Deferred Work

- Reopen review can be added later if there is a clear operator workflow.
- Dedicated filtering by status or symbol can be added when the queue grows.
- Separate reporting/export views can be added after paper review usage stabilizes.

## Remaining Legacy Execution Risks

Legacy execution-instruction tables, workers, and UI surfaces still exist elsewhere in the repo. This phase does not remove them. The review queue uses dedicated paper-ticket endpoints and safe DTOs, but adjacent legacy pages may still show historical execution-chain concepts until a later cleanup phase separates paper review from legacy execution surfaces more fully.
