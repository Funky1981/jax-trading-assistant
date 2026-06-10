# 02 — ETF-Only Hardening

## Goal

Make ETF-only behaviour universal across the application.

## Approved Phase-One ETF Universe

```text
SPY
QQQ
DIA
IWM
XLK
XLF
XLE
SMH
SOXX
TLT
GLD
```

## Excluded Instruments

Reject in phase 1:

```text
options
leveraged ETFs
inverse ETFs
volatility ETFs
single-name stocks
crypto
forex
futures
thin/niche ETFs
```

Explicitly blocked examples:

```text
TQQQ
SQQQ
UVXY
VXX
```

## Work Required

### 1. Defaults

Search and update defaults in:

```text
docker-compose.yml
.env.example
config/*
Docs/*
scripts/*
frontend/*
strategy configs
test seed data
```

Remove single-name stock defaults from ETF phase-one mode.

### 2. Instrument Policy

Add or verify a single source of truth for ETF instruments, likely:

```text
config/etf-instruments.json
```

Each instrument should include:

```json
{
  "symbol": "QQQ",
  "asset_type": "ETF",
  "name": "Invesco QQQ Trust",
  "exchange": "SMART",
  "currency": "USD",
  "allow_paper": true,
  "allow_live": false,
  "is_leveraged": false,
  "is_inverse": false,
  "is_volatility_product": false,
  "max_spread_bps": 10,
  "quote_freshness_seconds": 60,
  "requires_stop_loss": true,
  "requires_flatten_by_close": true
}
```

### 3. Enforcement Points

ETF allowlist must be checked before:

- candidate creation
- approval
- execution instruction creation
- broker order submission
- manual order endpoints
- strategy signal generation
- mobile approval action

### 4. Tests

Add tests for:

- approved ETF accepted
- single stock rejected
- leveraged ETF rejected
- inverse ETF rejected
- volatility ETF rejected
- missing ETF config rejected
- manual ETF order blocked unless through candidate approval path

## Acceptance Criteria

- No phase-one flow can produce a candidate for non-approved symbols.
- No phase-one flow can bypass approval.
- No live order can be created.
- UI only presents approved ETF universe.
