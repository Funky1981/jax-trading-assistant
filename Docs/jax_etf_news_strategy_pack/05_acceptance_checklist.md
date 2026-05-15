# ETF News Strategy Acceptance Checklist

## Strategy Pack Acceptance

Before these strategies are enabled even for paper trading:

- [ ] ETF allowlist exists
- [ ] ETF asset type exists
- [ ] leveraged ETFs blocked
- [ ] inverse ETFs blocked
- [ ] volatility ETFs blocked
- [ ] options blocked
- [ ] paper broker mode verified
- [ ] live trading disabled
- [ ] human approval required
- [ ] stop-loss required
- [ ] take-profit or flatten-by-close required
- [ ] quote freshness checked
- [ ] spread checked
- [ ] market session checked
- [ ] daily loss limit checked
- [ ] max open positions checked
- [ ] max trades per day checked
- [ ] order id persisted
- [ ] fills persisted
- [ ] exit reason persisted
- [ ] memory retained after decision
- [ ] post-trade reflection generated

## Live Smoke Tests

Run against paper stack:

### Market Data

- [ ] fetch quote for SPY
- [ ] fetch quote for QQQ
- [ ] fetch candles for SPY
- [ ] fetch candles for QQQ
- [ ] stale quote rejection works

### Broker

- [ ] paper mode is true
- [ ] live mode is false
- [ ] paper order preview works
- [ ] paper order create works
- [ ] paper order cancel works
- [ ] positions readback works
- [ ] account readback works

### Strategy

- [ ] market panic reversal can generate candidate
- [ ] sector momentum can generate candidate
- [ ] rates/bonds rotation can generate candidate
- [ ] invalid ETF rejected
- [ ] leveraged ETF rejected
- [ ] missing stop-loss rejected
- [ ] unapproved order rejected
- [ ] approved paper order accepted

## Go / No-Go

GO only if:

- all required gates pass
- all unsafe paths reject by default
- no live trading possible
- full audit trail exists
- paper trade can be entered and exited cleanly

Otherwise:

```text
NO-GO
```
