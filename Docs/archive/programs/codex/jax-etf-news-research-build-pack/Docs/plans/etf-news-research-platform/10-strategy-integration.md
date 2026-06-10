# 10 — Strategy Integration

## Goal

Integrate three ETF news strategies into Jax's strategy/research flow.

## Strategy 1

```text
ETF_NEWS_001_MARKET_PANIC_REVERSAL
```

Trades:

```text
SPY
QQQ
DIA
IWM
```

Used for:

- broad market panic
- geopolitical shock
- inflation/rate surprise
- systemic risk headlines

## Strategy 2

```text
ETF_NEWS_002_SECTOR_MOMENTUM
```

Trades:

```text
SMH
SOXX
XLK
XLE
XLF
IWM
GLD
```

Used for:

- AI/semiconductor news
- oil shocks
- banking stress
- sector regulation
- major bellwether earnings

## Strategy 3

```text
ETF_NEWS_003_RATES_BONDS_ROTATION
```

Trades:

```text
TLT
QQQ
SPY
GLD
XLF
```

Used for:

- inflation prints
- central-bank speeches
- rate-cut expectations
- bond-yield shocks
- recession fear

## Strategy Contract

Each strategy must define:

```text
id
name
plain_english_description
eligible_etfs
event_categories
required_confirmations
entry_rules
walk_away_rules
stop_loss_rule
take_profit_rule
flatten_rule
paper_only
approval_required
```

## Output

Strategies must produce candidates, not orders.

Output shape:

```json
{
  "strategy_id": "ETF_NEWS_002_SECTOR_MOMENTUM",
  "symbol": "SMH",
  "direction": "long",
  "candidate_type": "paper_etf",
  "confidence": 0.72,
  "evidence_bundle_id": "...",
  "entry_plan": {
    "entry_type": "market_or_limit",
    "stop_loss": 98.5,
    "take_profit": 103.0,
    "flatten_by_close": true
  },
  "approval_required": true
}
```

## Acceptance Criteria

- Strategies are visible/selectable in UI.
- Each has beginner explanation.
- Each can be disabled.
- Each only emits allowlisted ETFs.
- Candidate flow routes through approval.
- No strategy can submit broker orders directly.
