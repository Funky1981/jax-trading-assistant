# Documentation Index

This directory contains the current operating documentation for the active runtime (`cmd/trader` + `cmd/research`).

## Jax Product and Delivery Truth

- `JAX_PRODUCT_CHARTER.md` is the product truth for Jax.
- `CAPABILITY_MATRIX.md` tracks what Jax can and cannot do.
- `PHASE_CONTRACTS/` controls phase delivery.
- `PROJECT_MANAGEMENT/` contains the project-manager process.
- `TRADING_BRAIN/` contains decisioning design.
- `STRATEGIES/SWING_TRADING/` is the first active strategy area.
- `RESEARCH/`, `PAPER_TRADING/`, and `MEMORY_AND_REVIEW/` support evidence, paper trading, and learning.

## Core Docs (Current)

- `JAX_PRODUCT_CHARTER.md`
- `CAPABILITY_MATRIX.md`
- `PROJECT_OVERVIEW.md`
- `ARCHITECTURE.md`
- `STATUS.md`
- `ROADMAP.md`
- `CONTRIBUTING.md`

## Active Folders

- `ARCHITECTURE/` - architecture diagrams and ADRs that support `ARCHITECTURE.md`
- `OPERATIONS/` - operations, debugging, readiness, and alerting docs
- `SETUP/` - setup, quickstart, database, and broker guide docs
- `TESTING/` - testing plans and profitability validation docs
- `USER_GUIDES/` - user-facing guides
- `RUNBOOKS/` - incident response and operational runbooks
- `AUDIT/` - audit trail docs
- `SECURITY/` - security docs and placeholders
- `PHASE_CONTRACTS/` - phase delivery contracts
- `PROJECT_MANAGEMENT/` - project-manager process and ProjectOS workflow docs
- `TRADING_BRAIN/` - decisioning design
- `STRATEGIES/` - strategy documentation, with `STRATEGIES/SWING_TRADING/` as the first active strategy area
- `RESEARCH/` - evidence and research process support
- `PAPER_TRADING/` - paper-trading process support
- `MEMORY_AND_REVIEW/` - learning and review support
- `plans/` - imported/archive planning packs and completed plan evidence
- `runs/` - generated run evidence and readiness notes

## ADR

- `ARCHITECTURE/ADR/ADR-0012-two-runtime-modular-monolith.md` (active architecture decision)

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
