# 06 — ETF Event Classification

## Goal

Classify news by event type and map it to relevant ETFs even when the ETF ticker is not directly mentioned.

## Event Categories

Start with:

```text
macro_rates
inflation
central_bank
energy_oil
semiconductor_ai
broad_market
financial_credit
geopolitical
gold_safe_haven
small_caps
technology
earnings_bellwether
regulation
unknown
```

## ETF Mapping Rules

```text
AI / chip demand / Nvidia-style sector shock -> SMH, SOXX, QQQ
broad technology momentum -> QQQ, XLK
oil supply shock -> XLE
banking or credit stress -> XLF
rate cut expectations -> TLT, QQQ, SPY
rate hike fears -> TLT, QQQ, SPY
inflation shock -> TLT, SPY, QQQ, GLD
gold / fear / geopolitical safety -> GLD
small-cap risk-on or risk-off -> IWM
broad market risk-on or panic -> SPY, QQQ, DIA, IWM
```

## Classification Output

Each normalized event should be enriched with:

```json
{
  "event_type": "semiconductor_ai",
  "affected_etfs": ["SMH", "SOXX", "QQQ"],
  "primary_etf": "SMH",
  "confidence": 0.78,
  "source_quality": "trusted",
  "time_sensitivity": "high",
  "tradeable": true,
  "reason": "Headline affects semiconductor demand and ETF confirms sector exposure."
}
```

## Rule-Based First

Version 1 should be rule-based.

Do not start with ML.

Inputs:

- headline
- summary
- provider categories
- tickers
- macro calendar category
- source
- timestamp
- known ETF themes

## AI Use

AI may help explain classification but should not be the only classifier.

Hard rule:

```text
No trade candidate from AI-only classification without deterministic mapping support.
```

## Acceptance Criteria

- AI/chip news maps to SMH/SOXX.
- oil shock maps to XLE.
- rates/inflation maps to TLT/SPY/QQQ.
- banking stress maps to XLF.
- unclear news maps to `unknown` and cannot trade.
- classification reason is stored.
