# Jax Product Charter

## Product truth

Jax is an event-driven trading research assistant.

It watches market events, explains what is happening, rejects weak setups, surfaces only high-quality trade candidates, requires human approval, starts with paper trading, performs research/backtesting, and learns from every decision including no-trades.

## Primary behaviour

Jax must:

1. Ingest or receive market events.
2. Explain what happened and why it matters.
3. Identify affected assets.
4. Detect conflicting signals and weak setups.
5. Default to `NO_TRADE`.
6. Upgrade only when evidence justifies it.
7. Surface high-quality candidates for review.
8. Require human approval before paper trading.
9. Store every decision.
10. Review outcomes later, including no-trades.
11. Use backtesting/research evidence before promoting setup families.

## Default decision

The default decision is:

```text
NO_TRADE
```

`NO_TRADE` is not an error. It is the most common expected outcome.

## Product positioning

Jax is not:

- a live trading bot
- an auto-execution system
- a signal spammer
- a day-trading scalper
- a generic LLM financial chatbot
- a broker replacement
- a guaranteed profit engine

Jax is:

- a market-event research assistant
- a decision filter
- a trade rejection engine
- a swing-first research system
- a paper-trading validation system
- a learning/audit system

## Trading modes

### Current active mode

```text
Research + Paper Trading Only
```

### Current first trading style

```text
Swing trading
```

Swing trading is first because it fits the current system best:

- less dependent on tick-level data
- less sensitive to latency
- suitable for event digestion
- suitable for daily / 4-hour / multi-day confirmation
- easier to backtest and review
- better fit for human approval

### Not active yet

- day trading
- long-term investing brain
- live execution
- autonomous trading
- options trading
- leverage/margin trading

## Non-negotiable rules

1. No live trading.
2. No unattended execution.
3. No auto-trading.
4. Paper trading requires human approval.
5. Every trade candidate requires evidence.
6. Every no-trade must be logged.
7. Every phase must include explicit exclusions.
8. Every implementation must update the capability matrix.
9. Every decision feature must include golden tests.
10. If Jax cannot explain the edge, it must reject the setup.

## Decision ladder

```text
NO_TRADE
WATCH
SETUP_FORMING
TRADE_CANDIDATE
PAPER_APPROVAL_REQUIRED
APPROVED_FOR_PAPER
PAPER_REJECTED
PAPER_PROVEN
```

Live trading is not part of the current roadmap.

## Success definition

Jax is successful when it can process a market event and return:

```text
What happened?
Why it matters?
Which assets are affected?
What signals conflict?
Is there a trade edge?
What decision is justified?
What should be watched next?
When should this be reviewed?
```

Most outputs should be:

```text
NO_TRADE
```

That is the point.
