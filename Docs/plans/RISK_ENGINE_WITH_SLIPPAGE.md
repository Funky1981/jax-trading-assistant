# Risk Engine With Slippage

## Purpose

This phase adds deterministic candidate risk review after structural validation, evidence scoring, and trust gate review.

It answers: given a complete, evidence-scored, gate-ready candidate, what position size keeps slippage-adjusted loss within the allowed per-trade risk?

## What Risk Review Does

- Requires the trust gate to be `ready_for_risk_review`.
- Validates entry, stop loss, target, direction, slippage, and no-leverage constraints.
- Calculates position size from slippage-adjusted stop distance.
- Calculates normal max loss and slippage-adjusted max loss.
- Produces a risk status and next required phase.

## What Risk Review Does Not Do

- It does not approve trades.
- It does not create execution instructions.
- It does not place broker orders.
- It does not allow broker execution.
- It does not enable live trading.
- It does not change the approval flow.

## Risk Inputs

- `candidate_id`
- `account_equity`
- `max_risk_percent_per_trade`
- `proposed_entry_price`
- `stop_loss_price`
- `target_price`
- `slippage_allowance`
- `direction`
- `max_leverage`
- `requested_leverage`
- `gate_status`
- `gate_ready`
- candidate structural completeness
- approval and execution safety flags

## Risk Result Fields

- `candidate_id`
- `evaluated_at`
- `risk_status`
- `risk_ready`
- `position_size`
- `entry_price`
- `stop_loss_price`
- `target_price`
- `stop_distance`
- `slippage_allowance`
- `slippage_adjusted_stop_distance`
- `max_normal_loss`
- `max_slippage_adjusted_loss`
- `reward_amount`
- `reward_risk_ratio`
- `account_equity`
- `max_risk_percent`
- `max_allowed_loss`
- `reject_reasons`
- `warning_reasons`
- `next_required_phase`
- `broker_execution_allowed`
- `execution_instruction_created`
- `approval_granted`

`broker_execution_allowed`, `execution_instruction_created`, and `approval_granted` remain false in this phase.

## Sizing Formula

For long candidates:

```text
stop_distance = entry_price - stop_loss_price
```

For short candidates:

```text
stop_distance = stop_loss_price - entry_price
```

The position size uses slippage-adjusted risk:

```text
slippage_adjusted_stop_distance = stop_distance + slippage_allowance
max_allowed_loss = account_equity * max_risk_percent_per_trade
position_size = max_allowed_loss / slippage_adjusted_stop_distance
max_normal_loss = position_size * stop_distance
max_slippage_adjusted_loss = position_size * slippage_adjusted_stop_distance
```

## Long Example

```text
account_equity = 10000
max_risk_percent_per_trade = 0.01
entry = 100
stop = 98
target = 106
slippage_allowance = 0.50

stop_distance = 2.00
slippage_adjusted_stop_distance = 2.50
max_allowed_loss = 100
position_size = 40
max_normal_loss = 80
max_slippage_adjusted_loss = 100
reward_amount = 240
reward_risk_ratio = 2.4
```

## Short Example

```text
account_equity = 10000
max_risk_percent_per_trade = 0.01
entry = 100
stop = 102
target = 94
slippage_allowance = 0.50

stop_distance = 2.00
slippage_adjusted_stop_distance = 2.50
max_allowed_loss = 100
position_size = 40
max_normal_loss = 80
max_slippage_adjusted_loss = 100
reward_amount = 240
reward_risk_ratio = 2.4
```

## Slippage Example

With the same `10000` account and `1%` risk:

```text
entry = 100
stop = 98
slippage_allowance = 0.10
position_size = 100 / 2.10 = 47.619048

entry = 100
stop = 98
slippage_allowance = 1.00
position_size = 100 / 3.00 = 33.333333
```

Higher slippage reduces position size because each unit carries more expected loss.

## Status Meanings

- `blocked`: safety or phase boundary blocked review.
- `gate_not_ready`: trust gate did not produce `ready_for_risk_review`.
- `invalid_trade_plan`: entry, target, or reward direction is invalid.
- `invalid_stop`: stop loss is missing or on the wrong side of entry.
- `invalid_slippage`: slippage allowance is negative or nonsensical.
- `risk_too_high`: calculated slippage-adjusted loss exceeds allowed loss.
- `reward_risk_too_low`: candidate is blocked from approval readiness because reward/risk is below the configured minimum.
- `leverage_blocked`: configured or requested leverage is above `1.0`.
- `ready_for_approval_review`: risk sizing is complete and the candidate may move to human approval review.

## Hard Reject Rules

Risk review hard rejects when:

- candidate structure is incomplete
- trust gate is not ready for risk review
- entry price is missing or non-positive
- stop loss is missing or non-positive
- target price is missing or non-positive
- stop direction is invalid
- slippage allowance is negative
- configured max leverage is above `1.0`
- requested leverage is above `1.0`
- broker execution is already allowed
- an execution instruction already exists
- approval is already granted before risk review
- slippage-adjusted loss exceeds allowed loss after rounding

Low reward/risk is a warning block, not a hard reject.

## Safety Notes

Risk review is domain-only in this phase. It does not add persistence, broker calls, live mode changes, approval mutations, or execution instruction creation.

The default risk fraction is `1%` unless a stricter explicit config is supplied. Real broker account balances are not assumed; tests pass explicit account equity.

## Deferred Work

- Persist `candidate_risk_reviews` only if later workflow integration needs a durable risk review audit table.
- Wire risk review into approval queue eligibility after API and workflow contracts are defined.
- Add broker reconciliation and realized slippage feedback only in later paper/shadow/live readiness phases.
