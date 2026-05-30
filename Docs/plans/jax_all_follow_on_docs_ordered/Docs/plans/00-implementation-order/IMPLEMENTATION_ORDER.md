# Jax Implementation Order

## Current Rule

Finish the current `redesign` branch first.

Do not interrupt the redesign with n8n, Hermes-style agents, paid AI expansion, or production live-feed work.

## Correct Order

```text
1. Finish current redesign
2. ETF-only hardening
3. Historical ETF/news research schema
4. Backfill pipeline
5. ETF event classification
6. Priced-in engine
7. Evidence bundles
8. API/token cost-saving layer
9. AI guardrails
10. ETF news strategies
11. Mobile approval
12. n8n automation
13. Full paper-trading UAT
14. Production/live-feed hardening
15. Long paper-trading proof period
```

---

# Phase 1 — Finish Current Redesign

## Goal

Complete the user-facing redesign shell.

## Focus

```text
beginner UX
AI Trading page
ETF universe screen
strategy cards
candidate evidence view
research timeline
approval pages
paper-trading test pages
trading modes visibility
```

## Acceptance Criteria

```text
frontend builds
routes work
main screens compile
beginner UX makes sense
paper/live mode is visible
ETF direction is clear
no dead-end user journeys
```

---

# Phase 2 — ETF-Only Hardening

## Approved ETFs

```text
SPY
QQQ
DIA
IWM
XLK
XLF
XLE
SMH
SOXX
TLT
GLD
```

## Block

```text
single stocks
options
leveraged ETFs
inverse ETFs
volatility ETFs
crypto
forex
futures
```

## Acceptance Criteria

```text
AAPL/NVDA/TSLA cannot become ETF paper candidates
TQQQ/SQQQ/VXX/UVXY rejected
approved ETFs accepted
manual broker-write bypasses blocked
tests prove it
```

---

# Phase 3 — Historical ETF/News Research Schema

Add:

```text
event_windows
event_confounders
event_priced_in_scores
etf_context_snapshots
research_summaries
```

Acceptance:

```text
migrations apply cleanly
tables have constraints and indexes
event windows unique per event/symbol/window
priced-in score unique per event/symbol
integration tests pass
```

---

# Phase 4 — Historical Backfill Pipeline

Backfill:

```text
ETF candles
historical news/events
macro calendar events
event studies
priced-in scores
confounders
```

Use free/cheap first:

```text
Finnhub
Alpaca
existing calendar store
NewsAPI only as helper
```

Paid later:

```text
Polygon / Massive
```

---

# Phase 5 — ETF Event Classification

Map events to ETFs:

```text
Nvidia AI news -> SMH/SOXX
oil shock -> XLE
rates/inflation -> TLT/QQQ/SPY
bank stress -> XLF
gold/fear -> GLD
```

Unknown events cannot trade.

---

# Phase 6 — Priced-In Engine

Verdicts:

```text
not_priced_in
partially_priced_in
priced_in
overreaction
unclear
```

Hard rule:

```text
priced_in = no trade
unclear = no trade
```

---

# Phase 7 — Evidence Bundles

Each candidate needs:

```text
what happened
why this ETF
historical similar events
ETF price reaction
priced-in verdict
confounders
guardrail results
entry/stop/target
walk-away reasons
beginner summary
```

No evidence bundle means no approval.

---

# Phase 8 — API/Token Cost-Saving Layer

Internal order:

```text
1. Home server LiteLLM + Redis + Postgres + Ollama
2. Jax provider boundary
3. Token/cost logging
4. Model routing config
5. Cache policy
6. Context compaction
7. UAT proving paid APIs are opt-in
```

---

# Phase 9 — AI Guardrails

AI can summarise and recommend.

AI cannot:

```text
place orders
approve trades
change stops
override priced-in rejection
override ETF allowlist
invent missing data
enable live mode
```

---

# Phase 10 — ETF News Strategies

Implement:

```text
ETF_NEWS_001_MARKET_PANIC_REVERSAL
ETF_NEWS_002_SECTOR_MOMENTUM
ETF_NEWS_003_RATES_BONDS_ROTATION
```

Strategies produce candidates, not orders.

---

# Phase 11 — Mobile Approval

Start with Telegram.

Approval must include:

```text
ETF
strategy
action
confidence
why this ETF
priced-in verdict
other news/confounders
entry
stop-loss
take-profit
risk
expiry
Approve / Reject / Snooze
```

---

# Phase 12 — n8n Automation

Only after APIs are stable.

Safe first workflows:

```text
nightly ETF data backfill
morning ETF news digest
provider health alerts
failed job alerts
end-of-day paper trade summary
post-trade reflection batch
```

Never use n8n for:

```text
trade decisions
risk logic
broker orders
live trading enablement
position sizing
stop-loss changes
```

---

# Phase 13 — Full Paper UAT

Prove:

```text
historical event ingested
ETF mapped
price reaction calculated
priced-in score created
evidence bundle generated
strategy candidate created
AI summary produced
mobile approval sent
paper execution instruction created
order tracked
exit tracked
reflection stored
```

---

# Phase 14 — Production/Live-Feed Hardening

Development:

```text
Finnhub
Alpaca
calendar store
NewsAPI helper only
```

Production later:

```text
Polygon / Massive primary
Finnhub secondary
Alpaca backup
IBKR execution
```

---

# Phase 15 — Long Paper-Trading Proof

Minimum:

```text
3–6 months
```

Track:

```text
win rate
average R
max drawdown
false positives
missed opportunities
strategy performance
ETF performance
news category performance
priced-in accuracy
approval latency
cost per candidate
cost per paper trade
```

No live trading until paper evidence proves the system.
