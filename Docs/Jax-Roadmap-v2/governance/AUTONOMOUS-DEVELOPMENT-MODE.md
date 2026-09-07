# Jax Autonomous Development Mode

## Status

**ACTIVE from Phase 04 onward unless explicitly suspended by the technical lead.**

This changes the review cadence, not the roadmap architecture, acceptance criteria, safety model, or phase gates. `Docs/ROADMAP.md` remains authoritative.

## Autonomous phase loop

For the currently authorised phase, Codex must:

1. read the phase `README.md`, `GATE.md`, governance files and first incomplete work package;
2. inspect existing implementation before designing;
3. implement only the current work package;
4. add meaningful deterministic success and failure-path verification;
5. run focused verification;
6. adversarially self-review the change;
7. correct material findings;
8. update package evidence/status;
9. create one bounded local commit;
10. verify repository hygiene;
11. continue to the next work package without waiting for external review;
12. repeat until the phase exit condition is genuinely demonstrated or a hard stop occurs.

Package boundaries remain real. Autonomous mode does not authorise giant cross-phase commits or opportunistic later-phase implementation.

## External review boundary

External Sol technical-lead review is required when:

- the current phase exit condition is demonstrated;
- a hard-stop condition occurs;
- a material roadmap change is required;
- live trading/execution authority would change.

At a normal phase boundary Codex stops and returns `PHASE-REVIEW-HANDOVER.md`. Codex must not start the next phase until the technical lead returns GO or accepted CONDITIONAL GO.

## Supersession rule

While this mode is active, legacy wording in individual phase/package files such as "STOP after the handover", "do not begin the next work package", or "every work package requires independent external review" is interpreted as package-boundary discipline rather than an external-review requirement.

This file and `CODEX-OPERATING-RULES.md` govern the active cadence. Package acceptance criteria remain fully binding.

## Hard stops

Stop for user/technical-lead input when any of these occur:

### Money / external services
- paid subscription or paid dataset required;
- material cloud/service cost introduced;
- account upgrade or purchase required.

Default development policy is zero-cost/free where technically sound.

### Credentials / access
- a new secret is required and not already securely configured;
- correct credentials remain rejected;
- access controls would need bypassing.

### Architecture
- a roadmap assumption is materially wrong;
- major new runtime/language/infrastructure is proposed;
- a substantial irreversible design decision is required;
- broad duplication/replacement of accepted architecture would be required;
- a package cannot be completed without pulling later-phase scope forward.

### Database / data loss
- destructive migration;
- material restructuring of persisted production-like data;
- irreversible transformation.

Routine bounded additive migrations explicitly required by an authorised package remain allowed.

### Git
Never autonomously push, force-push, reset, rebase, clean, discard user work, or rewrite published history. Bounded local commits are authorised.

### Trading safety
Stop before real-money broker execution, autonomous live trading, real-money order submission, unrestricted leverage, or autonomous approval authority.

## Self-review requirement

Before moving to the next package, Codex must inspect its own diff for incorrect source semantics, fabricated timestamps, look-ahead leakage, missing provenance, fail-open pagination, identity collisions, stale data treated as current, missing negative paths, swallowed errors, non-deterministic gates, unsafe cursor/checkpoint behaviour, coupling to later phases, accidental trading-state mutation, and unnecessary complexity.

## Status semantics

During autonomous phase execution:

- `IMPLEMENTED / INTERNALLY VERIFIED` means Codex completed a package and may proceed within the same phase.
- `COMPLETE / GO` remains an externally accepted state.
- phase GO is always external.

## Safety

Autonomous mode changes development velocity only. Until separately authorised:

- `ALLOW_LIVE_TRADING=false`
- `BROKER_EXECUTION_ALLOWED=false`
- execution disabled
- execution worker disabled
- maximum leverage 1x
- Recommendation execution authority = NONE
