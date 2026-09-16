# PAPER-01 — Exploratory Single-Recovery Paper Package

Status: **IMPLEMENTED / EXTERNAL REVIEW REQUIRED**.

## Purpose

PAPER-01 makes the short-horizon trader model observable and reviewable through one bounded exploratory paper scenario. It is a contract and evidence-collection package, not a claim of profitability or edge.

## Scope

- one candidate thesis at a time;
- explicit entry assumptions, invalidation, recovery condition, and exit rationale;
- human approval before a hypothetical paper ticket is recorded;
- five-session monitoring with thesis state, checkpoints, and provenance;
- post-decision outcome review with unresolved and missing data preserved;
- no broker order, live order, execution instruction, or fill.

## Evidence boundary

Automated tests establish implementation behaviour. They do not establish predictive validity, profitability, a demonstrated trading edge, or readiness for formal forward-paper evidence. Those require external review and a separately authorized package.

## Required review questions

1. Is the single-recovery thesis sufficiently falsifiable and bounded?
2. Are approval, monitoring, exit, and outcome records auditable without implying execution?
3. Are exploratory observations kept separate from formal evidence and promotion decisions?
4. Are unknown, missing, ambiguous, and contradictory inputs retained rather than silently normalised?

See `Docs/PAPER_TRADING/EXPLORATORY_VS_FORMAL.md`, `Docs/PAPER_TRADING/TRADE_LIFECYCLE.md`, and `Docs/PAPER_TRADING/APPROVAL_AND_EXIT_POLICY.md` for the detailed contracts.
