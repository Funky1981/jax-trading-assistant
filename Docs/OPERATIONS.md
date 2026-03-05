# Operations Runbook

## Start Platform

```powershell
docker compose up -d
docker compose ps
Invoke-RestMethod http://localhost:8081/health   # jax-trader frontend API
Invoke-RestMethod http://localhost:8100/health   # jax-trader runtime
Invoke-RestMethod http://localhost:8091/health   # jax-research
Invoke-RestMethod http://localhost:8092/health   # ib-bridge
Invoke-RestMethod http://localhost:8093/health   # agent0-service
```

## Runtime Mode Guard

Always set runtime mode explicitly outside local development:

```powershell
$env:JAX_RUNTIME_MODE="paper"   # or live/research
$env:JAX_REQUIRE_EXPLICIT_RUNTIME_MODE="true"
```

For `live` mode, execution must be intentionally enabled:

```powershell
$env:ALLOW_LIVE_TRADING="true"
```

## Artifact Approval Flow

```powershell
$env:DATABASE_URL = "postgresql://jax:jax@localhost:5433/jax"

# List draft artifacts + current approval state
psql $env:DATABASE_URL -c @"
SELECT a.artifact_id, a.strategy_name, ap.state, a.created_at
FROM strategy_artifacts a
JOIN artifact_approvals ap ON ap.artifact_id = a.id
WHERE ap.state = 'DRAFT'
ORDER BY a.created_at DESC
LIMIT 10;
"@

# Promote artifact through API (example: DRAFT -> VALIDATED / APPROVED)
Invoke-RestMethod -Method Post -Uri "http://localhost:8081/api/v1/artifacts/<uuid>/promote" `
  -ContentType "application/json" `
  -Body '{"to_state":"APPROVED","promoted_by":"ops","reason":"manual approval"}'
```

## Emergency Revoke / Rollback

```powershell
$env:DATABASE_URL = "postgresql://jax:jax@localhost:5433/jax"

# Revoke one artifact
psql $env:DATABASE_URL -c @"
UPDATE artifact_approvals ap
SET state = 'REVOKED',
    state_changed_by = 'ops',
    state_change_reason = 'emergency revoke',
    state_changed_at = NOW()
FROM strategy_artifacts a
WHERE ap.artifact_id = a.id
  AND a.artifact_id = '<artifact_id>';
"@

docker compose restart jax-trader
```

## Monitoring

- Grafana: `http://localhost:3001`
- Prometheus: `http://localhost:9090`
- Service logs:

```powershell
docker compose logs -f jax-trader
docker compose logs -f jax-research
docker compose logs -f ib-bridge
docker compose logs -f agent0-service
docker compose logs -f hindsight
```

SLO targets and alert thresholds are defined in `Docs/SLO_ALERTS.md`.
Incident response flow is defined in `Docs/INCIDENT_RUNBOOK.md`.

## Audit Trail Query

```sql
SELECT t.id, t.symbol, t.side, t.quantity, t.created_at,
       a.artifact_id, a.strategy_name, a.hash
FROM trades t
JOIN strategy_artifacts a ON t.artifact_id = a.id
ORDER BY t.created_at DESC
LIMIT 20;
```

For decision-level and gate-level traceability, use `Docs/AUDIT_TRAIL.md`.

## Release Gate

Before production promotion, complete:

1. `.\scripts\test-platform.ps1 -Mode full`
2. Production checklist in `Docs/PRODUCTION_READINESS.md`
3. Audit trail verification from `Docs/AUDIT_TRAIL.md`

## Common Failures

### Trader has no approved artifacts

```powershell
$env:DATABASE_URL = "postgresql://jax:jax@localhost:5433/jax"
psql $env:DATABASE_URL -c @"
SELECT a.artifact_id, ap.state, ap.validation_passed, ap.validation_report_uri
FROM strategy_artifacts a
JOIN artifact_approvals ap ON ap.artifact_id = a.id
ORDER BY a.created_at DESC
LIMIT 20;
"@
```

### Migration drift / dirty migration state

```powershell
docker compose exec postgres psql -U jax -d jax -c "SELECT version, dirty FROM schema_migrations ORDER BY version;"
```
