# 01 — ETF-Only Hardening

## Approved ETFs

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

## Block

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

1. Add or verify ETF instrument source of truth.
2. Remove stock defaults from ETF paper-trading paths.
3. Enforce ETF allowlist before:
   - candidate creation
   - approval
   - execution instruction
   - broker submission
   - manual order endpoints
4. Add tests for allowed/rejected symbols.

## Suggested Instrument Shape

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

## Acceptance Criteria

- Approved ETFs accepted.
- Single stocks rejected.
- Leveraged/inverse/volatility ETFs rejected.
- Manual broker-write bypasses blocked.
- Tests prove all of the above.
