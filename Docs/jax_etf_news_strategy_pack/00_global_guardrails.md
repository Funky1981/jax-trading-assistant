# Global ETF Paper-Trading Guardrails

## Applies To

All ETF news strategies in this pack.

## Hard Requirements Before Any Trade

Jax must verify:

1. Runtime
   - `ALLOW_LIVE_TRADING=false`
   - broker reports paper mode
   - execution is enabled only for paper
   - audit logging is enabled

2. Instrument
   - symbol is in ETF allowlist
   - asset type is ETF
   - not leveraged
   - not inverse
   - not volatility-linked
   - not option/derivative product

3. Market Data
   - latest quote available
   - quote is not stale
   - bid and ask present
   - spread below configured threshold
   - candle data available
   - no trading halt detected

4. Risk
   - stop-loss required
   - take-profit or flatten-by-close required
   - position size calculated before order
   - max daily loss not breached
   - max open positions not breached
   - max trades per day not breached
   - no duplicate/correlated ETF already open

5. News
   - source is trusted
   - event is recent
   - event is not already fully priced in
   - at least one confirming signal exists
   - no contradictory major news exists

6. Approval
   - Jax proposes
   - operator approves
   - only approved order path may submit to broker

## Universal Walk-Away Rules

Jax must not trade if:

- news is vague, rumour-only, or social-media-only
- move already happened and risk/reward is poor
- ETF spread is abnormal
- volume/liquidity is abnormal
- market is within first 5 minutes or last 10 minutes unless explicitly allowed
- scheduled major event is imminent and unrelated
- stop-loss cannot be placed
- broker paper mode cannot be proven
- confidence is below strategy threshold

## Stop-Loss Rules

Every ETF entry must include a stop-loss.

Phase 1 options:
- fixed percentage stop
- ATR-based stop if available
- previous candle low/high stop

Recommended starting point:
- broad ETFs: 0.75% to 1.5%
- sector ETFs: 1.0% to 2.0%
- bond/gold ETFs: 0.5% to 1.25%

Jax must never widen a stop after entry.

## Take-Profit / Exit Rules

At least one must be present:
- take-profit target
- trailing stop
- flatten-by-close
- time-based exit
- signal invalidation exit

Phase 1 recommendation:
- take-profit at 1.5R to 2R
- flatten before market close
- no overnight holds unless explicitly approved

## Paper Trading Evidence Required

Each trade must record:

- news source
- event classification
- selected ETF
- reason for selection
- rejected alternatives
- entry price
- stop-loss
- take-profit
- position size
- spread at decision time
- quote timestamp
- approval decision
- order id
- fill info
- exit reason
- P/L
- post-trade reflection
