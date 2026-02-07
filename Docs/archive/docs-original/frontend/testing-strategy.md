# Testing Strategy

## Goals
- Prevent regressions in trading flows.
- Validate UI performance under streaming conditions.
- Ensure accessibility compliance.

## Test Layers

1. **Unit Tests**
   - Component props, rendering, and variants.
   - Domain logic (PnL, risk calculations).
2. **Integration Tests**
   - Page‑level flows: order placement, cancels, edits.
   - Data handling: real‑time updates and caching behavior.
3. **End‑to‑End Tests**
   - Critical user journeys:
     - Place order → confirm → verify blotter update.
     - Live price update → watchlist refresh.
     - Risk limits block order.

## Tooling Guidance (Documentation Only)
- **Unit/Integration:** React Testing Library + Jest/Vitest.
- **E2E:** Playwright or Cypress.
- **Accessibility:** axe or equivalent automated checks.

## Performance Testing
- Automated checks for render cadence and memory usage.
- Synthetic streams for high‑frequency updates.
- Threshold‑based alerts on render time regressions.

