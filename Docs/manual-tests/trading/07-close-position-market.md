# MT-07 — Close an Open Position at Market

**Area:** Portfolio / Positions Panel  
**Type:** Functional  
**Priority:** P0 — Critical Path  

---

## Objective

Verify that a user can close an open position at the market price using the Positions panel Close dialog.

---

## Preconditions

- [ ] An open long or short position exists (e.g. AAPL 10 shares long) — use MT-01 to create one
- [ ] App is open at `/portfolio` or `/trading` with Positions panel visible
- [ ] Pilot is not read-only
- [ ] Broker is connected

---

## Test Steps

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to **Portfolio** page or expand the Positions panel | |
| 2 | Locate the `AAPL` position row | |
| 3 | Click the **Close** button (✕ / XCircle icon) on the AAPL row | |
| 4 | The Close Position dialog appears | Verify it pre-fills the symbol and quantity |
| 5 | Set **Quantity** to the full position size (default may be pre-filled) | e.g. `10` |
| 6 | Set **Order Type** to `Market` (MKT) | |
| 7 | Leave **Limit Price** blank or hidden | Not applicable for MKT |
| 8 | Check the **Confirm close** checkbox | Required before submit |
| 9 | Click **Submit** | |
| 10 | Observe Positions panel and Blotter | |

---

## Expected Results

- [ ] Close dialog pre-fills symbol with "AAPL" and quantity with current position size
- [ ] Submit is blocked until the confirm checkbox is checked
- [ ] After submit, a notice appears: "The position stays open until IB fills the exit order. Check Trade Blotter for broker status."
- [ ] Trade Blotter gains a new row: AAPL, Sell (for long close), Market, workflow=Close, status=pending → filled
- [ ] After fill, the AAPL position is removed from the Positions panel (or quantity reduced)

---

## Failure Indicators

- Close dialog opens but symbol/qty not pre-filled → component not reading position context
- Submit button enabled without checkbox checked → confirmation guard missing
- Position panel still shows position after fill → auto-refresh not triggered
- Notice text not shown after close → `onSuccess` handler not displaying message
