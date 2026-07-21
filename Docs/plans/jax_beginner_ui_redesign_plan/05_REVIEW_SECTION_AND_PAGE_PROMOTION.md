# Review Section and Page Promotion Process

## Purpose

Avoid redesigning unfinished or unused pages before they have a real operator task.

All non-primary pages remain under the collapsed **Review** section.

## Review section groups

Recommended grouping:

### Trading experiments

- AI Trading
- Manual Trading
- Swing Trading
- order-ticket routes
- trading-mode routes

### Research and analysis

- Research
- Macro Events
- Analysis
- timelines
- strategy cards
- ETF universe

### Testing and diagnostics

- Testing
- Paper Trading Test Plan
- Mobile Approval Harness
- E2E Tests
- System diagnostic subpages

### Records and administration

- Notifications
- Settings
- Portfolio
- Blotter
- Assistant

### Legacy workflows

- legacy routes
- equity-alpha routes not currently active
- ETF routes not currently active
- specialist guides

## Review page warning

Pages that have not been redesigned should show a small phase banner:

> Review page — this area has not yet been redesigned for the current Jax workflow.

Do not imply they are production-ready.

## Promotion checklist

A page may move from Review into primary navigation only when all are true:

1. The capability is currently being used.
2. The underlying data is genuine or explicitly labelled test data.
3. The operator has a clear repeatable task.
4. The page has plain-language purpose text.
5. The page has help and empty states.
6. The page is responsive.
7. The page has accessible keyboard behaviour.
8. Any financial values have explicit provenance.
9. Safety boundaries are visible.
10. Relevant backend read models exist.
11. Unit and E2E tests pass.
12. The user has reviewed the page with real local data.

## Redesign workflow for a promoted page

1. Review all screenshots in `images/`.
2. Identify the operator's actual task.
3. Remove irrelevant actions and information.
4. Define the minimum read model.
5. Create beginner copy.
6. Add inline help.
7. Design empty, loading, partial and error states.
8. Build responsive layout.
9. Test with genuine persisted data.
10. Promote only after user acceptance.

## Explicit non-goal

Do not redesign a Review page simply because it exists.

Existing code volume is not a reason to promote a feature.

## Suggested future promotion order

Only when development reaches these capabilities:

1. Research
2. Analysis
3. Strategy Validation
4. Shadow Mode
5. Notifications
6. Portfolio and Blotter
7. Manual or broker-connected trading

This order is provisional. Product usage should decide the final sequence.
