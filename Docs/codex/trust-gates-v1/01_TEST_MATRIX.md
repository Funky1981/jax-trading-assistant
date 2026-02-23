# Trust Test Matrix (what to test where)

## Services
- `ib-bridge` (FastAPI): broker API access
- `jax-market` (Go): market data ingestion + UTCP tools
- `jax-api` (Go): orchestration + storage + strategy configs
- `jax-trade-executor` (Go): execution + risk gates
- Postgres: persistence and reconciliation source

---

## Matrix

| Area | Test Type | Owner Service | What it proves | Required artifact |
|---|---|---|---|---|
| Health/startup | Integration | compose + each service | deploy stability | health logs |
| Candle schema | Unit/contract | jax-market | tool output correctness | schema test output |
| Candle reconciliation | Batch job | jax-market + scripts | data truth vs source | recon.csv + summary.md |
| Backtest golden | Unit | libs/backtest | engine correctness | go test output |
| Backtest parity | Integration | libs/backtest + runner | backtest/live parity | parity.md |
| Signal invariants | Unit/property | strategy packages | no rule violations | cases.md |
| Order intent logging | Integration | trade-executor | intent==request | intent_vs_order.csv |
| Fill reconciliation | Batch job | trade-executor + scripts | P/L truth | fills.csv + pnl_recon.md |
| Risk gates | Unit | trade-executor | trading halts correctly | gate tests |
| Failure injection | Integration | compose | safe behaviour under outage | failure report |
| Flatten-by-close | Integration | trade-executor | no overnight exposure | flatten proof |

---

## Minimum automation
- `make test` runs: unit + contract + golden tests
- `make recon` runs: data recon + pnl recon (paper)
- `make failure-test` runs: scripted outage scenarios (paper)
