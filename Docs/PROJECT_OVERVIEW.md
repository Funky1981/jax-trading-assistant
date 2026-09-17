# Jax Trading Assistant — Project Overview

Jax is an evidence-first market research and trading decision platform. The repository contains a modular Go monolith, frontend read models, Postgres persistence, and explicit external service boundaries.

## Current truth

- Product direction: `Docs/JAX_PRODUCT_CHARTER.md`
- Roadmap sequence: `Docs/ROADMAP.md`
- Current status: `Docs/STATUS.md`
- Capability maturity: `Docs/CAPABILITY_MATRIX.md`
- Current implementation routing: `Docs/BUILD/CURRENT_PACKAGE.md`
- Documentation authority: `Docs/DOCUMENTATION-AUTHORITY.md`

PAPER-01 is implemented, PAPER-01A remediation is complete, and external re-review is required. It is exploratory and hypothetical. A demonstrated trading edge has not been established; PAPER-02, formal paper evidence, live-readiness, and Phase 13 are not authorized or are blocked as stated in the roadmap.

## Runtime topology

- `cmd/trader` — deterministic trader runtime and frontend-facing API;
- `cmd/research` — research, orchestration, replay, and memory runtime;
- `services/ib-bridge` — explicit Interactive Brokers boundary;
- `frontend` — operator dashboard;
- `db/postgres/migrations` — runtime schema history.

## Guardrails

Trader behaviour remains deterministic. Research and AI components cannot create execution authority. Human approval and explicit safety gates remain required. No documentation reset changes runtime code or immutable validation results.

## Setup and operations

- `Docs/SETUP/QUICKSTART.md`
- `Docs/SETUP/IB_GUIDE.md`
- `Docs/OPERATIONS/OPERATIONS.md`
- `Docs/USER_GUIDES/USER_GUIDE.md`
- `Docs/TESTING/LOCAL_PAPER_TRADING_TESTING.md`
