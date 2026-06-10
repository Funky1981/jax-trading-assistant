# 02 — Cross-Asset Confirmation

## Goal

Jax must confirm a trade thesis across related assets.

A single ETF move is not enough for macro/news trading.

## Why this matters

Example:

```text
Hot CPI + QQQ down + TLT down + yields up = clean rates confirmation.

Hot CPI + QQQ down + TLT up + yields down = conflict.
```

Without cross-asset confirmation, Jax may trade the wrong cause.

## Assets/proxies to check

Phase 1 ETF/proxy set:

```text
SPY  broad market
QQQ  growth/tech
IWM  small caps/risk appetite
TLT  long-duration bonds/rates
DIA  blue-chip confirmation
XLK  tech sector
SMH  semiconductors
SOXX semiconductors
XLF  financials/credit
XLE  energy/oil inflation
GLD  safe haven/inflation/fear
```

Optional if available:

```text
2-year yield
10-year yield
DXY
VIX
oil
credit spreads
```

## Confirmation baskets

### Hawkish rates basket

Expected:

```text
QQQ down
SPY down or weak
TLT down
IWM weak
XLK/SMH weak
```

### Dovish rates basket

Expected:

```text
QQQ up
SPY up
TLT up
IWM stable/up
XLK/SMH up
```

### Risk-off basket

Expected:

```text
SPY down
QQQ down
IWM down
XLF down
GLD up or stable
TLT up or mixed depending inflation
```

### Oil/inflation shock basket

Expected:

```text
XLE up
SPY/IWM weak
TLT weak if inflationary
GLD stable/up
```

### Semiconductor/AI basket

Expected:

```text
SMH/SOXX lead
QQQ follows
SPY less affected
XLK confirms
```

## Data model

### cross_asset_confirmations

```sql
CREATE TABLE cross_asset_confirmations (
    id UUID PRIMARY KEY,
    macro_event_id UUID NULL,
    playbook_key TEXT NOT NULL,
    as_of_utc TIMESTAMPTZ NOT NULL,
    confirmation_score NUMERIC NOT NULL,
    verdict TEXT NOT NULL,
    asset_results JSONB NOT NULL,
    conflicts TEXT[] NOT NULL DEFAULT '{}',
    missing_assets TEXT[] NOT NULL DEFAULT '{}',
    reasons TEXT[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Verdicts

```text
confirmed
partially_confirmed
conflicted
insufficient_data
not_confirmed
```

## Hard rules

```text
conflicted = no trade
insufficient_data = watch only unless low-risk/manual-only
not_confirmed = no trade
confirmed = eligible for next gate
partially_confirmed = lower confidence or watch only
```

## Codex task

```text
Create Cross-Asset Confirmation service.

Inputs:
- playbook expected asset directions
- reaction snapshots
- available ETF/proxy data

Outputs:
- confirmation score
- verdict
- conflicts
- missing assets
- reasons
```

## Tests

```text
hot CPI with QQQ/TLT down = confirmed
hot CPI with QQQ down/TLT up = conflicted
cool CPI with QQQ/TLT up = confirmed
AI news with SMH up but QQQ flat = partially_confirmed
missing TLT marks missing_assets
```

## Acceptance criteria

```text
cross-asset verdict persisted
conflicts visible in evidence bundle
candidate generator blocked by conflicted verdict
missing assets are explicit
