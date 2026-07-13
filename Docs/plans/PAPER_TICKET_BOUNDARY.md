# Paper Ticket Boundary

## Purpose

This phase separates human approval from execution-instruction creation.

After a candidate passes structure validation, evidence scoring, gate review, risk review, approval eligibility, and a human approval decision, Jax may mark the candidate as ready for a paper ticket. This is a paper-only record boundary for later paper workflow. It is not broker execution.

## Why Human Approval Is Separate From Execution Instructions

Human approval confirms that an operator accepts an eligible candidate for paper workflow review. It must not create an execution instruction, broker order, live order, or automatic placement request.

Keeping approval separate from execution prevents an approval button, mobile approval callback, or approval API from becoming an order-entry path.

## What Paper Ticket Creation Does

Paper ticket creation produces a deterministic paper-ticket-ready result with copied candidate trade-plan and risk fields:

- `paper_ticket_id`
- `candidate_id`
- `created_at`
- `status`
- `source_approval_id`
- `approval_decision_ref`
- `entry_price`
- `stop_loss_price`
- `target_price`
- `position_size`
- `max_normal_loss`
- `max_slippage_adjusted_loss`
- `reward_risk_ratio`
- `paper_only`
- `broker_execution_allowed`
- `execution_instruction_created`
- `live_trading_allowed`
- `leverage_allowed`

The current implementation is domain-only and marks approved candidates with `approval_status = "paper_ticket_ready"`. It does not add a new persistence table.

## What It Does Not Do

This boundary does not:

- create broker execution instructions
- place orders
- call IB or any broker adapter
- enable live trading
- allow leverage above `1.0`
- bypass structure validation, evidence scoring, trust gates, risk review, or approval eligibility
- allow an LLM to choose BUY/SELL/HOLD

## Required Prior Phase Checks

Paper ticket creation requires:

- structurally complete candidate
- sufficient evidence
- gate ready
- risk ready
- approval eligibility passed
- human approval granted
- no leverage above `1.0`
- broker execution not allowed
- no execution instruction already created
- live trading disabled

## Status Meanings

- `blocked`: required safety checks failed.
- `approval_not_ready`: approval eligibility or prior phase readiness has not passed.
- `approval_required`: human approval has not been granted.
- `paper_ticket_ready`: a human-approved eligible candidate is ready for later paper-ticket workflow.
- `paper_ticket_created`: deferred persistence status for a future phase.
- `paper_ticket_cancelled`: deferred cancellation status for a future phase.

## Safety Flags

Paper-ticket results default to:

```json
{
  "paper_only": true,
  "broker_execution_allowed": false,
  "execution_instruction_created": false,
  "live_trading_allowed": false,
  "leverage_allowed": false
}
```

## Example JSON

```json
{
  "paper_ticket_id": "pt_6f8a7b0e-39f0-4cb2-b7ec-5beffb3d1f40",
  "candidate_id": "6f8a7b0e-39f0-4cb2-b7ec-5beffb3d1f40",
  "created_at": "2026-07-13T15:00:00Z",
  "status": "paper_ticket_ready",
  "source_approval_id": "b49f1440-1f2c-4f1f-ae89-8559fbf7b2fb",
  "approval_decision_ref": "approval-paper-ready",
  "entry_price": 100,
  "stop_loss_price": 96,
  "target_price": 108,
  "position_size": 25,
  "max_normal_loss": 100,
  "max_slippage_adjusted_loss": 125,
  "reward_risk_ratio": 2,
  "paper_only": true,
  "broker_execution_allowed": false,
  "execution_instruction_created": false,
  "live_trading_allowed": false,
  "leverage_allowed": false
}
```

## Deferred Work

- Add a persisted `candidate_paper_tickets` table only if a future phase needs durable paper-ticket records.
- Add read models or API surfaces for paper-ticket review.
- Define later-phase paper execution workflow separately from human approval.
- Decide whether legacy execution-instruction rows remain supported for archived paper workflows.

## Remaining Legacy Execution Risks

Broker-capable and execution-instruction code still exists in legacy paths, including `internal/modules/approvals.Store.CreateExecutionInstruction`, the legacy `approvals.Service.buildInstruction` helper, `internal/modules/execution`, and the execution-instruction worker. The new roadmap approval path does not call those from human approval.
