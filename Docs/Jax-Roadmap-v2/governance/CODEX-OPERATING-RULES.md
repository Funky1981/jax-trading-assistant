# Codex Operating Rules

Autonomous phase execution is governed by `AUTONOMOUS-DEVELOPMENT-MODE.md`.

## Repository safety
1. Verify repository, branch, HEAD, upstream/ahead-behind, working tree and untracked files before beginning a phase and whenever state becomes ambiguous.
2. Do not reset, rebase, clean, stash, discard, overwrite, amend, switch branch or push unless explicitly instructed.
3. Local bounded commits are authorised for completed work packages.
4. Never hide or discard unexpected user work.

## Scope
5. Work only inside the currently authorised phase.
6. Implement one roadmap work package at a time.
7. Do not opportunistically implement later packages or phases.
8. Existing/legacy code does not override the active roadmap.
9. If a roadmap assumption is materially wrong, stop with `ROADMAP CHANGE`.

## Verification
10. Every package must satisfy acceptance criteria with meaningful success and failure-path evidence.
11. Run focused tests and appropriate wider verification before committing.
12. Perform an adversarial self-review before progressing.
13. Document limitations/deviations rather than hiding them.
14. A passing build alone is not acceptance proof.

## Autonomous cadence
15. Codex may progress through work packages inside the authorised phase without external review after each package.
16. Preserve a bounded commit/evidence record for each package.
17. Stop at the phase exit gate and return `PHASE-REVIEW-HANDOVER.md`.
18. Do not begin the next phase until external GO or accepted CONDITIONAL GO.
19. Hard-stop conditions override autonomous progression.

## Models
20. Follow `MODEL-ROUTING-POLICY.md`.
21. Luna is the default implementation model.
22. Sol is reserved for consequential architecture/phase decisions and technical-lead review.
23. Never claim a model-specific subreview unless runtime evidence confirms it.

## Safety
24. Preserve current paper/live boundaries unless a future externally approved gate changes them.
25. Never silently create or enable live execution, broker orders, fills, leverage, approvals or execution intents.
26. Old downstream trading pathways do not authorise their use in earlier phases.
27. Paid services/data require explicit user approval.
28. Credentials must never be printed, committed or written into evidence artifacts.
