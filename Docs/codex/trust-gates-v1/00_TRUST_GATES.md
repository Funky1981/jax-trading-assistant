# Jax Trust Gates v1 (Go/IB)

Purpose: define **objective gates** you must pass before any real-money trading.
Applies to your repo services: `jax-market`, `jax-api`, `jax-trade-executor`, `ib-bridge`, Postgres.

## Trust rule
If a gate is not proven with evidence (tests + reconciliation logs), the system is not trusted.

---

## Gate 0 — Build and deploy hygiene (must pass first)

### Evidence required
- `go test ./...` passes
- lints pass (if enabled)
- services start cleanly via `docker compose up`
- health endpoints return OK:
  - `jax-api /health`
  - `jax-market /health`
  - `jax-trade-executor /health`
  - `ib-bridge /health`

### Acceptance criteria
- No panics on startup
- Logs show config loaded, DB connected, providers reachable (or explicitly disabled)

---

## Gate 1 — Market data correctness (most important)

### Why
Bad timestamps / missing bars / wrong OHLC silently destroy backtests and signals.

### Tests
1. **Schema/contract tests** (UTCP tools)
   - `market.get_candles` returns:
     - monotonic timestamps
     - no duplicates
     - expected bar counts for (from,to,timeframe)
2. **Cross-source reconciliation**
   - Choose: 20 symbols × 10 random trading days × timeframe (1m and 1d)
   - Compare your candles vs an independent source (Polygon/Alpaca/IB historical)
   - Compute:
     - % missing bars
     - max absolute OHLC diff
     - timestamp alignment errors

### Acceptance criteria (suggested)
- Missing bars: < 0.5% on liquid symbols
- Timestamp mismatch: 0 (after normalisation to UTC)
- OHLC diffs: within your source’s documented tolerances (or explain and document)

### Evidence artifacts
- `reports/data_recon/<date>/recon.csv`
- `reports/data_recon/<date>/summary.md`

---

## Gate 2 — Deterministic backtest correctness

### Why
If your backtest is not deterministic and bias-controlled, results are not credible.
Note: your current repo backtest tool was a stub earlier; this gate requires the real engine.

### Tests
1. **Golden dataset tests**
   - Use tiny synthetic candle series with known expected trades.
   - Assert trade list equals expected.
2. **Invariants**
   - entry_time < exit_time
   - no trades after flatten time (same-day strategies)
   - P/L matches arithmetic from fills model
   - no future candle access
3. **Replay parity**
   - Run the same strategy in:
     - backtest mode (replay)
     - sim-live mode (stream the same candles)
   - Trade list must match (within fill model settings).

### Acceptance criteria
- Same input → same run_id → same trades and stats
- No randomness
- Invariant suite 100% passing

### Evidence artifacts
- `go test ./...` includes `backtest_golden_test.go`
- `reports/backtest_validation/<date>/parity.md`

---

## Gate 3 — Signal correctness (strategy logic)

### Why
Even with correct data, wrong signal logic creates unbounded risk.

### Tests (per strategy instance)
- Unit tests for the entry/exit criteria
- Property tests on edge cases:
  - missing data
  - partial session
  - spike candles
  - halts (no bars)
- Confirm the strategy respects:
  - entry window
  - max trades/day
  - flatten time

### Acceptance criteria
- Strategy never produces a signal outside configured windows
- Strategy refuses to trade when required inputs are missing

### Evidence artifacts
- `reports/strategy_validation/<strategy>/<date>/cases.md`

---

## Gate 4 — Order intent vs broker order (execution correctness)

### Why
You must prove: “what I intended” == “what I sent” == “what broker received”.

### Required logging (must be implemented)
For every execution request:
- correlation_id
- strategy_instance_id
- order_intent (symbol, side, qty, order type, prices, time)
- broker_request payload (sanitised)
- broker_response (order id, status)

### Tests
- Execute 50–200 paper orders across symbols and days.
- Verify 100%:
  - intent matches request
  - broker response persisted
  - no duplicate orders on retry/restart

### Acceptance criteria
- 0 unmatched intents
- 0 duplicate order submissions for the same signal_id

### Evidence artifacts
- `reports/execution_recon/<date>/intent_vs_order.csv`

---

## Gate 5 — Fill reconciliation and P/L truth

### Why
Your P/L must reconcile to broker fills, not assumptions.

### Tests
- For each paper trade:
  - fetch IB fills / status
  - persist fills
  - recompute P/L from fills
- Daily:
  - compare your “executions count” vs IB executions count
  - compare positions at end of day

### Acceptance criteria
- 100% trades have terminal broker status stored
- P/L differences are explainable (fees, FX, partial fills) and within tolerance

### Evidence artifacts
- `reports/pnl_recon/<date>/pnl_recon.md`
- `reports/pnl_recon/<date>/fills.csv`

---

## Gate 6 — Risk controls proven under failure

### Why
Most systems fail during outages and restarts, not during happy-path.

### Failure injection scenarios (paper only)
1. IB bridge down mid-execution
2. DB down during trade persist
3. trade-executor restart with open position
4. jax-market outage (no candles)
5. network timeouts causing retries

### Expected behaviour
- New trading halts safely
- No duplicate orders
- System recovers to a known state
- Forced flatten still occurs (or trading remains disabled until manual intervention)

### Acceptance criteria
- All scenarios pass with documented outcomes and logs
- Any unsafe condition results in “stop trading” not “try harder”

### Evidence artifacts
- `reports/failure_tests/<date>/report.md`

---

## Gate 7 — Flat-by-close enforcement (same-day strategies)

### Why
If you claim “same-day”, the system must enforce it.

### Tests
- Place positions in paper mode.
- Verify:
  - all open orders cancelled at flatten time
  - all positions closed by close
  - no new orders accepted after flatten time

### Acceptance criteria
- End-of-day positions = 0
- End-of-day open orders = 0

### Evidence artifacts
- `reports/flatten/<date>/proof.md`

---

## Promotion rule: backtest → paper → tiny live
- Pass Gates 0–3 to run backtests.
- Pass Gates 0–5 to run paper execution.
- Pass Gates 0–7 for 30+ trading days to consider tiny live size.
