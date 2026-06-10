# Strategy 3 — Rates / Inflation / Bonds Rotation ETF

## Strategy ID

`ETF_NEWS_003_RATES_BONDS_ROTATION`

## Plain-English Idea

Big inflation and interest-rate news can move bonds, tech, gold, and broad equities quickly. Jax reacts through ETFs instead of individual stocks.

## ETF Universe

Rates / bonds:
- TLT

Broad market:
- SPY
- QQQ

Gold / defensive:
- GLD

Financials:
- XLF

## News Events To Watch

Examples:
- inflation print surprise
- central-bank decision
- Fed/BoE/ECB speech
- jobs report shock
- bond-yield spike
- recession fear headline
- credit stress headline

## Event-To-ETF Mapping

```text
inflation hotter than expected -> usually bearish TLT, bearish QQQ/SPY
inflation cooler than expected -> usually bullish TLT, bullish QQQ/SPY
rate-cut expectations rising -> bullish TLT, bullish QQQ/SPY
rate-hike fear rising -> bearish TLT, bearish QQQ/SPY
financial stress -> watch XLF weakness, TLT/GLD strength
geopolitical fear -> watch GLD strength
```

## Phase-1 Direction Rule

To keep phase 1 safer:

- Prefer long-only ETF trades.
- If the thesis is bearish, either:
  - trade the defensive ETF that benefits, or
  - walk away.
- Do not use options.
- Do not use inverse ETFs until explicitly approved.

Examples:
- Hot inflation bearish for QQQ → do not short QQQ in phase 1; consider walking away.
- Fear shock bullish for GLD → consider GLD long if confirmed.
- Rate-cut shock bullish for TLT → consider TLT long if confirmed.

## Required Checks Before Trade

### Macro Event Checks

Jax must verify:

- event type
- release timestamp
- expected vs actual result, where relevant
- market interpretation
- ETF reaction direction
- no immediate contradictory headline

### Confirmation Checks

Require at least two:

- TLT confirms bond reaction
- QQQ/SPY confirms equity reaction
- GLD confirms safety reaction
- XLF confirms financial stress/relief
- yield proxy confirms, if available
- candle close confirms direction

### Priced-In Checks

Walk away if:

- ETF already made a large gap move
- first candle is too extended
- data release is old
- reaction is mixed/confused
- spread is abnormal
- liquidity is poor

## Entry Rule

Jax may propose a paper ETF trade when:

```text
news_class = macro_rates_event
AND selected_etf IN allowlist
AND selected_etf.asset_type = ETF
AND local_confirmation_count >= 2
AND move_not_overextended = true
AND spread_ok = true
AND stop_loss_defined = true
AND human_approval = true
```

## Exit Rules

Mandatory:
- stop-loss at invalidation level
- take-profit at 1.5R to 2R
- flatten-by-close for phase 1

Exit early if:
- interpretation reverses
- central-bank clarification contradicts thesis
- ETF breaks invalidation level
- market data becomes unreliable

## Walk-Away Rules

Do not trade if:

- release reaction is mixed
- selected ETF does not clearly benefit
- trade requires short/inverse exposure
- stop is too wide
- news is already priced in
- spreads widen during release volatility
- market is moving too fast for safe paper execution

## Example

News:
- inflation comes in cooler than expected

Possible Jax mapping:
- TLT long
- QQQ long

Jax waits:
- TLT confirms
- QQQ confirms
- spreads normalise
- stop can be placed

Jax proposes:
- buy TLT or QQQ
- stop below confirmation candle
- target 1.5R to 2R
- flatten before close

## Memory Retain

Bank:
- `signals`

Summary example:
```text
Macro rates candidate on TLT after cooler inflation print. TLT and QQQ confirmed rate-cut interpretation; spread normal; stop below confirmation candle; paper approval required.
```

Tags:
```json
["etf", "news", "macro", "rates", "inflation", "paper"]
```

## Reflection Questions

After trade:
- Did Jax interpret macro news correctly?
- Did ETF reaction match the expected macro mapping?
- Was entry too close to release volatility?
- Was spread acceptable?
- Did flatten-by-close improve risk?
