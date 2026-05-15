# Production Readiness Checklist

This checklist is the release gate for moving from paper validation to production deployment.

## 1) Runtime Safety

- `JAX_RUNTIME_MODE` is explicitly set in every deployment target (`research`, `paper`, or `live`).
- `JAX_REQUIRE_EXPLICIT_RUNTIME_MODE=true` is set for all non-local environments.
- `config/providers.json` passes strict policy for the target mode.
- `HARNESS_ENABLED=true` is explicitly set for environments where assistant chat is expected to use the evidence harness.
- `HARNESS_SHADOW_MODE` is explicitly set to the intended rollout mode (`false` for primary path, `true` for log-only shadow mode).
- `HARNESS_SESSION_RATE_LIMIT_PER_MINUTE` is set to the approved session throttle for the environment.
- For `live` mode only: `ALLOW_LIVE_TRADING=true` is explicitly set and approved.
- ETF phase 1 is paper-only: `config/etf-instruments.json` must load successfully and all ETF live submissions must remain blocked.
- ETF entries must use the candidate approval workflow; manual ETF entry orders from the order ticket are not release-eligible.
- ETF thresholds are confirmed: quote age <= 60s, spread <= 10 bps, bid/ask size > 0, RTH only, stop loss required, flatten-by-close required.
- ETF launch evidence is visible in `/api/v1/testing/readiness` under `etfPhase1Readiness`.

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

When frontend API auth is enabled, `./scripts/test-platform.ps1` authenticates automatically before probing protected routes.

## 3) Data Integrity and Auditability

- No synthetic data in truth-path providers for `research`, `paper`, or `live`.
- `/api/v1/testing/status` shows provenance gate pass.
- Audit queries in `Docs/AUDIT_TRAIL.md` return complete trade-to-decision lineage.
- ETF audit metadata is present on candidate metadata, candidate events, execution audit events, and trade risk metadata for ETF orders.
- `flow_id` and `run_id` are present for operational traces.
- Assistant responses persist `trace_id` and `evidence_bundle`.
- `harness_traces` is populated for advisory answers and trace payloads are queryable.
- Trace redaction has been reviewed for secrets/tokens in the target deployment.

## 4) Operations Readiness

- Alert thresholds are configured from `Docs/SLO_ALERTS.md`.
- On-call incident playbook is active from `Docs/INCIDENT_RUNBOOK.md`.
- Kill-switch procedure has been executed in staging at least once.
- Backup/restore run has been tested for the production database.
- Assistant operators can retrieve `/api/v1/chat/tools` and `/api/v1/chat/traces/{traceId}` from the deployed frontend API.

## 5) Security and Secrets

- Production secrets are loaded from the secret manager only.
- No placeholder credentials are used in runtime environments.
- JWT/CORS/rate-limit settings are reviewed and approved.

## 6) Sign-off

- Engineering sign-off
- Operations sign-off
- Risk/compliance sign-off
- ETF phase-1 sign-off variables are set only after automated validation, operator UAT, and limited paper pilot evidence are reviewed:
  - `ETF_PHASE1_AUTOMATED_VALIDATION=passed`
  - `ETF_PHASE1_OPERATOR_UAT=passed`
  - `ETF_PHASE1_PAPER_PILOT_SIGNOFF=passed`
  - `ETF_PHASE1_ENGINEERING_SIGNOFF=true`
  - `ETF_PHASE1_OPERATIONS_SIGNOFF=true`
  - `ETF_PHASE1_TRADING_RISK_SIGNOFF=true`

As of 2026-05-14, the technical validation path is green on the Docker-backed paper stack. Release remains blocked only until the ETF phase-1 sign-off values above are explicitly approved and recorded.

Release is blocked until all sections pass.
