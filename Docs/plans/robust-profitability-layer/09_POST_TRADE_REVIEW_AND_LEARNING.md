# 09 — Post-Trade Review and Learning

## Goal

Make Jax learn from paper-trading outcomes.

Jax should record not just win/loss, but whether the reasoning was good.

## Capture

```text
candidate thesis
entry
stop
target
position size
approval decision
actual price path
max favourable excursion
max adverse excursion
exit reason
R multiple
rule violations
operator notes
lessons
```

## Data model

### trade_reviews

```sql
CREATE TABLE trade_reviews (
    id UUID PRIMARY KEY,
    candidate_id UUID NOT NULL,
    symbol TEXT NOT NULL,
    strategy_key TEXT NOT NULL,
    entry_price NUMERIC NULL,
    exit_price NUMERIC NULL,
    stop_price NUMERIC NULL,
    target_price NUMERIC NULL,
    mfe_r NUMERIC NULL,
    mae_r NUMERIC NULL,
    final_r NUMERIC NULL,
    outcome TEXT NOT NULL,
    what_worked TEXT[] NOT NULL DEFAULT '{}',
    what_failed TEXT[] NOT NULL DEFAULT '{}',
    lesson TEXT NULL,
    reviewed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Outcomes

```text
win
loss
breakeven
avoided_good_trade
avoided_bad_trade
invalidated_before_entry
manual_reject_correct
manual_reject_incorrect
```

## Review questions

```text
Was the event classification correct?
Was the fundamental thesis correct?
Was the chart confirmation valid?
Did cross-asset confirmation help?
Was the entry too late?
Was the stop logical?
Was reward:risk realistic?
Did Jax chase?
Was there a missed confounder?
Should the playbook change?
```

## Learning loop

After review:

```text
store case study
update strategy stats
tag pattern
make recommendation for rule tuning
do not auto-change live rules without approval
```

## Codex task

```text
Create Post-Trade Review service.

Inputs:
- candidate
- price path
- trade result
- operator feedback

Outputs:
- review record
- case study update
- strategy performance update
```

## Tests

```text
winning trade creates review
losing trade creates review
manual reject can be marked correct/incorrect
MFE/MAE calculated from candles
review cannot mutate original decision
```

## Acceptance criteria

```text
every candidate can be reviewed
reviews feed memory/stats
rule changes require manual approval
