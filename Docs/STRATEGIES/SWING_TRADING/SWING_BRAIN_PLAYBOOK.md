# Swing Brain Playbook

## Purpose

The Swing Brain is the first trading brain for Jax.

It evaluates whether a market event creates a swing-trading opportunity over days to weeks.

It must reject weak setups by default.

## Time horizon

```text
2 days to 8 weeks
```

## Default decision

```text
NO_TRADE
```

## Allowed decisions

```text
NO_TRADE
WATCH
SETUP_FORMING
TRADE_CANDIDATE
```

## Allowed setup families v1

### 1. Event-driven pullback continuation

A strong asset pulls back after a temporary event but remains in a valid trend.

Requires:

- clear catalyst
- trend still intact
- support area identifiable
- risk/reward >= 2:1
- no major unresolved event risk

### 2. Post-earnings drift continuation

Company reports strong earnings and guidance, then confirms with price/volume follow-through.

Requires:

- earnings beat
- guidance not negative
- volume confirmation
- sector not strongly against it
- entry not chased

### 3. Commodity-linked equity dislocation

Commodity moves meaningfully and related equity has not yet repriced.

Requires:

- commodity move confirmed
- related stock/ETF lag identified
- spread/liquidity acceptable
- macro conflict low
- defined invalidation

### 4. Sector-relative repricing

A sector starts outperforming after a credible catalyst.

Requires:

- sector confirmation
- leading assets identifiable
- broad market not strongly hostile
- risk/reward valid

### 5. Index-heavyweight distortion watch

Index movement is caused by heavyweight constituents rather than broad market weakness.

Usually returns:

```text
WATCH
```

or:

```text
NO_TRADE
```

Not usually:

```text
TRADE_CANDIDATE
```

## Required inputs

Minimum v1:

- structured event
- affected assets
- daily price data
- 4-hour price data if available
- volume
- trend context
- support/resistance estimate
- sector comparison
- market regime
- economic calendar context
- open portfolio/exposure context

## Scoring

| Score | Meaning |
|---|---|
| clarity_score | How clear the event and driver are |
| edge_score | Whether there is a tradable mispricing |
| conflict_score | Whether drivers contradict each other |
| risk_score | Trade/event risk |
| confirmation_score | Whether price/volume confirms |
| timing_score | Whether entry timing is acceptable |

## Rejection rules

| Rule | Outcome |
|---|---|
| conflict_score >= 0.70 and edge_score < 0.60 | NO_TRADE |
| clarity_score < 0.50 | NO_TRADE |
| risk_score > 0.70 | NO_TRADE |
| no clear catalyst | NO_TRADE |
| no invalidation condition | NO_TRADE |
| no risk/reward calculation | NO_TRADE |
| risk/reward < 2:1 | NO_TRADE |
| major event imminent | WATCH at most |
| missing confirmation | WATCH at most |
| move already extended | WATCH or NO_TRADE |

## Candidate requirements

A `TRADE_CANDIDATE` must include:

- asset
- setup family
- catalyst
- entry zone
- invalidation condition
- stop area
- target area
- risk/reward
- required confirmations
- review horizon
- paper approval requirement

## FTSE/oil/labour rule

For the event:

```text
FTSE falls because oil drops while UK labour data is strong and BoE decision is pending.
```

Expected Swing Brain output:

```text
NO_TRADE
```

or at most:

```text
WATCH
```

Never:

```text
TRADE_CANDIDATE
```

Reason:

```text
Conflicting macro drivers. No clean asset-specific edge. Policy uncertainty unresolved.
```

## Forbidden in v1

The Swing Brain must not:

- create live orders
- day trade
- use tick-level scalping logic
- rely on options flow
- chase large gaps
- treat news sentiment alone as an edge
- create candidates without invalidation
- bypass risk veto
