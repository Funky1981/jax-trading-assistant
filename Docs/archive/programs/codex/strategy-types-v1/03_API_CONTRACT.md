# 03_API: Strategy Type metadata endpoint (for UI)

Add endpoints in `jax-api`:

## GET /api/v1/strategy-types
Returns:
```json
[
  {
    "strategyId": "same_day_earnings_drift_v1",
    "name": "Same-day Earnings Drift v1",
    "description": "...",
    "requiredInputs": {
      "candles": ["1m"],
      "needsEarnings": true,
      "needsNews": false
    },
    "parameters": [
      {"key":"entryDelayMins","type":"int","default":30,"min":5,"max":120},
      {"key":"minGapPct","type":"float","default":2.0,"min":0.0,"max":20.0},
      {"key":"minVolumeMultiple","type":"float","default":1.5,"min":0.0,"max":10.0}
    ]
  }
]
```

## GET /api/v1/strategy-types/{strategyId}
Returns same object.

Purpose:
- UI dropdown for strategyId
- UI can auto-build a parameter form
- Instance validation can be mirrored client-side

Acceptance criteria:
- Endpoint returns all five strategy types.
- Deterministic ordering (stable).
