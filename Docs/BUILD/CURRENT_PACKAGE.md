# Current Build Package

## PAPER-01 — event-driven exploratory paper-trader package

Status: **PAPER-01 NO-GO / PAPER-01B IMPLEMENTED / EXTERNAL RE-REVIEW REQUIRED**.

PAPER-01C is the current implementation package. It closes the runtime-loop seam for the bounded event-driven exploratory paper-trader model: canonical source-backed event/evidence handoff, human-approved paper entry, isolated paper lifecycle, restart-safe scheduled review, deterministic exit approval, and outcome review. It does not run a pilot, demonstrate a trading edge, authorize PAPER-02, create formal evidence, or permit live execution.

Read in this order:

1. `Docs/JAX_PRODUCT_CHARTER.md`
2. `Docs/ROADMAP.md`
3. `Docs/STATUS.md`
4. `Docs/BUILD/PAPER-01.md`
5. `Docs/BUILD/PAPER-01-CAPABILITY-MAP.md`
6. Relevant paper-trading contracts under `Docs/PAPER_TRADING/`

The next decision is external re-review of PAPER-01B. No pilot, formal
forward-paper run, Phase 13 work, broker execution, or live trading is
authorized by this package.
