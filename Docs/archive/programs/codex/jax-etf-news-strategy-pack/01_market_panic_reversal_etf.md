# Strategy 1 — Market Panic Reversal ETF

## Strategy ID

`ETF_NEWS_001_MARKET_PANIC_REVERSAL`

## Plain-English Idea

When scary news hits the whole market, ETFs often dump quickly. Sometimes the first move is emotional and overdone. Jax waits for confirmation that panic is fading, then paper trades a rebound using a broad liquid ETF.

## ETF Universe

Preferred:
- SPY
- QQQ
- DIA
- IWM

Avoid:
- leveraged ETFs
- inverse ETFs
- volatility ETFs
- options

## News Events To Watch

Examples:
- sudden geopolitical shock
- surprise central-bank comment
- major inflation/rate headline
- market-wide selloff headline
- systemic bank/credit fear
- broad tech selloff panic

## Entry Thesis

Trade only if:
- news caused a broad-market selloff
- ETF dropped quickly
- price stabilises
- volume confirms buyers returning
- spread is normal
- risk/reward is still acceptable

## Required Checks Before Trade

### News Checks

Jax must confirm:

- event timestamp
- trusted source
- no major contradiction
- event affects broad market, not only one company
- news is still active/relevant

### Priced-In Checks

Walk away if:

- ETF has already fully recovered
- ETF has already dropped too far and remains unstable
- move happened more than configured time window ago
- spread widened materially
- price is chopping violently with no structure

### Market Confirmation

Require at least two:

- ETF holds above recent low
- higher low forms on short timeframe
- volume spike begins fading
- broad index futures stabilise
- related ETFs stabilise too
- VIX/fear proxy stops accelerating, if available

## Entry Rule

Jax may propose a paper long trade when:

```text
news_class = market_panic
AND etf IN allowlist
AND paper_mode = true
AND price_drop >= configured threshold
AND stabilisation_confirmed = true
AND spread_ok = true
AND stop_loss_defined = true
AND human_approval = true
```

## Exit Rules

Mandatory:
- stop-loss below panic low or ATR-based level
- take-profit at 1.5R to 2R
- flatten-by-close for phase 1

Exit early if:
- new worse news appears
- ETF breaks panic low
- spread becomes abnormal
- broker/market data becomes unreliable

## Walk-Away Rules

Do not trade if:

- news is unverified
- market keeps making lower lows
- ETF is outside allowlist
- price is already near take-profit target
- stop would be too wide
- signal occurs outside regular session
- another correlated ETF trade is already open
- daily loss limit hit

## Example

News:
- surprise inflation fear causes broad market drop

Jax candidate:
- SPY or QQQ

Jax waits:
- panic low forms
- price holds
- bid/ask normal
- stop-loss can be placed

Jax proposes:
- buy SPY
- stop below recent low
- target 1.5R
- flatten before close

## Memory Retain

Bank:
- `signals`

Summary example:
```text
Market panic reversal candidate on SPY after broad inflation shock. Price stabilised after initial selloff; spread normal; stop below panic low; human approval required.
```

Tags:
```json
["etf", "news", "market-panic", "reversal", "paper"]
```

## Reflection Questions

After trade:
- Was the news genuinely market-wide?
- Did Jax enter too early?
- Was the move already priced in?
- Did stop placement make sense?
- Was ETF selection correct?
