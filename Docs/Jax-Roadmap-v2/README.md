# Jax Roadmap v2 — Consolidated Pack

> **SUPPORTING CURRENT PACKAGE DETAIL** — The authoritative human-readable
> roadmap is `Docs/ROADMAP.md`. Use this pack for detailed phase briefs, gates,
> package acceptance records and evidence navigation; do not use it as a second
> roadmap authority.

Start with `JAX-ROADMAP-OVERVIEW.md`, then read the active governance files
and the current phase `README.md`, `GATE.md` and first incomplete package.

Detailed package decisions are recorded in `ROADMAP-DECISION-LOG.md`; the
current autonomous package is recorded in `NEXT-WORK-PACKAGE.md`. The
human-facing current status is summarized in `Docs/ROADMAP.md`.

This pack is intentionally implementation-oriented. Each phase contains:
- `README.md` — purpose, prerequisites, scope and exit gate.
- one or more `WP-*.md` files — independently reviewable Codex work packages.
- `GATE.md` — evidence required before the next phase can begin.

Cross-cutting governance lives in `governance/`. Research and reference material lives in `references/`.

This roadmap uses Fincept Terminal intensively as a capability/reference system, while explicitly avoiding dependency on Fincept source code. Every borrowed idea must pass implementation-maturity, licensing, fit and alternative-library checks.

## Active execution model

From Phase 04 onward Jax uses Autonomous Development Mode. Codex implements
one bounded work package at a time, verifies and self-reviews it, creates a
bounded local commit, then continues through the authorised phase. External
technical-lead review normally occurs at the phase boundary. Legacy package
files may still contain per-package external-stop wording; the active
governance files supersede that cadence wording while package acceptance
criteria remain binding.

## Model routing

- GPT-5.6 Luna — default implementation and verification model.
- GPT-5.6 Sol — consequential architecture decisions and phase-level review.
- GPT-5.6 Terra — optional bounded escalation when Luna materially struggles.

## Current programme position

- Phases 00–03: `COMPLETE / GO`
- Phase 04: `COMPLETE / GO`
- Phase 05: `COMPLETE / GO`
- Phase 06: `COMPLETE / GO`
- Phase 07: `COMPLETE / GO`
- Phase 08: `COMPLETE / GO`
- Phase 09: `COMPLETE / GO`
- Phase 10: `COMPLETE / GO`
- Phase 11: `COMPLETE / GO`
- Current package: `PHASE 12 — REAL HYP-EVENT-001A SCIENTIFIC EVALUATION`
- Phase 12: `COMPLETE / CONDITIONAL GO — SCIENTIFIC EVALUATION COMPLETE; PROMOTION CLOSED`
- Phase 13: `NOT STARTED`

`HYP-EVENT-001A` has completed its real historical scientific evaluation. The
private result artifact contains 939 event-time direction records, frozen
development/validation selection, one formal 2024 OOS score and retained
falsification results. The conditioned comparison remains research-only and
promotion is closed because the 2025 holdout is sealed and survivorship is
unresolved. WP-12.01 was assessed without adopting a new runtime; Phase 13 is
not started.


## 2026-08-10 integrated roadmap change

This consolidated pack includes the LLM token/cost/context-efficiency requirements discovered during Phase 00 hosted-model evaluation. These requirements are integrated into the existing Phase 06 and Phase 08 folders; there are **no duplicate `06-*` or `08-*` phase folders**.

Key cross-cutting documents:
- `ROADMAP-CHANGE-2026-08-10-LLM-COST-CONTEXT-EFFICIENCY.md`
- `references/LLM-COST-CONTEXT-EFFICIENCY.md`
- `governance/LLM-COST-BUDGET-GATE.md`
- `06-research-recommendation-engine/LLM-CONTEXT-BUDGET-REQUIREMENTS.md`
- `08-controlled-ai-tools-durable-research-agents/LLM-COST-CONTEXT-EFFICIENCY-REQUIREMENTS.md`

`MANIFEST.sha256` fingerprints every file in this pack except the manifest itself.
