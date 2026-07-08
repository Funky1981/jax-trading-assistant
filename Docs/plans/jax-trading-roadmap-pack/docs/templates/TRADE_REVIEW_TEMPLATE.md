# Trade Review Template

## Trade Summary

```yaml
trade_id:
candidate_id:
instrument:
direction:
setup_type:
entry_time:
exit_time:
entry_price:
planned_stop:
actual_exit_price:
target_price:
position_size:
mode: paper | shadow | live
```

## Result

```yaml
profit_loss:
r_multiple:
planned_risk:
actual_loss_or_gain:
slippage_expected:
slippage_actual:
fees:
held_duration:
```

## What Happened?

```text
Describe the trade outcome in plain English.
```

## Was The Original Thesis Correct?

```yaml
thesis_correct: yes | no | partially
reason:
```

## Did Jax Behave Correctly?

```yaml
candidate_quality_good: true | false
risk_calculation_correct: true | false
gatekeeper_correct: true | false
execution_correct: true | false
journal_complete: true | false
```

## Did The Human Operator Behave Correctly?

```yaml
followed_rules: true | false
overrode_jax: true | false
override_reason:
emotional_decision_detected: true | false
mistake_type:
```

## Lessons

```text
What should be changed, if anything?
```

## Strategy Update

```yaml
strategy_keep_running: true | false
strategy_needs_review: true | false
strategy_should_be_paused: true | false
change_required:
```
