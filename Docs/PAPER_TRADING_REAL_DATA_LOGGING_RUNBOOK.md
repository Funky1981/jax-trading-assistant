# Paper Trading with Real Data and Full Logging Runbook

## Purpose
This runbook explains how to:
- Run JAX in paper-trading mode with real market/event data.
- Verify that data is real and linked to news/earnings context.
- Verify that runs and AI decisions are logged for analysis.
- Start on a home server first, then move to GCP later.

---

## 1) Fix local IDE/module errors first

If your IDE shows errors in `libs/utcp/go.mod` but CLI passes:

1. Open the repo root (`c:\Projects\jax-trading assistant`), not only `libs/utcp`.
2. Ensure Go SDK is `1.25.x`.
3. Run:

```powershell
go work sync
cd libs/utcp
go mod tidy
go test ./...
```

4. Restart the language server / reload IDE window.

---

## 2) Understand current logging coverage

### Already logged
- `runs` with `flow_id`.
- `ai_decisions`.
- `ai_decision_acceptance`.
- `test_runs` and `gate_status`.

### Not fully logged yet
- `audit_events` is not yet fully instrumented across all paths.
- `order_intents` and `fills` tables exist, but execution path is not yet fully writing all intent/fill lifecycle records.

If you need "everything logged", plan a follow-up instrumentation pass after baseline deployment.

---

## 3) Home server first (recommended)

### 3.1 Prerequisites
- Docker + Docker Compose.
- IBKR paper account + IB Gateway/TWS paper session.
- API keys:
  - `POLYGON_API_KEY`
  - `FINNHUB_API_KEY`

### 3.2 Create `.env`

```env
POSTGRES_USER=jax
POSTGRES_PASSWORD=jax
POSTGRES_DB=jax
DATABASE_URL=postgresql://jax:jax@postgres:5432/jax?sslmode=disable

IB_GATEWAY_HOST=host.docker.internal
IB_GATEWAY_PORT=4002
IB_PAPER_TRADING=true
IB_AUTO_CONNECT=true

IB_BRIDGE_URL=http://ib-bridge:8092
JAX_ORCHESTRATOR_URL=http://jax-research:8091

POLYGON_API_KEY=REPLACE_ME
FINNHUB_API_KEY=REPLACE_ME

# Current implementation gates /api/v1/execute on this flag
ALLOW_LIVE_TRADING=true

# Fail-closed behavior for event providers in production mode
APP_ENV=production
```

### 3.3 Start stack

```powershell
docker compose up -d --build
docker compose ps
```

### 3.4 Verify service health

```powershell
iwr http://localhost:8091/health | % Content   # jax-research
iwr http://localhost:8100/health | % Content   # jax-trader
iwr http://localhost:8081/health | % Content   # frontend API surface
```

---

## 4) Verify real data providers (not fake/fallback)

### 4.1 Quote

```powershell
$req = @{ tool="market.get_quote"; input=@{symbol="AAPL"} } | ConvertTo-Json -Depth 5
iwr http://localhost:8100/tools -Method POST -ContentType "application/json" -Body $req | % Content
```

### 4.2 News

```powershell
$req = @{ tool="market.get_news"; input=@{symbol="AAPL"; limit=10} } | ConvertTo-Json -Depth 5
iwr http://localhost:8100/tools -Method POST -ContentType "application/json" -Body $req | % Content
```

### 4.3 Earnings

```powershell
$req = @{ tool="market.get_earnings"; input=@{symbol="AAPL"; limit=8} } | ConvertTo-Json -Depth 5
iwr http://localhost:8100/tools -Method POST -ContentType "application/json" -Body $req | % Content
```

Expected:
- Recent timestamps.
- Real provider-backed records (Polygon/Finnhub/calendar macro where applicable).

---

## 5) Verify Codex API surface

### 5.1 Strategy types and instances

```powershell
iwr http://localhost:8081/api/v1/strategy-types | % Content
iwr http://localhost:8081/api/v1/instances | % Content
```

### 5.2 Trigger a backtest run

```powershell
$body = @{
  strategyId = "rsi_momentum_v1"
  from = "2025-01-01"
  to = "2025-01-31"
  symbolsOverride = @("SPY")
} | ConvertTo-Json
iwr http://localhost:8081/api/v1/backtests/run -Method POST -ContentType "application/json" -Body $body | % Content
```

Expected:
- Response includes `runId`.

---

## 6) Verify trust gates and artifact generation

```powershell
iwr http://localhost:8081/api/v1/testing/recon/data -Method POST | % Content
iwr http://localhost:8081/api/v1/testing/recon/pnl -Method POST | % Content
iwr http://localhost:8081/api/v1/testing/failure-suite -Method POST | % Content
iwr http://localhost:8081/api/v1/testing/flatten-proof -Method POST | % Content
```

Check generated artifacts:
- `reports/data_recon/<YYYY-MM-DD>/summary.md`
- `reports/data_recon/<YYYY-MM-DD>/recon.csv`
- `reports/pnl_recon/<YYYY-MM-DD>/pnl_recon.md`
- `reports/pnl_recon/<YYYY-MM-DD>/fills.csv`
- `reports/failure_tests/<YYYY-MM-DD>/report.md`
- `reports/flatten/<YYYY-MM-DD>/proof.md`

---

## 7) Verify DB logging for analysis

Run these queries on Postgres:

```sql
SELECT COUNT(*) AS runs_count FROM runs;
SELECT COUNT(*) AS ai_decisions_count FROM ai_decisions;
SELECT COUNT(*) AS ai_acceptance_count FROM ai_decision_acceptance;
SELECT COUNT(*) AS test_runs_count FROM test_runs;
SELECT COUNT(*) AS gate_status_count FROM gate_status;
SELECT COUNT(*) AS audit_events_count FROM audit_events;
```

Expected:
- `runs`, `ai_decisions`, `ai_decision_acceptance`, `test_runs`, and `gate_status` should increase as you execute flows.
- `audit_events` may remain low until full instrumentation is added.

---

## 8) Paper-trading soak process

1. Keep IBKR in paper mode only.
2. Run daily for a fixed soak window.
3. Track:
   - Real-data freshness.
   - Signal-to-news linkage quality.
   - Execution outcomes and rejects.
   - Trust-gate outcomes and artifact quality.
4. Promote only after stable evidence.

---

## 9) Move to GCP after home verification

Recommended sequence:
1. Move Postgres to Cloud SQL.
2. Deploy `jax-trader` + `jax-research` to a VM or GKE.
3. Store API keys in Secret Manager.
4. Put HTTPS ingress in front of `8081`.
5. Re-run sections 3 through 7 exactly.

Do not switch deployment target until home-server checks are stable.

---

## 10) Optional follow-up work for "log everything"

Implement next:
1. Call `LogAuditEvent(...)` across orchestration, execution, backtest, and trust-gate flows.
2. Persist complete `order_intents` and `fills` lifecycle from order submission to final state.
3. Add a flow timeline query/view keyed by `flow_id` for one-click analysis.

