# Add AiTradingPage Shell With Unified Opportunity Feed V1

## Summary

- Priority: P0
- Phase: Phase 1
- Estimate: 3-4d
- Outcome: dedicated AI home

## Objective

Create a dedicated AI Trading route that shows scanner state, a unified Opportunity feed, and clear next actions.

## In-scope touchpoints

- `frontend/src/pages/AiTradingPage.tsx`
- `frontend/src/data/types.ts`
- `frontend/src/data/signals-service.ts`
- `frontend/src/data/approvals-service.ts`

## Implementation notes

- Add an `AiTradingPage` route and navigation item.
- Show scanner status at the top even if Phase 2 API data is not ready yet.
- Render a unified feed using the Opportunity adapter from `03-opportunity-adapter.md`.
- Each feed item should show symbol, summary, confidence band, route, route reason, expiry, and status.
- Next actions should be route-aware: review order, send to approval, watch, dismiss, or open blocked-state guidance.
- Avoid treating "AI" as an optional per-signal add-on in the primary workflow.

## Acceptance criteria

- AI Trading exists as a dedicated route.
- Feed items use user-facing Opportunity language.
- Opportunities from existing signals, candidates, or approvals can be represented in one list.
- Every Opportunity item has a visible next action.
- Loading, empty, error, and stale-data states are explicit.

## Suggested validation

- Run targeted frontend unit tests for feed rendering and state mapping.
- Run route/navigation e2e checks that open AI Trading from the shell.
- Use mocked or fixture data for signals, candidates, approvals, and empty states.

## Dependencies

- Pairs with `03-opportunity-adapter.md`.
- Benefits from `04-scanner-sentiment-controls.md` for scanner controls.
