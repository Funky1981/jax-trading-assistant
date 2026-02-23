# Trust Runbook (step-by-step)

## Step 1 — Prove market data
1. Bring stack up (paper):
   - `docker compose up -d`
2. Run candle contract tests (1m + 1d).
3. Run reconciliation job:
   - pick 20 liquid symbols
   - pick 10 historical days
   - compare to secondary provider
4. Store artifacts under `reports/data_recon/<date>/`.

Stop if any timestamp or missing-bar issue appears.

## Step 2 — Prove backtest engine
1. Run golden tests.
2. Run invariants suite.
3. Run parity test (replay vs sim-live).

Stop if trade lists differ without a documented reason.

## Step 3 — Prove strategy rules
For each strategy instance:
- run unit tests
- run property tests
- confirm no signals outside session windows

## Step 4 — Paper execution verification
1. Enable paper account only.
2. Execute 50–200 trades (small size).
3. Reconcile intent vs broker order.
4. Reconcile fills and P/L.

## Step 5 — Failure tests
Run outage scripts:
- ib-bridge down during execution
- db down during persist
- restart executor with open positions

Expected: trading halts safely; flatten still works.

## Step 6 — Flatten-by-close proof
Run at least 10 sessions where:
- positions are opened intraday
- system flattens at configured time
- end-of-day positions/orders are zero

## Promotion gate
Only consider tiny live size after:
- 30 trading days paper
- 0 reconciliation failures
- 0 flatten failures
- 0 duplicate order incidents
