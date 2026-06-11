# 06 — Position Sizing and Portfolio Risk

## Goal

Jax must size trades from risk, not vibes.

Never use fixed stake sizing as the core rule.

## Inputs

```text
account equity
max risk per trade
entry price
stop price
symbol
volatility
confidence
current positions
same-theme exposure
daily loss
weekly loss
drawdown state
```

## Core formula

```text
cash_risk = account_equity * risk_percent
risk_per_share_or_unit = abs(entry_price - stop_price)
position_size = cash_risk / risk_per_share_or_unit
```

For ETFs, adapt unit sizing to broker constraints.

## Risk limits

Phase 1 defaults:

```text
max_risk_per_trade = 0.5%
max_macro_event_risk = 0.5%
max_daily_loss = 1.0%
max_weekly_loss = 2.0%
max_open_macro_candidates = 1
max_same_theme_exposure = 1
max_correlated_etf_exposure = 1
```

## Dynamic risk reduction

Reduce risk when:

```text
high volatility
low confidence
unclear regime
partial cross-asset confirmation
recent loss streak
drawdown active
event type has poor backtest stats
```

## Data model

### position_size_recommendations

```sql
CREATE TABLE position_size_recommendations (
    id UUID PRIMARY KEY,
    candidate_id UUID NULL,
    symbol TEXT NOT NULL,
    account_equity NUMERIC NOT NULL,
    entry_price NUMERIC NOT NULL,
    stop_price NUMERIC NOT NULL,
    risk_percent NUMERIC NOT NULL,
    cash_risk NUMERIC NOT NULL,
    position_size NUMERIC NOT NULL,
    adjusted_risk_percent NUMERIC NOT NULL,
    adjustments TEXT[] NOT NULL DEFAULT '{}',
    verdict TEXT NOT NULL,
    reasons TEXT[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Verdicts

```text
allowed
reduced
blocked
insufficient_data
```

## Hard blocks

```text
no stop
stop distance zero
risk above limit
daily loss limit hit
weekly loss limit hit
same-theme exposure exceeded
open correlated position conflict
account equity missing
```

## Codex task

```text
Create Position Sizing and Portfolio Risk service.

Inputs:
- candidate trade details
- account state
- portfolio state
- risk config

Outputs:
- size recommendation
- adjusted risk
- verdict
- reasons
```

## Tests

```text
valid entry/stop calculates size
missing stop blocks
risk over limit blocks
daily loss limit blocks
same-theme exposure blocks
high volatility reduces risk
```

## Acceptance criteria

```text
candidate cannot proceed without size recommendation
risk limits enforced
adjustments visible in evidence
no fixed stake default
