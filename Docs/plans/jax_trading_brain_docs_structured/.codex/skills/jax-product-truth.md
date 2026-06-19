# Jax Product Truth Skill

Use this skill before planning or coding any Jax change.

Jax is an event-driven trading research assistant.

Primary behaviour:

- watch market events
- explain what is happening
- reject weak setups
- surface only high-quality trade candidates
- require human approval
- start with paper trading
- perform research and backtesting
- learn from every decision including no-trades

Default decision:

```text
NO_TRADE
```

Forbidden drift:

- building generic platform features without moving a named product capability
- adding live trading
- adding auto execution
- adding day-trading infrastructure before swing vertical slice is proven
- creating docs without acceptance tests
- creating code without capability matrix update
- treating LLM prose as the source of truth

Before outputting a plan, include:

- delivers
- does not deliver
- capability changed
- tests required
- remaining gaps
