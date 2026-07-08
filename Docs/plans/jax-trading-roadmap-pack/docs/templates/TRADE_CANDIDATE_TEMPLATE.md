# Trade Candidate Template

## Candidate Summary

```yaml
candidate_id:
created_at:
mode: research | paper | shadow | live
instrument:
direction: long | short | avoid
setup_type:
catalyst:
market_regime:
status: draft | rejected | awaiting_approval | approved | executed | closed
```

## Trade Thesis

```text
One-sentence reason for the trade.
```

## Evidence

| Evidence Type | Source | Timestamp | Bullish/Bearish | Confidence | Notes |
|---|---|---:|---|---:|---|
| Catalyst |  |  |  |  |  |
| Market reaction |  |  |  |  |  |
| Volume/liquidity |  |  |  |  |  |
| Technical location |  |  |  |  |  |
| Contradictory evidence |  |  |  |  |  |

## Risk Plan

```yaml
account_size:
risk_percent:
risk_budget:
entry_price:
stop_price:
target_price:
stop_distance:
slippage_allowance:
realistic_risk_per_unit:
position_size:
max_normal_loss:
max_slippage_adjusted_loss:
expected_reward:
reward_risk_ratio:
leverage_used: false
margin_required:
```

## Invalidation

```text
The trade is wrong if...
```

## Gatekeeper Decision

```yaml
setup_allowed: true | false
evidence_score:
risk_check_passed: true | false
slippage_check_passed: true | false
leverage_check_passed: true | false
human_approval_required: true
final_gate_result: pass | fail
reject_reason:
```

## Human Approval

```yaml
approved_by:
approved_at:
decision: approved | rejected
override_reason:
```
