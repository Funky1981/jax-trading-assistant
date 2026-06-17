# 02 — Trading Modes and Horizons

## Goal

Add explicit trading horizons so Jax can support swing-first research without weakening intraday guardrails.

## Mode Catalog

Create or extend:

```text
internal/modules/tradingmodes
```

Modes:

```json
[
  {
    "id": "swing_research",
    "name": "ETF Swing Research",
    "runtimeMode": "research",
    "executionPolicy": "no_execution",
    "assetClass": "ETF",
    "primaryHorizon": "swing",
    "paperOnly": true,
    "approvalRequired": true
  },
  {
    "id": "swing_paper",
    "name": "ETF Swing Paper",
    "runtimeMode": "paper",
    "executionPolicy": "candidate_approval_only",
    "assetClass": "ETF",
    "primaryHorizon": "swing",
    "paperOnly": true,
    "approvalRequired": true
  },
  {
    "id": "intraday_paper",
    "name": "ETF Intraday Paper",
    "runtimeMode": "paper",
    "executionPolicy": "candidate_approval_only",
    "assetClass": "ETF",
    "primaryHorizon": "intraday",
    "paperOnly": true,
    "approvalRequired": true
  }
]
```

## Horizon Contract

Add:

```go
type TradingHorizon string

const (
    HorizonResearchOnly TradingHorizon = "research_only"
    HorizonIntraday     TradingHorizon = "intraday"
    HorizonSwing        TradingHorizon = "swing"
)
```

Candidate horizon fields:

```go
type CandidateHorizonPolicy struct {
    Horizon                 string   `json:"horizon"`
    HoldPeriodTargetDays    int      `json:"holdPeriodTargetDays,omitempty"`
    MaxHoldDays             int      `json:"maxHoldDays,omitempty"`
    FlattenByClose          bool     `json:"flattenByClose"`
    OvernightRiskAllowed    bool     `json:"overnightRiskAllowed"`
    WeekendHoldAllowed      bool     `json:"weekendHoldAllowed"`
    RequiresDailyReview     bool     `json:"requiresDailyReview"`
    RevalidationSchedule    string   `json:"revalidationSchedule,omitempty"`
    ThesisInvalidators      []string `json:"thesisInvalidators,omitempty"`
}
```

## Intraday Rules

Intraday candidates must have:

```text
horizon = intraday
flatten_by_close = true
overnight_risk_allowed = false
max_hold_days = 0
fresh quote required
RTH required
same-session expiry required
```

## Swing Rules

Swing candidates must have:

```text
horizon = swing
flatten_by_close = false
overnight_risk_allowed = true
hold_period_target_days between 2 and 10
max_hold_days <= 10 in phase 1
requires_daily_review = true
calendar_risk_check = pass
weekend_hold_allowed = false by default
smaller risk size than intraday
```

## Required Strategy Families

Primary:

```text
ETF_SWING_001_MACRO_RATES_ROTATION
ETF_SWING_002_SECTOR_EVENT_MOMENTUM
ETF_SWING_003_RISK_OFF_RISK_ON_REVERSAL
```

Optional intraday paper mode:

```text
ETF_INTRADAY_001_MARKET_PANIC_REVERSAL
ETF_INTRADAY_002_NEWS_REPRICING
ETF_INTRADAY_003_RATES_REACTION
```

## Config Defaults

Create/modify:

```text
config/trading-modes/swing-research.json
config/trading-modes/swing-paper.json
config/trading-modes/intraday-paper.json
config/strategy-instances/etf-swing-macro-rates-paper-v1.json
config/strategy-instances/etf-swing-sector-momentum-paper-v1.json
config/strategy-instances/etf-swing-risk-reversal-paper-v1.json
```

Example swing instance:

```json
{
  "id": "etf-swing-macro-rates-paper-v1",
  "enabled": false,
  "mode": "swing_paper",
  "strategyTypeId": "etf_swing_macro_rates_rotation_v1",
  "assetClass": "ETF",
  "universe": ["TLT", "QQQ", "SPY", "GLD", "XLF"],
  "paperOnly": true,
  "approvalRequired": true,
  "horizonPolicy": {
    "horizon": "swing",
    "holdPeriodTargetDays": 3,
    "maxHoldDays": 10,
    "flattenByClose": false,
    "overnightRiskAllowed": true,
    "weekendHoldAllowed": false,
    "requiresDailyReview": true,
    "revalidationSchedule": "daily_after_close"
  },
  "risk": {
    "riskPerTradePct": 0.25,
    "maxOpenPositions": 1,
    "maxPortfolioRiskPct": 1.0,
    "stopLossRequired": true,
    "takeProfitRequired": false
  }
}
```

## Tests

- Catalog includes all modes.
- Intraday mode rejects overnight risk.
- Swing mode rejects flatten-by-close requirement.
- Swing mode cannot run without daily revalidation enabled.
- All strategy instances are ETF-only.
- All defaults are disabled until UAT passes.
