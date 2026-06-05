# Add Sentiment-Aware Approval Evidence Pack

## Summary

- Priority: P1
- Phase: Phase 2
- Estimate: 2-3d
- Outcome: policy-safe evidence

## Objective

Include sentiment context in approval-required opportunities without weakening human approval or execution policy.

## In-scope touchpoints

- Approval read models
- `ApprovalsPage` redesign or detail panel

## Implementation notes

- Add sentiment summary, source count, top drivers, price agreement/divergence, limitations, and snapshot timestamp to approval evidence.
- Keep route reason, policy reason, risk controls, expiry, and decision history visible.
- Approver actions remain approve, reject, defer, or request deeper analysis.
- Approval decisions should preserve the sentiment snapshot that was reviewed.
- Do not allow sentiment evidence to auto-approve or bypass ETF/manual trading policy.

## Acceptance criteria

- Approval-required Opportunities show sentiment evidence when available.
- Approval UI distinguishes sentiment evidence from policy decision state.
- Human action is still required for approval-gated flows.
- Decision history persists notes and reviewed evidence context.

## Suggested validation

- Run backend tests for approval read-model sentiment fields.
- Run frontend approval page tests for evidence rendering and action flow.
- Run behavior-sensitive approval/e2e tests if approval submission paths are changed.

## Dependencies

- Depends on `11-opportunity-sentiment-api-fields.md`.
- Uses drawer conventions from `12-opportunity-drawer-sentiment-explanation.md` where possible.
