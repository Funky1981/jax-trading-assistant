# Step 04: Domain State & Models

## Objective

Create a domain-focused layer that encapsulates trading business logic, ensuring separation of concerns and testability.

## Actions

1. **Model trading entities**
   - Orders, positions, instruments, risk limits, alerts.
   - Use explicit event types (`OrderPlaced`, `PriceUpdated`).

2. **Business logic utilities**
   - PnL calculations, exposure, risk summaries.
   - Keep these utilities framework-agnostic.

3. **State management approach**
   - Use unidirectional data flow via event reducers.
   - Avoid business logic inside React components.

4. **Selectors and derived data**
   - Memoized selectors for expensive calculations (later phase).

## Deliverables
- `frontend/src/domain/models.ts`, `frontend/src/domain/events.ts`, `frontend/src/domain/state.ts`.
- `frontend/src/domain/calculations.ts`, `frontend/src/domain/selectors.ts`.
- Tests under `frontend/src/domain/__tests__/`.

## Exit Criteria
- Domain logic is unit-tested.
- UI components receive derived data via selectors.
