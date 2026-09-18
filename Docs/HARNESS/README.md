# Jax Harness Documentation

Status: **HARNESS-02 IMPLEMENTED / EXTERNAL REVIEW REQUIRED**.

This directory is the canonical documentation surface for the Jax harness
programme. HARNESS-01 adds versioned, dependency-light foundation contracts and
deterministic tests; HARNESS-02 adds an offline, deterministic
retrieval/context-builder package. It does not call a model, integrate with
JaxMind, execute evaluation, change memory behavior, or enter a trading path.

## Canonical documents

- [Harness Architecture Specification](ARCHITECTURE.md) — the target contract
  for the engineering harness and the separate Jax runtime/research harness.
- [Harness Evaluation Specification](EVALUATION.md) — scenarios, metrics,
  ablations, provenance reconstruction, and future laboratory gates.
- [Harness Implementation Roadmap](ROADMAP.md) — gated packages from
  HARNESS-01 through conditional advanced orchestration.
- [Engineering Harness Gap Analysis](ENGINEERING-HARNESS-GAP-ANALYSIS.md) —
  the current repository audit and durable documentation gaps.

## HARNESS implementation surface

The foundation contracts live in `internal/modules/harnesscontracts`. The
package uses only the Go standard library and is intentionally not imported by
`cmd/trader`. It provides versioned validation for research objectives, task
state, context packages, budgets, canonical evidence and memory references,
checkpoints, tool records, structured outputs, evaluator results, provenance,
harness runs, engineering state, verification plans, and clean-exit reports.

`internal/modules/contextbuilder` provides read-only retriever interfaces,
explicit contradiction retrieval, deterministic multi-factor selection,
counter-evidence budget protection, temporal/as-of filtering, omission and
deferred-reference records, and an auditable `BuildReport`. It currently has
offline fixture adapters only; no production evidence or memory adapter was
forced into this package.

The pre-existing `internal/modules/harness` package is a separate legacy
advisory/chat implementation. HARNESS-01 does not rename, refactor, integrate,
or reinterpret that package; it is not the foundation contract package.

## Authority and reconciliation

`Docs/ROADMAP.md` remains the single authoritative product roadmap. This
directory supplies the detailed HARNESS architecture and sequencing referenced
by that roadmap. `Docs/CONTEXT_ENGINEERING_ROADMAP_RECORD.md` remains retained
history and is reconciled to this package; it is no longer a competing detailed
architecture.

The active Jax topology remains the ADR-0012 two-runtime modular monolith:
`cmd/trader` is deterministic and safety-critical, while `cmd/research` owns
research/orchestration paths. Harness work must respect that boundary.

## Non-authorizations

HARNESS-02 does not authorize HARNESS-03, task-state persistence, compaction,
JaxMind calls, evaluator execution, orchestration, PAPER-02 runtime or context
changes, formal forward paper, live trading, Phase 13, or any multi-agent
orchestration. Each later package requires its own gate and review.
