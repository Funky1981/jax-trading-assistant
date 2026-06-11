# Robust Profitability Layer Testing

Use this guide to test the robust profitability layer locally after pulling or building the current branch.

## What Is Ready

Implemented and testable now:

- Deterministic robust-profitability services in `internal/modules/profitability`.
- Database migration `000042_robust_profitability_layer`.
- Read-only performance API: `GET /api/v1/robust/performance`.
- Active plan docs in `Docs/plans/robust-profitability-layer/`.

Not implemented in this pass:

- A dedicated frontend page for robust profitability metrics.
- Live provider ingestion for calendar/confounder/execution-quality feeds.
- Any broker/order-writing path. This layer is read-only or candidate-gating logic only.

## Start Locally

From the repo root:

```powershell
.\start.ps1 -Build -TestMode quick
```

Expected core services:

- Trader frontend API: `http://localhost:8081/health`
- Trader runtime: `http://localhost:8100/health`
- Research runtime: `http://localhost:8091/health`
- IB bridge: `http://localhost:8092/health`
- Agent0 service: `http://localhost:8093/health`

Check health:

```powershell
Invoke-RestMethod http://localhost:8081/health
Invoke-RestMethod http://localhost:8100/health
Invoke-RestMethod http://localhost:8091/health
Invoke-RestMethod http://localhost:8092/health
Invoke-RestMethod http://localhost:8093/health
```

The same command starts the frontend. Open:

```text
http://localhost:5173
```

## Apply Or Verify Migrations

If this is an existing local database, make sure migration `42` has run.

```powershell
docker compose up db-migrate
docker compose exec postgres psql -U jax -d jax -c "SELECT version, dirty FROM schema_migrations ORDER BY version;"
```

Verify robust tables exist:

```powershell
docker compose exec postgres psql -U jax -d jax -c "\dt *regime*"
docker compose exec postgres psql -U jax -d jax -c "\dt *walkaway*"
docker compose exec postgres psql -U jax -d jax -c "\dv *performance*"
```

## Automated Checks

Run the targeted checks used for this implementation:

```powershell
scripts\go-verify.ps1 -Mode standard -Packages ./internal/modules/profitability/...
go test ./db/postgres/migrations ./cmd/trader
go test ./internal/modules/...
scripts\golden-check.ps1 -Mode verify
git diff --check
```

Expected result: all commands pass.

## API Smoke Test

The robust dashboard endpoint is read-only:

```powershell
Invoke-RestMethod http://localhost:8081/api/v1/robust/performance
```

Expected shape:

```json
{
  "funnel": {
    "eventsAnalyzed": 0,
    "candidatesCreated": 0,
    "blockingWalkaways": 0,
    "reviewedTrades": 0
  },
  "strategies": []
}
```

The counts may be non-zero if your local database already has macro events, candidates, walk-away decisions, or trade reviews.

This must be rejected because the endpoint is read-only:

```powershell
Invoke-WebRequest -Method Post -Uri http://localhost:8081/api/v1/robust/performance
```

Expected result: HTTP `405 Method Not Allowed`.

## Insert A Local Review Fixture

Use this to create one local paper-review row and verify the performance view.

```powershell
docker compose exec postgres psql -U jax -d jax -c "
INSERT INTO trade_reviews (
  candidate_id, symbol, strategy_key, entry_price, exit_price, stop_price,
  target_price, mfe_r, mae_r, final_r, outcome
) VALUES (
  gen_random_uuid(), 'QQQ', 'cpi_rates_shock', 100, 103, 98,
  104, 2.5, -1.0, 1.5, 'win'
);
"
```

Then call:

```powershell
Invoke-RestMethod http://localhost:8081/api/v1/robust/performance
```

Expected result:

- `funnel.reviewedTrades` increases by at least `1`.
- `strategies` includes `cpi_rates_shock`.
- `cpi_rates_shock.averageR` is positive.

Clean up the fixture:

```powershell
docker compose exec postgres psql -U jax -d jax -c "
DELETE FROM trade_reviews
WHERE strategy_key = 'cpi_rates_shock'
  AND symbol = 'QQQ'
  AND final_r = 1.5;
"
```

## Manual Gate Scenarios

Use the Go tests as executable examples for the gate behavior:

```powershell
go test ./internal/modules/profitability -run TestClassifyMarketRegime
go test ./internal/modules/profitability -run TestEvaluateCrossAssetConfirmation
go test ./internal/modules/profitability -run TestEvaluateCandidateGateBlocksHardVetoes
go test ./internal/modules/profitability -run TestRunRiskSimulation
```

Expected behavior:

- Risk-on inputs classify as `risk_on`.
- QQQ down with TLT up against a hot-CPI basket returns `conflicted`.
- Conflicted cross-asset confirmation blocks candidate progression.
- Risk simulation returns warnings or verdicts without touching broker/order paths.

## Safety Checks

Confirm the robust layer has no broker write path:

```powershell
rg -n "broker|order|execute|ALLOW_LIVE_TRADING" internal\modules\profitability cmd\trader\robust_api.go
```

Expected result:

- No broker/order execution calls in `internal/modules/profitability`.
- `cmd/trader/robust_api.go` only reads database views and returns JSON.

## Troubleshooting

If `/api/v1/robust/performance` returns a missing relation error:

```powershell
docker compose up db-migrate
docker compose restart jax-trader
```

If `jax-trader` is not ready because no approved strategy artifacts are loaded, the frontend API may still answer `:8081` health checks, but runtime readiness on `:8100/ready` can fail. See `Docs/OPERATIONS.md` under "Trader has no approved artifacts".

If Docker Postgres is not reachable from local Go tests:

```powershell
$env:TEST_DATABASE_URL = "postgresql://jax:jax@localhost:5433/jax?sslmode=disable"
docker compose up -d postgres
go test ./cmd/trader
```
