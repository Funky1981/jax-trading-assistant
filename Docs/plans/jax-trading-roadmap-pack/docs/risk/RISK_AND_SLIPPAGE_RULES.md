# Jax Risk And Slippage Rules

## Prime Directive

No trade is allowed unless the downside is known before entry.

```text
No catalyst, no stop, no sizing, no trade.
```

## Beginner Live Defaults

```text
First live bank: £100–£250
Risk per trade: 1%
Leverage: off
Max trades per week: 1–3
Max daily loss: 1 losing trade
Max weekly loss: 3 losing trades
```

## Required Trade Inputs

Jax must know these before a trade can be approved:

- Account size.
- Max risk per trade.
- Instrument.
- Direction.
- Entry price.
- Stop price.
- Target price.
- Slippage allowance.
- Position size.
- Expected reward/risk.
- Max normal loss.
- Max slippage-adjusted loss.
- Whether leverage is used.
- Margin required if leverage is used.
- Reason for trade.
- Reason to exit.
- Invalidation point.

## Position Sizing Formula

```text
risk_budget = account_size * risk_percent
stop_distance = abs(entry_price - stop_price)
realistic_risk_per_unit = stop_distance + slippage_allowance
position_size = risk_budget / realistic_risk_per_unit
```

## Example

```text
Account size: £250
Risk per trade: 1%
Risk budget: £2.50
Entry: £100
Stop: £98
Stop distance: £2
Slippage allowance: £0.50
Realistic risk per unit: £2.50
Position size: 1 unit
```

## Stop Loss Rule

A normal stop loss is not guaranteed. It usually triggers a market order.

That means:

```text
planned stop price != guaranteed exit price
```

Jax must treat slippage as part of risk.

## Slippage Rule

Slippage is the difference between the expected price and the actual fill price.

```text
Expected sell stop: £98
Actual fill: £97.40
Slippage: £0.60
```

Jax must record:

- Expected stop price.
- Actual fill price.
- Slippage amount.
- Slippage percentage.
- Market condition at fill.
- Whether slippage exceeded assumption.

## Leverage Rule

Leverage is disabled during:

- Research mode.
- Early paper mode.
- First small-bank live testing.

Leverage may only be considered after:

- A strategy has proven positive expectancy.
- Slippage is measured.
- Stop-loss behaviour is reliable.
- Broker reconciliation is working.
- Human approval remains mandatory.

## Hard Reject Rules

Reject the trade if:

- Entry price is missing.
- Stop price is missing.
- Target price is missing.
- Slippage allowance is missing.
- Position size cannot be calculated.
- Max loss exceeds allowed risk.
- Leverage is requested while disabled.
- Data is stale.
- Spread/slippage is too high.
- Catalyst is unclear.
- Human approval is missing.
