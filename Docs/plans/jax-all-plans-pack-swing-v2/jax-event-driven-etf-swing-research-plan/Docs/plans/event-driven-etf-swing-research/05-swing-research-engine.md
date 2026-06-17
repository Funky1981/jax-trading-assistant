# 05 — Swing Research Engine

## Goal

Make swing research the primary Jax strategy path.

Jax should answer:

```text
Is this event likely to create a 2-10 day ETF opportunity, or is it only an intraday reaction/no trade?
```

## Swing Research Pipeline

```text
ResearchTrigger
  -> source verification
  -> ETF mapping
  -> event classification
  -> historical event study
  -> priced-in check
  -> swing edge score
  -> confounder check
  -> risk thesis
  -> candidate eligibility
  -> evidence bundle
```

## Research Thesis Contract

```json
{
  "thesisId": "uuid",
  "eventId": "uuid",
  "symbol": "QQQ",
  "strategyId": "etf_swing_sector_event_momentum_v1",
  "horizon": "swing",
  "direction": "long",
  "confidence": 0.68,
  "holdPeriodTargetDays": 3,
  "maxHoldDays": 10,
  "whyThisETF": "QQQ is exposed to large-cap growth and AI/semiconductor sentiment.",
  "historicalEdge": {
    "sampleCount": 42,
    "medianReturn3D": 0.012,
    "medianReturn5D": 0.018,
    "winRate5D": 0.62,
    "medianDrawdown5D": -0.007
  },
  "pricedIn": {
    "verdict": "not_priced_in",
    "score": 0.22,
    "hardReject": false
  },
  "confounders": [],
  "risk": {
    "riskPerTradePct": 0.25,
    "stopLossRequired": true,
    "maxPortfolioRiskPct": 1.0,
    "overnightRiskAllowed": true,
    "weekendHoldAllowed": false
  },
  "invalidators": [
    "ETF closes below stop level",
    "event thesis contradicted by new primary source",
    "high-impact macro event arrives before target hold period",
    "daily revalidation fails"
  ],
  "candidateEligible": true
}
```

## Swing Strategies

### ETF_SWING_001_MACRO_RATES_ROTATION

Primary ETFs:

```text
TLT, QQQ, SPY, GLD, XLF
```

Used for:

```text
inflation surprise
central bank surprise
bond-yield shock
rate-cut/rate-hike repricing
recession-risk shift
```

Direction examples:

```text
Dovish rates surprise -> TLT long / QQQ long candidate research
Hawkish inflation surprise -> TLT avoid/short not allowed phase 1, QQQ/SPY avoid, GLD watch
Banking stress + yields down -> TLT/GLD research
```

Phase 1 allows long-only ETF paper candidates unless shorting is explicitly enabled later.

### ETF_SWING_002_SECTOR_EVENT_MOMENTUM

Primary ETFs:

```text
SMH, SOXX, XLK, XLE, XLF, IWM, GLD
```

Used for:

```text
AI/semiconductor shock
oil/energy shock
banking stress
sector regulation
supply chain shift
commodity shock
```

### ETF_SWING_003_RISK_OFF_RISK_ON_REVERSAL

Primary ETFs:

```text
SPY, QQQ, DIA, IWM, GLD, TLT
```

Used for:

```text
market panic
geopolitical shock
systemic risk headline
liquidity scare
risk-on recovery after overreaction
```

## Candidate Eligibility Rules

A swing thesis can become a paper candidate only if:

```text
ETF is phase-one allowlisted
paper mode is proven
human approval is required
source verification passes
historical sample quality passes
priced-in hard reject is false
confounder hard reject is false
calendar risk passes
stop-loss exists
max hold days exists
revalidation schedule exists
position sizing passes
```

## Swing Risk Defaults

```text
risk_per_trade_pct = 0.25 initially
max_open_swing_positions = 1 initially
max_hold_days = 10
weekend_hold_allowed = false initially
stop_loss_required = true
take_profit_optional = true
trailing_review_required = true
```

## Daily Revalidation

For every open swing paper position, Jax must run a daily revalidation:

```text
1. Has thesis changed?
2. Has new contradictory news arrived?
3. Has the ETF hit stop-loss or invalidation level?
4. Has a high-impact calendar event arrived?
5. Has max hold period been reached?
6. Is liquidity/spread still acceptable?
7. Should position be held, reduced, closed, or reviewed manually?
```

Daily revalidation output:

```json
{
  "candidateId": "uuid",
  "tradeId": "uuid",
  "status": "warn",
  "recommendedAction": "manual_review",
  "summary": "Original AI/semiconductor thesis remains valid, but CPI is due tomorrow before open.",
  "evidence": {
    "newConfounders": ["CPI release tomorrow"],
    "priceVsStop": "above_stop",
    "daysHeld": 2,
    "maxHoldDays": 10
  }
}
```

## Tests

- Swing thesis does not create candidate when historical sample count is insufficient.
- Swing thesis does not create candidate when priced-in verdict is hard reject.
- Swing thesis does not create candidate when high-relevance confounder exists.
- Swing candidate requires stop-loss and max hold period.
- Swing candidate requires daily revalidation.
- Swing paper execution remains approval-gated.
