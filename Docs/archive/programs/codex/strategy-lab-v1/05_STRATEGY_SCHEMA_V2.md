# Strategy Schema V2 (Executable Instances)

## Objective
Move from descriptive strategy JSON to **executable instance configs**.

## Storage model (required)
- **Files**: `config/strategy-instances/<instance_id>.json`
- **DB**: `strategy_instances` table stores the same config JSON for audit/repro

## Strategy Instance Config (V2)

### Mandatory fields
- `instanceId`: unique instance identifier
- `strategyId`: strategy implementation key (code)
- `symbols`: explicit list OR a `universeId` reference
- `session`:
  - `timezone` (use `America/New_York` for US equities)
  - `entryWindow` (start/end)
  - `flattenTime` (flat-by-close cut-off)
- `risk`:
  - `riskPerTradePct`
  - `maxTradesPerDay`
  - `maxDailyLossPct`
  - optional: `maxConsecutiveLosses`
- `execution`:
  - `orderType` (default `LMT`)
  - `slippageBps`
  - `feeBps`
- `parameters`: strategy-specific numeric params only

### Example
```json
{
  "instanceId": "inst_same_day_earnings_drift_safe",
  "strategyId": "same_day_earnings_drift_v1",
  "symbols": ["AAPL","MSFT","AMZN"],
  "session": {
    "timezone": "America/New_York",
    "entryWindow": { "start": "10:00", "end": "11:00" },
    "flattenTime": "15:30"
  },
  "risk": {
    "riskPerTradePct": 0.25,
    "maxTradesPerDay": 3,
    "maxDailyLossPct": 1.0,
    "maxConsecutiveLosses": 3
  },
  "execution": {
    "orderType": "LMT",
    "slippageBps": 3,
    "feeBps": 1
  },
  "parameters": {
    "minGapPct": 2.0,
    "minVolumeMultiple": 1.5
  }
}
```

## Validation
Add a strict validator:
- required fields present
- time formats are `HH:mm`
- risk bounds are sane (`riskPerTradePct <= 2`, etc.)
- symbols list non-empty

## Acceptance criteria
- Instances can be loaded from file, upserted into DB, and referenced by ID.
- Backtests and paper runs record the instance ID + config hash.
