# 09 — ETF News Strategy Integration

## Strategies

```text
ETF_NEWS_001_MARKET_PANIC_REVERSAL
ETF_NEWS_002_SECTOR_MOMENTUM
ETF_NEWS_003_RATES_BONDS_ROTATION
```

## Strategy Contract

Each strategy defines:

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

## Rule

Strategies produce candidates, not orders.

## Acceptance Criteria

- Strategies visible/selectable.
- Each has beginner explanation.
- Each can be disabled.
- Each only emits allowlisted ETFs.
- All candidates route through approval.
- No strategy submits broker orders.
