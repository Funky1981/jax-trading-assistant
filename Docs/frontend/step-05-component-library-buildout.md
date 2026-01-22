# Step 05: Component Library Build-out

## Objective
Build a reusable component library to power all pages and widgets consistently.

## Actions
1. **Build primitives first**
   - Buttons, inputs, selectors, tabs, tooltips.
   - Ensure keyboard navigation and ARIA roles.

2. **Add data-dense components**
   - Tables, DataGrid, charts, order book, depth ladder.
   - Implement virtualization for large datasets.

3. **Trading-specific modules**
   - OrderTicket, PositionCard, RiskSummary, PnLIndicator.

4. **Documentation**
   - Document props, variants, performance notes, and examples.

## Deliverables
- `frontend/src/components/primitives/PrimaryButton.tsx`
- `frontend/src/components/primitives/TextInput.tsx`
- `frontend/src/components/primitives/SelectInput.tsx`
- `frontend/src/components/data/DataTable.tsx`
- `frontend/src/components/trading/PnLIndicator.tsx`
- `frontend/src/components/trading/PositionCard.tsx`
- `frontend/src/components/trading/RiskSummary.tsx`
- `frontend/src/components/trading/OrderTicket.tsx`
- Tests under `frontend/src/components/**/__tests__/`.

## Exit Criteria
- Components are theme-compatible and responsive.
- Components have baseline tests and run under CI.
