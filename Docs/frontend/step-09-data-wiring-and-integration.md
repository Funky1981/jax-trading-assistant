# Step 09: Data Wiring & Domain Integration

## Objective
Replace mock UI data with domain-backed state and streaming updates so pages reflect live trading workflows.

## Actions
1. **Domain provider**
   - Introduce a `DomainProvider` wrapping the app with a reducer-driven store.
   - Expose typed actions (order placement, price updates, position updates).

2. **Streaming bridge**
   - Simulate market ticks using `createStreamBuffer` to batch updates.
   - Dispatch `PriceUpdated` events at a stable cadence.

3. **Page integration**
   - Replace mock data in dashboard, blotter, portfolio, and order ticket.
   - Use selectors for derived data (exposure, PnL, watchlist).

4. **Order flow**
   - Wire order ticket submission to domain actions.
   - Update positions and blotter rows on fills.

## Deliverables
- `frontend/src/domain/store.tsx` with provider, actions, and stream wiring.
- Pages wired to domain selectors instead of mock arrays.
- Dashboard widgets powered by ticks, orders, and positions.

## Exit Criteria
- No mock data remains in core pages.
- Dashboard and blotter update when new orders are placed.
- Streamed prices update risk and PnL summaries without jitter.
