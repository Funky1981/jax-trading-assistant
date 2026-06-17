# 06 — Optional Intraday Paper Mode

## Goal

Keep intraday paper trading as a secondary mode for fast feedback and UAT, but do not let it define the whole product.

## Intraday Mode Definition

```text
Mode: intraday_paper
Purpose: paper-test same-session ETF event reactions
Execution: candidate approval only
Hold: same session only
Flatten: required before close
Risk: strict
```

## Intraday Use Cases

```text
major surprise inflation print
Fed rate shock
market panic reversal
large oil shock
semiconductor headline with immediate ETF reaction
banking stress headline
```

## Intraday Candidate Requirements

```text
horizon = intraday
fresh quote pass
spread pass
RTH pass
market session pass
stop-loss required
take-profit or exit condition required
flatten-by-close required
same-session expiry required
human approval required
paper mode proof required
```

## Intraday Walk-Away Rules

Walk away if:

```text
headline older than configured freshness window
ETF already moved target-like amount
spread too wide
bid/ask missing
quote stale
outside regular trading hours
confounder unresolved
priced-in verdict hard reject
no stop loss
manual approval not available
```

## Intraday Candidate Contract

```json
{
  "candidateType": "paper_etf",
  "horizon": "intraday",
  "symbol": "SPY",
  "direction": "long",
  "strategyId": "etf_intraday_market_panic_reversal_v1",
  "evidenceBundleId": "uuid",
  "approvalRequired": true,
  "entryPlan": {
    "entryType": "limit",
    "stopLoss": 498.50,
    "takeProfit": 505.00,
    "flattenByClose": true,
    "expiresAt": "same_session"
  }
}
```

## Separation From Swing

Do not share ambiguous config fields.

Bad:

```json
{"holdMode": "auto"}
```

Good:

```json
{"horizon": "intraday", "flattenByClose": true, "overnightRiskAllowed": false}
```

```json
{"horizon": "swing", "flattenByClose": false, "overnightRiskAllowed": true, "requiresDailyReview": true}
```

## Tests

- Intraday candidate with `flattenByClose=false` rejects.
- Intraday candidate outside RTH rejects.
- Intraday candidate with stale quote rejects.
- Intraday candidate with missing stop-loss rejects.
- Intraday candidate cannot remain open after close without close/cancel/protect workflow evidence.
