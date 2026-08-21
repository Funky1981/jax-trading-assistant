# GO / NO-GO Process

The architecture reviewer checks the implementation handover against the package acceptance criteria and can inspect the repository if needed.

## Decisions

**GO** — acceptance criteria are proven; no material unresolved defect blocks the next package.

**CONDITIONAL GO** — only non-blocking issues remain. Conditions must be written down with a specific follow-up owner/package.

**NO-GO** — acceptance evidence is incomplete, an invariant is broken, tests do not prove the requirement, or implementation risk is materially higher than accepted.

**ROADMAP CHANGE** — new evidence shows the planned architecture, dependency or sequencing is wrong. Pause implementation and update the affected roadmap packages.

## Reviewer checklist

- Scope discipline
- Contract correctness
- Determinism/replay where required
- Data provenance and freshness where required
- Negative/failure-path tests
- Safety boundaries
- Migration reversibility/compatibility where applicable
- Observability/operator evidence
- No hidden coupling to later phases
- Working tree/repository hygiene
