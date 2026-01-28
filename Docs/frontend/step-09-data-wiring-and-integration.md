# Step 09: Data Wiring & Domain Integration

## Status: ✅ COMPLETED

## Objective
Replace mock UI data with domain-backed state and streaming updates so pages reflect live trading workflows.

## Actions
1. **Domain provider** ✅
   - Introduce a `DomainProvider` wrapping the app with a reducer-driven store.
   - Expose typed actions (order placement, price updates, position updates).

2. **Streaming bridge** ✅
   - Simulate market ticks using `createStreamBuffer` to batch updates.
   - Dispatch `PriceUpdated` events at a stable cadence (250ms ticks, 500ms flush).

3. **Page integration** ✅
   - Replace mock data in dashboard, blotter, portfolio, and order ticket.
   - Use selectors for derived data (exposure, PnL, watchlist).

4. **Order flow** ✅
   - Wire order ticket submission to domain actions.
   - Update positions and blotter rows on fills (800ms simulated fill delay).

## Deliverables
- ✅ `frontend/src/domain/store.tsx` with provider, actions, and stream wiring.
- ✅ `frontend/src/data/stream-buffer.ts` for batched tick processing.
- ✅ `frontend/src/data/types.ts` with MarketTick interface.
- ✅ Pages wired to domain selectors instead of mock arrays.
- ✅ Dashboard widgets powered by live ticks, orders, and positions.
- ✅ All type declarations complete (`vitest-axe.d.ts`).

## Implementation Details

### DomainProvider Features
- Seed data for initial positions (AAPL, MSFT) and orders
- Watchlist with 4 symbols (AAPL, MSFT, SPY, TSLA)
- Real-time price streaming with random drift simulation
- Order placement with automatic fill after 800ms
- Position averaging on fills (weighted average cost)

### Selectors Added
- `selectOrders`, `selectOpenOrders`
- `selectPositions`
- `selectTicks`, `selectTickBySymbol`
- `selectTotalExposure`, `selectTotalUnrealizedPnl`
- `selectRiskBreach`

### Pages Updated
- **DashboardPage**: Uses domain state for positions, orders, ticks
- **BlotterPage**: Displays orders from domain store
- **PortfolioPage**: Live exposure and PnL calculations
- **OrderTicketPage**: Wired to `actions.placeOrder`

## Exit Criteria
- ✅ No mock data remains in core pages.
- ✅ Dashboard and blotter update when new orders are placed.
- ✅ Streamed prices update risk and PnL summaries without jitter.
- ✅ TypeScript compiles without errors.
- ✅ All 23 tests pass.
