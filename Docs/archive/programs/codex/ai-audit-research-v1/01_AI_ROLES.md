# AI Roles in Jax (allowed vs disallowed)

## Allowed AI roles (v1)
AI can:
- Classify events (earnings/news type, severity)
- Extract structured fields from text (tickers, guidance direction)
- Rank candidate symbols
- Produce human-readable rationale for a trade idea
- Suggest parameter ranges for research sweeps (bounded)
- Summarise backtest results and anomalies (NOT compute them)

## Disallowed AI roles (v1)
AI must NOT:
- Place trades automatically
- Override risk engine
- Modify strategy logic at runtime
- Change parameters silently
- Decide position sizing or leverage

## Practical implementation rule
AI output is an **advisory artifact** that can influence which deterministic strategy type/instance is selected,
but cannot bypass deterministic constraints.

