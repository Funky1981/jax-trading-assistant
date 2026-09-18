# Jax Harness Documentation

Status: **HARNESS-01 IMPLEMENTED / EXTERNAL REVIEW REQUIRED**.

This directory is the canonical documentation surface for the Jax harness
programme. HARNESS-01 adds versioned, dependency-light foundation contracts and
deterministic tests; it does not implement retrieval, model calls, evaluation,
memory behavior, or any trading path.

## Canonical documents

- [Harness Architecture Specification](ARCHITECTURE.md) — the target contract
  for the engineering harness and the separate Jax runtime/research harness.
- [Harness Evaluation Specification](EVALUATION.md) — scenarios, metrics,
  ablations, provenance reconstruction, and future laboratory gates.
- [Harness Implementation Roadmap](ROADMAP.md) — gated packages from
  HARNESS-01 through conditional advanced orchestration.
- [Engineering Harness Gap Analysis](ENGINEERING-HARNESS-GAP-ANALYSIS.md) —
  the current repository audit and durable documentation gaps.

## HARNESS-01 implementation surface

The foundation contracts live in `internal/modules/harnesscontracts`. The
package uses only the Go standard library and is intentionally not imported by
`cmd/trader`. It provides versioned validation for research objectives, task
state, context packages, budgets, canonical evidence and memory references,
checkpoints, tool records, structured outputs, evaluator results, provenance,
harness runs, engineering state, verification plans, and clean-exit reports.

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

HARNESS-01 does not authorize HARNESS-02, retrieval, context construction,
JaxMind calls, compaction, evaluator execution, orchestration, PAPER-02 runtime
changes, formal forward paper, live trading, Phase 13, or any multi-agent
orchestration. Each later package requires its own gate and review.
