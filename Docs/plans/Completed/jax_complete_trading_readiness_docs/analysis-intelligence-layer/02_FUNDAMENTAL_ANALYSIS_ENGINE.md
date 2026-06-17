# 02 — Fundamental Analysis Engine

## Goal

Make Jax understand why an event matters and what else could affect the asset.

Fundamental analysis must **not** only consider the news headline.

It must consider:

```text
the headline
the actual data behind the headline
market expectations
other same-time events
sector context
macro regime
rates/yields
earnings
policy
geopolitics
liquidity
positioning
valuation
confounders
```

## Core rule

A headline is only one input.

Jax must ask:

```text
Is this the real driver of the move, or is something else moving the market?
```

## Fundamental analysis scope

### Macro data

```text
CPI
Core CPI
PPI
Nonfarm payrolls
Unemployment
Wage growth
Retail sales
GDP
PMI
Consumer confidence
FOMC decision
Fed statement
Fed dot plot
Fed press conference
Treasury auctions
```

### Company/sector events

Even in ETF-only mode, Jax should understand stock/sector drivers because ETFs can move due to large constituents.

Examples:

```text
Nvidia news can move SMH/SOXX/QQQ
Apple/Microsoft/Amazon can move QQQ/SPY
banks can move XLF
oil majors can move XLE
mega-cap earnings can move SPY/QQQ
```

### Other events that can affect the trade

Jax must check for:

```text
same-time economic releases
Fed speakers
Treasury auctions
earnings from mega-cap constituents
geopolitical shocks
oil shocks
credit events
bank stress
currency/yield shocks
options expiry
index rebalancing
major regulatory/legal news
market-wide liquidity events
```

## Required analysis questions

For every candidate event:

```text
1. What happened?
2. Was it expected?
3. How big was the surprise?
4. What market regime are we in?
5. Which ETFs should be affected?
6. What is the expected direction?
7. Did rates/yields confirm?
8. Did related ETFs confirm?
9. Is another event a better explanation?
10. Is the event already priced in?
11. Is the fundamental thesis strong enough to trade?
```

## Fundamental result values

```text
strong_bullish
moderate_bullish
neutral
moderate_bearish
strong_bearish
conflicted
insufficient_data
```

## Fundamental score

```text
Event surprise size:          0–20
Policy/rates impact:          0–20
ETF relevance:                0–15
Expectation/priced-in clarity:0–15
Cross-market confirmation:    0–15
Confounder cleanliness:       0–10
Source quality:               0–5
```

Total:

```text
0–49   weak
50–69  moderate
70–84  strong
85+    exceptional
```

## Cross-market checks

For macro/rates trades, Jax should check:

```text
2-year yield proxy
10-year yield proxy
TLT
DXY/USD proxy if available
SPY
QQQ
IWM
sector ETF affected by theme
```

If direct yield/DXY data is not available in phase 1, Jax should mark it as missing evidence, not fake it.

## Confounder model

A confounder is anything that could explain the move better than the headline.

Examples:

```text
CPI released at same time as jobless claims
Fed speaker contradicts data reaction
mega-cap earnings dominate QQQ
Treasury auction moves yields
oil shock moves inflation expectations
geopolitical escalation creates risk-off
options expiry distorts price action
```

## Data model

### fundamental_analysis_snapshots

```sql
CREATE TABLE fundamental_analysis_snapshots (
    id UUID PRIMARY KEY,
    macro_event_id UUID NULL,
    symbol TEXT NOT NULL,
    analysis_time_utc TIMESTAMPTZ NOT NULL,
    event_summary TEXT NOT NULL,
    expected_market_impact TEXT NOT NULL,
    affected_themes TEXT[] NOT NULL,
    cross_market_checks JSONB NOT NULL,
    confounders JSONB NOT NULL,
    fundamental_score NUMERIC NOT NULL,
    verdict TEXT NOT NULL,
    reasons TEXT[] NOT NULL,
    missing_evidence TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## Fundamental hard blocks

```text
unsupported event type
unclear expected impact
major unresolved confounder
no ETF relevance
missing actual vs expected for macro release
source quality too low
market reaction contradicts thesis
```

## Example: strong jobs

```text
Headline:
US jobs beat expectations.

Fundamental read:
Growth strong but rate-cut expectations fall.
For QQQ/TLT this is hawkish-rates bearish.
For financials, impact may be mixed/positive.
Need yield and ETF confirmation.

Possible confounders:
Fed speaker
wage growth
unemployment miss
revisions
same-time ISM data
```

## Example: Nvidia AI news

```text
Headline:
Nvidia announces stronger AI demand.

ETF relevance:
SMH/SOXX high
QQQ moderate
SPY moderate

Other checks:
Is this new information?
Were semis already pricing it in?
Are yields moving against growth?
Are other chip names confirming?
Is there a competing export-control headline?
```

## Codex task

```text
Create a Fundamental Analysis Engine.

Inputs:
- macro_event or research_trigger
- ETF mapping
- source metadata
- calendar context
- related events
- optional cross-market data

Outputs:
- fundamental snapshot
- score
- verdict
- reasons
- confounders
- missing evidence
```

## Tests

```text
hot CPI creates bearish fundamental verdict for QQQ/TLT
cool CPI creates bullish fundamental verdict for QQQ/TLT
strong jobs creates hawkish-rates verdict
Nvidia AI event maps to SMH/SOXX/QQQ but not XLE
same-time Fed speech creates confounder
missing actual/expected creates insufficient_data
major conflicting event blocks candidate
```

## Acceptance criteria

```text
fundamental snapshot persisted
headline is never the only evidence
other events are checked
confounders are explicit
missing evidence is explicit
candidate generator cannot bypass fundamental verdict
```
