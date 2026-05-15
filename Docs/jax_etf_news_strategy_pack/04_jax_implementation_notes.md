# Jax Implementation Notes

## Where These Strategies Should Fit

These strategies should not directly execute trades.

Recommended flow:

```text
news/event detected
    ↓
strategy generates ETF candidate
    ↓
risk/eligibility gate
    ↓
Jax memory recall
    ↓
Agent0/research review
    ↓
human approval
    ↓
paper broker order
    ↓
position monitoring
    ↓
exit rule
    ↓
memory retain + reflection
```

## Required Data Structures

Jax should support:

```text
Instrument
- symbol
- asset_type
- name
- exchange
- currency
- allow_paper
- allow_live
- is_leveraged
- is_inverse
- is_volatility_product
- min_avg_volume
- max_spread_bps
```

```text
ETFSignal
- strategy_id
- symbol
- direction
- news_event_id
- confidence
- entry_reason
- invalidation_reason
- stop_loss
- take_profit
- flatten_by
- risk_amount
- approval_required
```

## Required Gates

Before broker submission:

1. ETF allowlist gate
2. paper-mode gate
3. quote freshness gate
4. spread gate
5. session gate
6. stop-loss gate
7. daily loss gate
8. max trades gate
9. human approval gate
10. broker contract gate

## Broker Order Expectations

For IBKR-style ETF paper orders, verify:

- paper account active
- correct symbol
- correct exchange/routing
- correct currency
- correct quantity
- order type supported
- protective exit can be tracked
- order id persisted
- fill status persisted

## Logging / Audit

Every decision should log:

- why trade was considered
- why ETF was selected
- what alternatives were rejected
- which guardrails passed
- which guardrails failed
- who approved
- order response
- exit reason

## Memory Banks

Recommended:

- `signals`
  - candidate ETF signals

- `trades`
  - approved paper trade decisions and results

- `reflections`
  - post-trade lessons if/when explicitly stored

- `research`
  - event analysis, ETF mapping notes, sector thesis

## Do Not Implement Yet Unless Missing

These strategies assume Jax already has or will have:

- ETF allowlist
- asset type handling
- paper-only guard
- stop-loss enforcement
- approved order path
- broker order lifecycle
- trade result persistence
- memory retain/recall/reflect
