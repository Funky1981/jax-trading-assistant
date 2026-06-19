# Incident Runbook

Use this runbook for production-impacting issues in trading, data integrity, or decision traceability.

## Severity Levels

1. `SEV-1`: live trading safety or audit integrity at risk.
2. `SEV-2`: paper/live trading degraded with partial service availability.
3. `SEV-3`: non-critical degradation without execution risk.

## Immediate Actions (SEV-1 / SEV-2)

1. Freeze execution:

```powershell
# live mode kill switch
$env:ALLOW_LIVE_TRADING="false"
```

2. Confirm service state:

```powershell
Invoke-RestMethod http://localhost:8081/health
Invoke-RestMethod http://localhost:8091/health
Invoke-RestMethod http://localhost:8092/health
Invoke-RestMethod http://localhost:8093/health
```

3. Capture latest platform evidence:

```powershell
.\scripts\test-platform.ps1 -Mode quick
```

4. Pull logs:

```powershell
docker compose logs --tail=300 jax-trader
docker compose logs --tail=300 jax-research
docker compose logs --tail=300 ib-bridge
```

## Scenario Playbooks

### A) Provenance Gate Failure

1. Query `testing/status` and identify failing gate detail.
2. Run provenance SQL from `Docs/AUDIT_TRAIL.md`.
3. Block artifact promotions until pass is restored.
4. Record incident with affected `run_id` and `flow_id`.

### B) Broker/Execution Failure Spike

1. Confirm IB bridge health.
2. Verify broker provider configuration and runtime mode.
3. Stop new order submissions until failure rate is below threshold.
4. Reconcile pending orders and update incident notes.

### C) Audit Trail Gap (missing run/flow linkage)

1. Query missing `run_id` or `flow_id` rows.
2. Correlate with service logs by timestamp window.
3. Mark affected trades as audit-incomplete.
4. Open remediation task before next release.

## Recovery and Closeout

1. Verify health and gate status are green.
2. Run full platform validation:

```powershell
.\scripts\test-platform.ps1 -Mode full
```

3. Add incident summary:
   - trigger and detection time
   - impacted runs/trades
   - root cause
   - corrective actions
4. Update checklist in `Docs/PRODUCTION_READINESS.md` if policy changed.

