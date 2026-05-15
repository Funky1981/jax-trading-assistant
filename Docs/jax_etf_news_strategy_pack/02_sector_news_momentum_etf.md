# Strategy 2 — Sector News Momentum ETF

## Strategy ID

`ETF_NEWS_002_SECTOR_MOMENTUM`

## Plain-English Idea

When major news affects a whole sector, Jax trades the sector ETF instead of guessing the winning single stock. This reduces single-company risk and keeps execution simple.

## ETF Universe

Technology:
- QQQ
- XLK

Semiconductors / AI:
- SMH
- SOXX

Energy:
- XLE

Financials:
- XLF

Small caps:
- IWM

Gold/fear:
- GLD

## News Events To Watch

Examples:
- AI/semiconductor demand shock
- major chip export/rule change
- oil supply shock
- bank credit/stress event
- broad cyber/tech regulation event
- major sector earnings surprise from a bellwether
- government subsidy/tariff affecting a sector

## Event-To-ETF Mapping

```text
AI / chip demand / Nvidia-style sector shock -> SMH or SOXX
broad technology momentum -> QQQ or XLK
oil supply shock -> XLE
banking/credit stress -> XLF
small-cap risk-on/risk-off -> IWM
gold/fear/geopolitical safety -> GLD
```

## Entry Thesis

Trade only if:
- news clearly maps to one sector
- ETF is liquid and allowlisted
- sector ETF confirms movement
- at least one related ETF or bellwether confirms
- price has not already overextended

## Required Checks Before Trade

### News Quality

Require:

- trusted source
- clear sector impact
- event timestamp
- no major contradiction
- event is not just rumour

### Sector Confirmation

Require at least two:

- sector ETF breaks above/below recent level
- sector ETF volume above normal
- leading stock confirms direction
- related ETF confirms direction
- broad market does not directly contradict trade

### Priced-In Checks

Walk away if:

- ETF has already made most of expected move
- entry would chase a vertical candle
- spread widened
- volume faded before entry
- news is old

## Entry Rule

Jax may propose a sector ETF paper trade when:

```text
news_class = sector_event
AND mapped_etf IN allowlist
AND mapped_etf.asset_type = ETF
AND mapped_etf.not_leveraged = true
AND mapped_etf.not_inverse = true
AND sector_confirmation_count >= 2
AND spread_ok = true
AND stop_loss_defined = true
AND human_approval = true
```

## Long / Short Direction

Phase 1 preferred:
- long-only standard ETFs

If negative sector news:
- phase 1 should usually walk away unless inverse/short-selling support is explicitly approved and tested

## Exit Rules

Mandatory:
- stop-loss below breakout/reversal level
- take-profit at 1.5R to 2R
- flatten-by-close unless operator approves longer hold

Exit early if:
- news is denied/reversed
- sector ETF loses confirmation
- related ETF diverges sharply
- broad market shock overrides the thesis

## Walk-Away Rules

Do not trade if:

- event maps to more than one unclear sector
- ETF is not allowlisted
- ETF is leveraged/inverse/volatility-linked
- signal appears after move is exhausted
- stop would be too wide
- no clear invalidation level
- existing correlated ETF position open

## Example

News:
- major AI chip demand headline

Jax mapping:
- SMH or SOXX

Checks:
- SMH moves up
- SOXX confirms
- QQQ supportive
- spread normal
- stop below breakout level

Jax proposes:
- buy SMH
- stop below confirmation level
- target 2R
- human approval required

## Memory Retain

Bank:
- `signals`

Summary example:
```text
Sector momentum candidate on SMH after confirmed AI semiconductor news. SMH and SOXX confirmed direction; spread normal; no leveraged ETF exposure; stop below breakout level.
```

Tags:
```json
["etf", "news", "sector-momentum", "semiconductors", "paper"]
```

## Reflection Questions

After trade:
- Was ETF mapping correct?
- Did the sector move continue after entry?
- Did Jax chase too late?
- Did broader market help or hurt?
- Was single-stock news too narrow for ETF exposure?
