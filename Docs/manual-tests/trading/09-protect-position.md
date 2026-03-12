# MT-09 — Add Stop-Loss and Take-Profit to an Open Position

**Area:** Portfolio / Positions Panel  
**Type:** Functional  
**Priority:** P0 — Critical Path  

---

## Objective

Verify that a user can attach bracket protection (stop-loss and/or take-profit) to an already-open position using the Protect dialog.

---

## Preconditions

- [ ] An unprotected long position exists, e.g. AAPL 10 shares (no SL/TP attached)
- [ ] App is open at `/portfolio` or `/trading` with Positions panel visible
- [ ] Pilot is not read-only

---

## Test Steps

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to **Positions** panel | |
| 2 | Locate the unprotected AAPL row | |
| 3 | Click the **Protect** button (shield icon) on the AAPL row | |
| 4 | The Protect Position dialog appears | |
| 5 | Verify **Quantity** is pre-filled with the current position size | |
| 6 | Verify **Stop Loss** is pre-filled with a default of ~2% below market price | Default calculation: `marketPrice * 0.98` |
| 7 | Verify **Take Profit** is pre-filled with a default of ~4% above market price | Default: `marketPrice * 1.04` |
| 8 | Adjust Stop Loss if desired | e.g. set to `195.00` |
| 9 | Adjust Take Profit if desired | e.g. set to `215.00` |
| 10 | Check the **Confirm protect** checkbox | |
| 11 | Click **Submit** | |
| 12 | Observe the Positions panel and Trade Blotter | |

---

## Expected Results

- [ ] Protect dialog pre-fills quantity, stop-loss (2% below market), and take-profit (4% above market)
- [ ] Checkbox must be checked before Submit can be clicked
- [ ] After submission, Trade Blotter shows new protect order rows (stop and/or limit bracket legs)
- [ ] Protect order workflow column shows "Protect"
- [ ] Position row in Positions panel reflects protection attached (if annotated in UI)
- [ ] No error banner or toast

---

## Failure Indicators

- Default SL/TP values not pre-populated → `getDefaultStop`/`getDefaultTarget` helpers not called
- Protect dialog appears for a position with no market price and shows blank defaults → zero price guard missing
- Both SL and TP left blank and form still submits → empty protect should be rejected or warned
- Protect order created but shows workflow = "Entry" → workflow tag not set to "protect" on backend
