# Jax Harness Documentation

Status: **HARNESS-00 IMPLEMENTED / EXTERNAL REVIEW REQUIRED**.

This directory is the canonical documentation surface for the future Jax
harness programme. HARNESS-00 is architecture and roadmap documentation only;
it does not implement a runtime harness, change JaxMind, alter evidence or
memory retrieval, or change any trading path.

## Canonical documents

- [Harness Architecture Specification](ARCHITECTURE.md) — the target contract
  for the engineering harness and the separate Jax runtime/research harness.
- [Harness Evaluation Specification](EVALUATION.md) — scenarios, metrics,
  ablations, provenance reconstruction, and future laboratory gates.
- [Harness Implementation Roadmap](ROADMAP.md) — gated packages from
  HARNESS-01 through conditional advanced orchestration.
- [Engineering Harness Gap Analysis](ENGINEERING-HARNESS-GAP-ANALYSIS.md) —
  the current repository audit and durable documentation gaps.

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

HARNESS-00 does not authorize HARNESS-01, PAPER-02 runtime changes, formal
forward paper, live trading, Phase 13, or any multi-agent orchestration. Each
later package requires its own gate and review.
