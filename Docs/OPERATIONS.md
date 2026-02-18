# Operations Runbook

## Starting the Platform

```powershell
# Ensure IB Gateway is running on host (port 4002 paper / 4001 live)
docker compose up -d

# Verify health
docker compose ps
Invoke-RestMethod http://localhost:8081/health   # jax-api
Invoke-RestMethod http://localhost:8100/health   # trader
Invoke-RestMethod http://localhost:8091/health   # research
```

## Approving a New Strategy Artifact

When the research runtime generates a new backtest artifact it lands in `DRAFT` state. Promote it through the approval gate:

```powershell
# 1. List draft artifacts
$env:DATABASE_URL = "postgresql://jax:jax@localhost:5433/jax"
psql $env:DATABASE_URL -c "SELECT artifact_id, strategy_name, created_at FROM strategy_artifacts WHERE state = 'DRAFT' ORDER BY created_at DESC LIMIT 5;"

# 2. Review and approve
go run cmd/artifact-approver/main.go `
    -id "<artifact_id>" `
    -approver "your.name" `
    -type TECHNICAL `
    -notes "Reviewed backtest — Sharpe > 1.2, drawdown < 15%"

# 3. Restart trader to load new artifact
docker compose restart jax-trader
```

## Revoking a Strategy (Emergency)

```powershell
$env:DATABASE_URL = "postgresql://jax:jax@localhost:5433/jax"

# Revoke immediately — trader will stop using on next restart
psql $env:DATABASE_URL -c "UPDATE strategy_artifacts SET state = 'REVOKED' WHERE artifact_id = '<artifact_id>';"

docker compose restart jax-trader
```

## Rolling Back to a Previous Artifact

```powershell
$env:DATABASE_URL = "postgresql://jax:jax@localhost:5433/jax"

# Promote previous artifact back to APPROVED
psql $env:DATABASE_URL -c "UPDATE strategy_artifacts SET state = 'APPROVED' WHERE artifact_id = '<previous_artifact_id>';"

# Revoke the bad one
psql $env:DATABASE_URL -c "UPDATE strategy_artifacts SET state = 'REVOKED' WHERE artifact_id = '<current_artifact_id>';"

docker compose restart jax-trader
```

## Monitoring

- **Grafana**: http://localhost:3001 (admin / password from `.env`)
- **Prometheus**: http://localhost:9090
- **Key metrics**: trader executions/min, position sizes, signal latency

## Viewing Audit Trail

Every trade is linked to the artifact that authorised it:

```sql
SELECT t.id, t.symbol, t.action, t.quantity, t.created_at,
       a.artifact_id, a.strategy_name, a.hash
FROM trades t
JOIN strategy_artifacts a ON t.artifact_id = a.id
ORDER BY t.created_at DESC
LIMIT 20;
```

## Troubleshooting

### Trader won't start — no approved artifacts

```powershell
$env:DATABASE_URL = "postgresql://jax:jax@localhost:5433/jax"
psql $env:DATABASE_URL -c "SELECT artifact_id, state FROM strategy_artifacts;"
# If empty or all DRAFT, approve one (see above)
```

### jax-api migration fails on startup

Migrations are idempotent (IF NOT EXISTS guards). If a dirty state is left:

```powershell
# Check dirty state
docker exec jax-tradingassistant-postgres-1 psql -U jax -d jax -c "SELECT version, dirty FROM schema_migrations ORDER BY version;"

# Clear dirty version (replace N with the dirty version number)
docker exec jax-tradingassistant-postgres-1 psql -U jax -d jax -c "DELETE FROM schema_migrations WHERE version = N;"

# Rebuild and restart
docker compose build --no-cache jax-api
docker compose up -d jax-api
```

### Determinism test failing

```powershell
go test -v ./internal/modules/backtest/... -run TestEngine_Deterministic
# Check for: time.Now() usage, map iteration order, external API calls without mocks
```
