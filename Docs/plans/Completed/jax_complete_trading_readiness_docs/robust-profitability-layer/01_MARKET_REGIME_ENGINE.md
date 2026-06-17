# 01 — Market Regime Engine

## Goal

Jax must understand the current market regime before interpreting news or charts.

The same event can mean different things in different regimes.

Example:

```text
Hot CPI in a strong bull market:
Dip may get bought.

Hot CPI in fragile risk-off market:
QQQ/TLT may sell off hard.

Weak jobs in a dovish market:
Could lift QQQ/TLT due to rate-cut hopes.

Weak jobs in recession-fear market:
Could sell off SPY/IWM due to growth fear.
```

## Regimes to classify

Phase 1 regime labels:

```text
risk_on
risk_off
high_volatility
low_volatility
rates_dominant
growth_dominant
inflation_fear
recession_fear
liquidity_stress
tech_momentum
defensive_rotation
unclear
```

## Inputs

Use available proxies first:

```text
SPY trend
QQQ trend
IWM trend
TLT trend
GLD trend
XLF trend
XLE trend
SMH/SOXX trend
VIX proxy if available
yield proxies if available
dollar proxy if available
market breadth if available
```

If a feed is missing, mark it as missing evidence. Do not fake it.

## Simple phase 1 rules

### Risk-on

```text
SPY above 20/50 day moving averages
QQQ outperforming SPY
IWM stable or rising
VIX proxy falling/stable
credit/bank proxies stable
```

### Risk-off

```text
SPY below key moving averages
QQQ weak
IWM weak
TLT/GLD defensive bid
VIX proxy rising
```

### Rates-dominant

```text
TLT leading market reaction
growth ETFs inversely reacting to yields
QQQ/TLT relationship strong
Fed/CPI/jobs headlines dominating
```

### Liquidity stress

```text
SPY/QQQ/IWM all falling
spreads/volatility elevated
XLF weak
defensive assets mixed
moves disorderly
```

## Data model

### market_regime_snapshots

```sql
CREATE TABLE market_regime_snapshots (
    id UUID PRIMARY KEY,
    as_of_utc TIMESTAMPTZ NOT NULL,
    primary_regime TEXT NOT NULL,
    secondary_regimes TEXT[] NOT NULL DEFAULT '{}',
    confidence NUMERIC NOT NULL,
    inputs JSONB NOT NULL,
    missing_inputs TEXT[] NOT NULL DEFAULT '{}',
    reasons TEXT[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Regime decision contract

```json
{
  "primary_regime": "rates_dominant",
  "secondary_regimes": ["high_volatility"],
  "confidence": 0.74,
  "reasons": [
    "TLT and QQQ are reacting strongly to macro data",
    "Rate-sensitive ETFs are leading index moves"
  ],
  "missing_inputs": ["direct_2y_yield_feed"]
}
```

## Hard rules

```text
unclear regime = no high-confidence candidate
liquidity_stress = reduce risk or no trade
high_volatility = require stronger confirmation
rates_dominant = macro/rates playbooks allowed
risk_off = long risk assets require extra confirmation
```

## Codex task

```text
Create Market Regime Engine.

Inputs:
- ETF candles
- optional yield/VIX/DXY proxies
- recent market movement
- macro event context

Outputs:
- regime snapshot
- confidence
- reasons
- missing inputs
```

## Tests

```text
SPY/QQQ up + VIX falling = risk_on
SPY/QQQ/IWM down + VIX rising = risk_off
TLT/QQQ inverse reaction around CPI = rates_dominant
missing proxies produce missing_inputs not fake values
unclear data produces unclear regime
```

## Acceptance criteria

```text
regime snapshot persisted
candidate evidence includes regime
regime can veto or reduce confidence
unclear regime blocks high-confidence candidate
