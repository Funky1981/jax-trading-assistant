# 10 — Analysis Output Contracts

## Purpose

Define stable output contracts for TA, FA, scoring, and review.

These contracts make the UI, evidence bundle, memory, and tests consistent.

## Technical analysis output

```json
{
  "symbol": "QQQ",
  "timeframe": "15m",
  "trend_state": "downtrend",
  "structure_state": "breakdown",
  "technical_score": 78,
  "verdict": "confirmed_bearish",
  "reasons": [
    "Broke below pre-event low",
    "Failed VWAP reclaim",
    "Volume expanded above baseline",
    "QQQ weaker than SPY"
  ],
  "invalidation_rules": [
    "Reject bearish thesis if QQQ reclaims VWAP",
    "Reject if price closes back inside pre-event range"
  ],
  "hard_blocks": []
}
```

## Fundamental analysis output

```json
{
  "symbol": "QQQ",
  "event_type": "US_NONFARM_PAYROLLS",
  "fundamental_score": 82,
  "verdict": "strong_bearish",
  "expected_market_impact": "Strong jobs reduce rate-cut expectations and pressure growth equities.",
  "reasons": [
    "Payrolls materially beat forecast",
    "Unemployment did not offset the hawkish interpretation",
    "QQQ is rate-sensitive"
  ],
  "confounders": [],
  "missing_evidence": [
    "Direct 2-year yield feed unavailable"
  ]
}
```

## Analyst decision output

```json
{
  "symbol": "QQQ",
  "fundamental_score": 82,
  "technical_score": 78,
  "risk_score": 74,
  "confidence_score": 70,
  "candidate_score": 78.1,
  "decision": "candidate_allowed",
  "hard_vetoes": [],
  "reasons": [
    "Fundamental and technical analysis align",
    "Risk can be defined",
    "No major confounder detected"
  ],
  "approval_required": true
}
```

## No-trade output

```json
{
  "symbol": "QQQ",
  "decision": "candidate_rejected",
  "hard_vetoes": [
    "chart_confirmation_missing",
    "major_confounder_detected"
  ],
  "reasons": [
    "QQQ reclaimed VWAP after the initial drop",
    "A separate mega-cap headline may explain the move"
  ],
  "approval_required": false
}
```

## UI labels

Use these human labels:

```text
Fundamental Analysis
Technical Analysis
Risk Review
Final Decision
Why Jax rejected this
What would change the decision
```

## Contract rules

```text
Never hide missing evidence
Never hide hard vetoes
Never convert no-trade into weak candidate
Never allow LLM text to contradict structured verdict
```
