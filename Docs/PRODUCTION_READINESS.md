# Production Readiness Checklist

This checklist is the release gate for moving from paper validation to production deployment.

## 1) Runtime Safety

- `JAX_RUNTIME_MODE` is explicitly set in every deployment target (`research`, `paper`, or `live`).
- `JAX_REQUIRE_EXPLICIT_RUNTIME_MODE=true` is set for all non-local environments.
- `config/providers.json` passes strict policy for the target mode.
- For `live` mode only: `ALLOW_LIVE_TRADING=true` is explicitly set and approved.

Validation command:

```powershell
go test ./libs/runtimepolicy ./libs/utcp ./cmd/trader ./cmd/research
```

## 2) CI and Release Gates

- CI passes for:
  - Go lint + format + tests
  - runtime policy package tests
  - frontend lint/typecheck/vitest
  - frontend Playwright e2e
- Golden workflow passes with no `continue-on-error` bypasses.

Validation command:

```powershell
.\scripts\test-platform.ps1 -Mode full
```

## 3) Data Integrity and Auditability

- No synthetic data in truth-path providers for `research`, `paper`, or `live`.
- `/api/v1/testing/status` shows provenance gate pass.
- Audit queries in `Docs/AUDIT_TRAIL.md` return complete trade-to-decision lineage.
- `flow_id` and `run_id` are present for operational traces.

## 4) Operations Readiness

- Alert thresholds are configured from `Docs/SLO_ALERTS.md`.
- On-call incident playbook is active from `Docs/INCIDENT_RUNBOOK.md`.
- Kill-switch procedure has been executed in staging at least once.
- Backup/restore run has been tested for the production database.

## 5) Security and Secrets

- Production secrets are loaded from the secret manager only.
- No placeholder credentials are used in runtime environments.
- JWT/CORS/rate-limit settings are reviewed and approved.

## 6) Sign-off

- Engineering sign-off
- Operations sign-off
- Risk/compliance sign-off

Release is blocked until all sections pass.

