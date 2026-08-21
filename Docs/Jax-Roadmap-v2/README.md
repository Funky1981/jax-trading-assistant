# Jax Roadmap v2 — Consolidated Pack

Start with `JAX-ROADMAP-OVERVIEW.md`.

Current authority is recorded in `ROADMAP-DECISION-LOG.md`; the first incomplete package awaiting technical-lead authorization is recorded in `NEXT-WORK-PACKAGE.md`.

This pack is intentionally implementation-oriented. Each phase contains:
- `README.md` — purpose, prerequisites, scope and exit gate.
- one or more `WP-*.md` files — independently reviewable Codex work packages.
- `GATE.md` — evidence required before the next phase can begin.

Cross-cutting governance lives in `governance/`. Research and reference material lives in `references/`.

This roadmap uses Fincept Terminal intensively as a capability/reference system, while explicitly avoiding dependency on Fincept source code. Every borrowed idea must pass implementation-maturity, licensing, fit and alternative-library checks.


## 2026-08-10 integrated roadmap change

This consolidated pack includes the LLM token/cost/context-efficiency requirements discovered during Phase 00 hosted-model evaluation. These requirements are integrated into the existing Phase 06 and Phase 08 folders; there are **no duplicate `06-*` or `08-*` phase folders**.

Key cross-cutting documents:
- `ROADMAP-CHANGE-2026-08-10-LLM-COST-CONTEXT-EFFICIENCY.md`
- `references/LLM-COST-CONTEXT-EFFICIENCY.md`
- `governance/LLM-COST-BUDGET-GATE.md`
- `06-research-recommendation-engine/LLM-CONTEXT-BUDGET-REQUIREMENTS.md`
- `08-controlled-ai-tools-durable-research-agents/LLM-COST-CONTEXT-EFFICIENCY-REQUIREMENTS.md`

`MANIFEST.sha256` fingerprints every file in this pack except the manifest itself.
