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

## Assistant Harness Runtime

The assistant chat API now runs through the read-only research harness by default.
Configure it explicitly in each environment:

```powershell
$env:HARNESS_ENABLED="true"
$env:HARNESS_SHADOW_MODE="false"
$env:HARNESS_SESSION_RATE_LIMIT_PER_MINUTE="20"
```

Behavior by mode:

- `research`: widest advisory surface; all registered read-only tools are available.
- `paper`: weak-inference tools are blocked; advisory answers stay stricter.
- `live`: only hard internal evidence tools with acceptable freshness remain available.

Shadow mode keeps the existing chat response path and runs the harness in the background for trace and validator logging only:

```powershell
$env:HARNESS_SHADOW_MODE="true"
```

## Assistant Traceability

Run the assistant migrations before enabling traces in a new environment:

```powershell
$env:DATABASE_URL = "postgresql://jax:jax@localhost:5433/jax"
go run ./tools/cmd/migrate
```

Assistant endpoints:

- `POST /api/v1/chat`
- `GET /api/v1/chat?session=<id>`
- `GET /api/v1/chat/tools`
- `GET /api/v1/chat/traces/<traceId>`

Operational checks:

```powershell
Invoke-RestMethod http://localhost:8081/api/v1/chat/tools
Invoke-RestMethod http://localhost:8081/api/v1/chat/traces/<traceId>
```

`/api/v1/chat/tools` now returns:

- mode
- harness enabled state
- shadow mode state
- session rate limit
- per-tool evidence level, freshness, and policy availability

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

## ETF Phase-1 Paper Trading

ETF phase 1 is approval-only and paper-only.

Operator checks:

```powershell
Invoke-RestMethod http://localhost:8081/api/v1/instruments/etfs
Invoke-RestMethod http://localhost:8081/api/v1/trading/pilot-status
Invoke-RestMethod http://localhost:8081/api/v1/testing/readiness
```

Rules enforced by runtime:

- Approved ETF list lives in `config/etf-instruments.json`.
- Manual ETF entry orders through `/api/v1/broker/orders` and `/api/v1/broker/orders/bracket` are blocked.
- Legacy signal approvals through `/api/v1/signals/{id}/approve` are blocked for catalog ETF symbols.
- ETF entries must come from an approved candidate and generated execution instruction.
- ETF broker submission is rejected when quote age exceeds 60 seconds, spread exceeds 10 bps, bid/ask or sizes are missing, the request is outside US RTH, stop loss is missing, or runtime mode is not `paper`.
- Manual close, cancel, and protection paths remain available for managing existing exposure.

Approve an ETF candidate only from the Approvals page or the candidate approval API after checking the row-level ETF metadata. The expected approved path is:

1. Candidate is created with ETF policy metadata.
2. Operator approval rechecks the ETF catalog before creating execution instructions.
3. Execution rechecks quote freshness, spread, RTH, stop loss, flatten-by-close, and paper mode immediately before broker submission.

Investigate blocked ETF entries with:

```sql
SELECT id, symbol, status, metadata
FROM candidate_trades
WHERE symbol IN ('SPY','QQQ','DIA','IWM','XLK','XLF','XLE','SMH','SOXX','TLT','GLD')
ORDER BY created_at DESC
LIMIT 20;

SELECT id, candidate_id, event_type, detail, created_at
FROM candidate_events
ORDER BY created_at DESC
LIMIT 20;
```

Revoke a single ETF by changing its `eligibility_state` to `revoked` or adding an exclusion in `config/etf-instruments.json`, then redeploy/restart `jax-trader`. The catalog hash in new audit metadata should change after restart.

ETF rollout readiness remains `not_ready` until validation, UAT, pilot, and owner sign-offs are explicitly recorded with the `ETF_PHASE1_*` environment variables documented in `Docs/TESTING/UAT_PAPER_TRADING.md`.

Capture pilot evidence before sign-off with:

```powershell
.\scripts\etf-paper-pilot-evidence.ps1
```

To halt ETF trading immediately, disable execution or revoke the relevant strategy/artifact approval, then restart `jax-trader`:

```powershell
$env:EXECUTION_ENABLED="false"
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
