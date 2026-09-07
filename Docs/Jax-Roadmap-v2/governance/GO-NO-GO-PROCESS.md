# GO / NO-GO Process

## Review boundary

The normal external architecture-review boundary is now the **phase gate**, not every individual work package. Work packages inside an authorised phase are implemented and internally self-reviewed by Codex under Autonomous Development Mode.

The technical lead may still request package-level review at any time.

## Decisions

**GO PHASE NN** — package work is sufficiently proven, the phase exit condition is demonstrated, and no material unresolved defect blocks the next phase.

**CONDITIONAL GO PHASE NN** — only non-blocking issues remain; conditions are explicit and tracked.

**NO-GO PHASE NN** — the exit condition is not demonstrated, an invariant is broken, verification is insufficient, or implementation risk is materially above threshold.

**ROADMAP CHANGE** — evidence shows the planned architecture, dependency, acceptance condition or sequencing is wrong.

## Reviewer checklist
- Exit condition demonstrated, not asserted
- Scope/phase discipline
- Contract correctness
- Determinism/replay where required
- Provenance/freshness where required
- Negative/failure-path verification
- Point-in-time/backtest leakage controls
- Safety/trading boundaries
- Migration compatibility/reversibility where applicable
- Observability/operator evidence
- External dependency/cost decisions
- No hidden future-phase coupling
- Repository hygiene
- Accepted debt explicitly tracked
- Dedicated adversarial phase review completed for Phase 05 and later
- Actual phase diff reviewed; tests tested; important expected values checked independently

## Authority
Codex cannot self-award external phase GO. Only external technical-lead review advances Jax into the next phase.
