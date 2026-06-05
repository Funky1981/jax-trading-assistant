# Replace Manual ETF Dead-End With Policy Reroute Card

## Summary

- Priority: P0
- Phase: Phase 1
- Estimate: 1-2d
- Outcome: fewer dead ends

## Objective

Prevent users from discovering ETF policy constraints only after investing effort in the manual order flow.

## In-scope touchpoints

- `frontend/src/components/dashboard/OrderTicketPanel.tsx`

## Implementation notes

- Perform route/policy checks as early as possible after symbol entry.
- If the symbol is manual-allowed, continue to the current ticket.
- If approval is required, replace or interrupt the ticket with a policy card explaining the route and linking to the approval flow.
- If blocked, explain why and provide a recovery path such as choosing another symbol or opening the approved ETF workflow.
- Preserve existing backend enforcement; this is a UX improvement, not a policy bypass.

## Acceptance criteria

- ETF manual entry never ends in a dead-end message without a next-step CTA.
- Approval-required symbols show "Open approval flow" or equivalent route-aware CTA.
- Blocked symbols explain the policy reason.
- Existing broker submission guards remain unchanged and enforce policy server-side.

## Suggested validation

- Run targeted frontend tests for manual-allowed, approval-required, and blocked symbols.
- Run affected e2e trading flow coverage.
- Do not skip backend guard tests if any server behavior is touched.

## Dependencies

- None, but the final UX should align with `03-opportunity-adapter.md` route language.
