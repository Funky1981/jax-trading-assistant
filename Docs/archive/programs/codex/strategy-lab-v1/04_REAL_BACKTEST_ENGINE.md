# Replace Fake Backtest with Real Engine

## Objective
Replace `libs/utcp/backtest_local_tools.go` fake stats generator with a **real deterministic replay** engine.

## Current problem
The existing backtest tool generates fabricated win rate/sharpe. This makes research invalid.

## Requirements

### Engine must
- Replay candles chronologically for each symbol.
- Use strategy instance config to generate entries/exits.
- Produce a trade list and metrics:
  - P/L, win rate, max drawdown, exposure, avg win/loss, worst day
- Prevent look-ahead bias.
- Be deterministic: same input => same output.

### Fill model (v1, explicit)
- Entry fill: configurable:
  - either `next_bar_open` or `signal_bar_close`
- Slippage: configurable bps per instance
- Fees: configurable (bps or per-share)

## Implementation plan

### 1) Create backtest module
Add `libs/backtest/`:
- `types.go` (Trade, Bar, RunResult)
- `engine.go` (replay loop)
- `fills.go` (fill model)
- `metrics.go` (drawdown, pnl, etc.)

### 2) Wire UTCP tools to real engine
Replace fake `generateRun()` with:
- Load instance config (file/DB)
- Pull candles via UTCP market tool
- Execute engine
- Persist run + trades
- Return run stats + reference

### 3) Persistence
Add tables:
- `backtest_runs`
- `backtest_trades` (or store trades JSONB in the run row for v1)

## Acceptance criteria
- Backtest run produces a real trade list.
- No random numbers.
- Run includes config hash and git commit hash (if available at runtime).
