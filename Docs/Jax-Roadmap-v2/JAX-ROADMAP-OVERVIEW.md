# Jax Roadmap v2 — Master Overview

## Purpose

Jax should become a personal, evidence-first market research and trading decision platform that can ingest trustworthy information, resolve what assets events affect, perform reproducible quantitative analysis, form explainable recommendations, validate those recommendations historically, account for portfolio risk, and only then support controlled paper execution. Optional live execution is a final, separately gated capability rather than the definition of success.

This roadmap replaces the previous feature-by-feature plan. The new approach is **reference-led engineering**: define the Jax requirement, study mature systems such as Fincept Terminal plus specialist libraries/frameworks, then implement the smallest Jax-specific capability that satisfies the requirement.

## Current starting point

Jax is not greenfield. The public repository currently documents a modular-monolith architecture with `cmd/trader` and `cmd/research` runtimes, Postgres-backed state, research/backtest pathways, artifact trust gates, frontend APIs, an IB bridge and an AI service boundary. The latest project handover supplied for this roadmap places current work at the frozen v4 issuer-recognition evaluation gate on the `capability-reset` branch. Before any roadmap package runs, Codex must verify the actual branch, HEAD, working tree, upstream and ahead/behind state.

## Current programme status

As of 2026-08-21, Phase 00 is **GO**. The accepted architecture is `Event -> typed causal attribution -> deterministic policy -> DIRECT / PROXY / UNRESOLVED -> deterministic resolver`. Phase 00 model evaluation is closed. Luna remains the default runtime model and Terra is retained only as a validated higher-capability future option; no escalation implementation is authorized now.

All Phase 00 packages, including the evidence-baseline closure in WP-00.04, are complete. The exact first incomplete package is Phase 01 `WP-01.01 - Inventory existing Jax domain contracts before adding new ones`. It is identified for technical-lead authorization but has not started. See `ROADMAP-DECISION-LOG.md` and `NEXT-WORK-PACKAGE.md`.

## Non-negotiable principles


- Evidence before inference: models interpret evidence; they do not manufacture facts or deterministic metrics.
- Raw before normalized: retain immutable provider payloads or immutable references before transformation.
- Provenance everywhere: outputs must identify inputs, source/provider, versions, timestamps, model/prompt/algorithm and validation state.
- Deterministic gates around stochastic components: AI may propose or interpret; safety, eligibility, sizing and execution gates are deterministic.
- Replayability: a historical decision should be reconstructable from persisted inputs and versions.
- AI cost/context is a controlled resource: avoid unnecessary model calls, bound context, exploit safe provider caching/reuse, route to the least expensive proven-adequate model, and record token/cost evidence without weakening provenance or correctness.
- Best tool for the job: no language restriction, but each language/runtime must map to a bounded architectural responsibility.
- Build vs adopt: use established libraries/services where they solve commodity problems well; build Jax-specific differentiation around evidence synthesis, validation and controlled decision support.
- Paper first: live execution is optional and remains a separately gated capability.
- No silent progression: after every roadmap work package, Codex stops and produces the standard review handover. The architecture reviewer returns GO / CONDITIONAL GO / NO-GO / ROADMAP CHANGE.
- Fincept is a reference system, not a dependency or code donor. Concepts must be independently reimplemented or satisfied by external libraries/services after licensing and maturity review.


## Programme map

| Phase | Outcome | Fincept / external reference focus |
|---|---|---|
| 00 | Prove issuer recognition and deterministic asset resolution | Fincept instrument concepts only; preserve current work |
| 01 | Canonical contracts, provenance and audit foundation | Fincept bounded contexts, Alpha Arena replay/audit |
| 02 | Provider/normalization/data-health platform | DataHub, normalization, provider patterns |
| 03 | Core market/corporate/macro evidence | Fincept connectors; SEC, FRED, market-data providers |
| 04 | World Monitor becomes corroborated event intelligence | news NLP/cluster/correlation, ACLED, maritime, predictions |
| 05 | Deterministic quant core | Fincept analytics; NumPy/SciPy/skfolio/Riskfolio candidates |
| 06 | Evidence-backed research and recommendations | structured synthesis, contradiction/unknown handling, bounded evidence context |
| 07 | Evaluation, replay and backtesting | Alpha Arena, LEAN, VectorBT, event replay |
| 08 | Controlled AI tools and durable research agents | Fincept MCP/agentic research, tool permissions, model routing/cache/budget controls |
| 09 | Portfolio intelligence and deterministic risk | portfolio analytics, risk limits, exposure |
| 10 | Workflow, HITL and operational safety | workflow executor, confirmation, audit, circuit breakers |
| 11 | High-fidelity paper trading | broker abstraction, paper venue, reconciliation |
| 12 | Advanced quant research | Qlib/RD-Agent, factor research, drift/model evaluation |
| 13 | Optional live execution | broker adapters, explicit approval, reconciliation |

## Maturity milestones

**M1 — Understand:** Phase 00 complete. Jax reliably identifies the correct issuer/instrument or explicitly remains unknown.

**M2 — Observe:** Phases 01–04 complete. Jax can gather, normalize, validate and correlate real evidence with source health and provenance.

**M3 — Analyse:** Phase 05 complete. Jax calculates reproducible quantitative and risk context from trusted data.

**M4 — Recommend:** Phase 06 complete. Jax can produce evidence-backed WATCH / NO_TRADE / CANDIDATE-style recommendations with bull/bear cases, unknowns and invalidation conditions.

**M5 — Prove:** Phase 07 complete. Recommendation logic is evaluated through frozen benchmarks, historical replay and outcome tracking.

**M6 — Research:** Phase 08 complete. AI can use controlled tools to perform durable, checkpointed research without bypassing deterministic gates.

**M7 — Portfolio-aware:** Phases 09–10 complete. Recommendations are evaluated against holdings, concentration, risk budgets and human approval controls.

**M8 — Paper-capable:** Phase 11 complete. Approved recommendations can be simulated with realistic costs, slippage, state and reconciliation.

**M9 — Advanced:** Phase 12 selectively adds only statistically validated factor/ML capabilities.

**M10 — Optional live:** Phase 13 is considered only after sustained paper evidence and explicit user decision.

## Review workflow

Every numbered work package follows:

`Roadmap package -> Codex implementation -> verification -> REVIEW HANDOVER -> STOP -> architecture review -> GO / CONDITIONAL GO / NO-GO / ROADMAP CHANGE`

Codex must not start a subsequent package based on its own confidence.

See `governance/CODEX-REVIEW-HANDOVER.md` and `governance/GO-NO-GO-PROCESS.md`.

## How to use this pack

1. Open the current phase `README.md`.
2. Select the first incomplete work package.
3. Give Codex that package file plus `governance/CODEX-OPERATING-RULES.md`.
4. Codex completes only that package, verifies it, commits only if instructed by the user, then returns the standard review handover.
5. Give that handover to the architecture reviewer.
6. Proceed only after a GO or accepted CONDITIONAL GO.

The roadmap is controlled but not immutable. If implementation evidence invalidates an assumption, use **ROADMAP CHANGE** rather than forcing later phases to fit a bad design.


## Cross-cutting LLM efficiency requirement

The 2026-08-10 roadmap change makes LLM cost and context efficiency explicit rather than implicit. Phase 00 frozen capability benchmarks continue to invoke models genuinely; application-level result caching is not permitted to contaminate those measurements. Production research work later introduces bounded context construction, exact-result reuse with complete validity keys, provider prompt-cache awareness, model routing/escalation, durable-agent context compaction, spend limits and usage telemetry. See `ROADMAP-CHANGE-2026-08-10-LLM-COST-CONTEXT-EFFICIENCY.md`.
