# Codex Work-Package Internal Handover Template

This records an internally completed work package during Autonomous Development Mode. It keeps every package independently reviewable but is not normally an external stop point.

## Identity
- Phase:
- Work package:
- Branch:
- Starting commit:
- Resulting commit:
- Working-tree state:

## Scope completed
- Acceptance criteria:
- Implemented:
- Deliberately not implemented:
- Later-phase boundaries preserved:

## Files and data
- Files changed:
- Migrations:
- Contract/schema/API changes:
- Configuration changes:

## Verification
- Tests added/changed:
- Exact commands/results:
- Negative/failure paths:
- Live/integration verification:

## Self-review
- Material findings:
- Corrections made:
- Remaining non-blocking limitations:
- Why progression inside the phase is safe:

## Safety and invariants
- Paper/live state:
- Broker/execution state:
- Candidate/order/trade mutation:
- Credential exposure:
- Paid dependency introduced:

## Acceptance evidence
Map each acceptance criterion to concrete evidence.

## Internal package result
Use one:
- `IMPLEMENTED / INTERNALLY VERIFIED`
- `BLOCKED`
- `ROADMAP CHANGE REQUIRED`

If internally verified, Codex may continue to the next work package inside the same authorised phase. If blocked or roadmap change is required, stop. At phase end use `PHASE-REVIEW-HANDOVER.md`.
