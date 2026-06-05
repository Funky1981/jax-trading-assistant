# Build Reusable Opportunity Drawer With Sentiment Explanation

## Summary

- Priority: P0
- Phase: Phase 2
- Estimate: 3-4d
- Outcome: unified evidence UX

## Objective

Create one reusable Opportunity detail drawer that explains strategy, price, news, sentiment, policy, risk, and next actions in one place.

## In-scope touchpoints

- New shared drawer/component set
- AI Trading feed and other Opportunity entry points

## Implementation notes

- Show a sentiment summary sentence rather than raw scores alone.
- Include current sentiment score and direction, weighted source breakdown, time window, top drivers, source list, price/sentiment agreement or divergence, sentiment contribution to confidence, and limitations.
- Include route-aware actions: review order, send to approval, watch, dismiss, request deeper analysis, or open blocked guidance.
- Keep policy and risk evidence visible beside sentiment so sentiment does not appear to overrule guardrails.
- Support loading, stale, missing sentiment, sparse sentiment, and scoring error states.

## Acceptance criteria

- The drawer can open from AI Trading, notifications, and approval contexts.
- Users can identify why the Opportunity exists and what to do next.
- Sentiment is explained as evidence, not a magic score.
- Route and policy status are visible before any action.

## Suggested validation

- Run component tests for each sentiment state and route state.
- Run e2e flow from AI Trading feed to drawer action.
- Manually verify mobile drawer layout, text wrapping, and keyboard accessibility.

## Dependencies

- Depends on `11-opportunity-sentiment-api-fields.md`.
- Can be linked from `06-notification-centre-inbox.md`.
