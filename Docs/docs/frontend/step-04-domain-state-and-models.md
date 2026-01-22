# Step 04: Domain State & Models

## Objective
Create a domain‑focused layer that encapsulates trading business logic, ensuring separation of concerns and testability.

## Actions
1. **Model trading entities**
   - Orders, positions, instruments, risk limits, alerts.
   - Use explicit event types (`OrderPlaced`, `PriceUpdated`).

2. **Business logic utilities**
   - PnL calculations, margin impact, risk summaries.
   - Keep these utilities framework‑agnostic.

3. **State management approach**
   - Use unidirectional data flow (Redux Toolkit, Zustand, or Context + reducers).
   - Avoid business logic inside React components.

4. **Selectors and derived data**
   - Memoized selectors for expensive calculations.

## Deliverables
- `domain/` models and utilities.
- `domain/events` and `domain/selectors`.

## Exit Criteria
- Domain logic is unit‑tested.
- UI components receive derived data via selectors.

