# 07 — Strategy Playbooks

## Goal

Jax must not freestyle trades.

Every candidate must match a named strategy playbook with tested rules.

## Phase 1 strategies

```text
macro_reaction_continuation
macro_reaction_pullback
fed_delayed_confirmation
cpi_rates_shock
nfp_rates_shock
sector_shock_rotation
geopolitical_risk_off
overreaction_watch_only
```

## Strategy contract

Each strategy must define:

```text
name
allowed event types
allowed ETFs
entry rules
exit rules
stop rules
target rules
time limit
required confirmations
hard no-trade conditions
backtest status
risk limits
```

## Example: macro_reaction_continuation

Entry allowed only if:

```text
fundamental score >= 70
technical score >= 70
cross-asset confirmation confirmed
regime not conflicting
execution quality acceptable
risk sizing allowed
```

Exit:

```text
stop hit
target hit
VWAP reclaim against thesis
end-of-session time stop
evidence invalidation
```

## Example: fed_delayed_confirmation

Special rules:

```text
wait 15-30 minutes
ignore first move
require post-Powell confirmation
reject whipsaw
```

## Data model

### strategy_playbook_results

```sql
CREATE TABLE strategy_playbook_results (
    id UUID PRIMARY KEY,
    event_id UUID NULL,
    candidate_id UUID NULL,
    playbook_key TEXT NOT NULL,
    matched BOOLEAN NOT NULL,
    result TEXT NOT NULL,
    reasons TEXT[] NOT NULL,
    failed_rules TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Results

```text
matched_allowed
matched_watch_only
matched_blocked
no_strategy_match
```

## Codex task

```text
Create Strategy Playbook engine.

Inputs:
- event
- TA/FA scores
- regime
- cross-asset confirmation
- execution quality
- risk result

Outputs:
- playbook result
- matched strategy
- failed rules
```

## Tests

```text
hot CPI bearish continuation matches cpi_rates_shock
Fed first candle fails fed_delayed_confirmation
conflicting regime blocks playbook
no strategy match blocks candidate
failed rules are visible
```

## Acceptance criteria

```text
every candidate links to strategy playbook
no strategy match means no trade
strategy failed rules shown in evidence
