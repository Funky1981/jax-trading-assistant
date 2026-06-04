# 05 — ETF Event Classification

## Goal

Map news to ETFs even when ETF ticker is not mentioned.

## Categories

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

## Mapping Examples

```text
AI / chip demand -> SMH, SOXX, QQQ
technology momentum -> QQQ, XLK
oil shock -> XLE
banking stress -> XLF
rate cuts -> TLT, QQQ, SPY
rate hike fear -> TLT, QQQ, SPY
inflation shock -> TLT, SPY, QQQ, GLD
gold/fear -> GLD
small-cap risk -> IWM
broad market -> SPY, QQQ, DIA, IWM
```

## Rule

Version 1 must be rule-based.

AI may explain classification but must not be the only classifier.

## Acceptance Criteria

- AI/chip maps to SMH/SOXX.
- Oil maps to XLE.
- Rates/inflation maps to TLT/SPY/QQQ.
- Banking stress maps to XLF.
- Unknown cannot trade.
- Reason/confidence stored.
