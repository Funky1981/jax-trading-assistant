# Documentation Index

This directory contains the current operating documentation for the active runtime (`cmd/trader` + `cmd/research`).

## Core Docs (Current)

- `PROJECT_OVERVIEW.md`
- `ARCHITECTURE.md`
- `ARCHITECTURE_DIAGRAM.md`
- `QUICKSTART.md`
- `OPERATIONS.md`
- `DEBUGGING.md`
- `db-setup.md`
- `IB_GUIDE.md`
- `USER_GUIDE.md`
- `STATUS.md`
- `ROADMAP.md`
- `TODO.md`
- `TEST_PLAN.md`
- `AUDIT_TRAIL.md`
- `UAT_PAPER_TRADING.md`
- `PRODUCTION_READINESS.md`
- `SLO_ALERTS.md`
- `INCIDENT_RUNBOOK.md`
- `CONTRIBUTING.md`

## Active Folders

- `plans/` - active implementation plans and completed plan evidence
- `runs/` - generated run evidence and readiness notes

## ADR

- `ADR-0012-two-runtime-modular-monolith.md` (active architecture decision)

Supporting ADR planning and historical rollout notes are archived in:
- `Docs/archive/adrs/supporting/`

## Archive Layout

- `Docs/archive/runtime-history/` - historical phase implementation notes
- `Docs/archive/reports/` - completion/status reports kept for traceability
- `Docs/archive/programs/` - legacy planning workstreams (`masterplan`, `upgrades`, `codex`, `ejlayer`)
- `Docs/archive/evidence/` - historical run/UAT output
- `Docs/archive/diagrams/` - non-canonical legacy diagrams
- `Docs/archive/docs-original/` - preserved snapshot of older documentation set

Archive guidance:
- Archived docs are preserved for traceability, not as operating runbooks.
- Expect historical references to removed services such as `hindsight`, `jax-memory`, older bank names like `trade_decisions`, and pre-consolidation paths.
- When archive content conflicts with current runtime behavior, follow the active docs in this directory.

Recently archived from the top level:
- `Docs/plans/Completed/jax-harness-full-package/`
- `Docs/archive/programs/codex/jax-assistant-approval-continuation-v1/`
- `Docs/archive/programs/codex/jax-paper-trading-finish-plan-phased-v1/`
- `Docs/archive/programs/codex/jax-etf-news-research-build-pack/`
- `Docs/archive/programs/codex/jax-etf-news-strategy-pack/`
- `Docs/archive/programs/codex/superpowers-execution-notes/`
- `Docs/archive/reports/deep-research-report.md`
- `Docs/archive/evidence/manual-tests/`
- `Docs/archive/evidence/runs/20260305/`

Use `Docs/archive/README.md` for archive notes.
