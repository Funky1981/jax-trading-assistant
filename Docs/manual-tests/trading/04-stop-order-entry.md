# MT-04 — Place a Stop-Entry Order

**Area:** Order Ticket  
**Type:** Functional  
**Priority:** P1 — High  

---

## Objective

Verify that the "Stop" order type reveals the stop trigger price field and submits a proper stop-entry order.

---

## Preconditions

- [ ] App is open at `/trading` or `/order-ticket`
- [ ] Broker connected / paper mode active
- [ ] Pilot is not read-only

---

## Test Steps

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to **Order Ticket** panel | |
| 2 | Type `QQQ` in the **Symbol** field | |
| 3 | Set **Side** to `Buy` | |
| 4 | Set **Order Type** to `Stop` | A **Stop Price** (trigger) field should appear |
| 5 | Enter **Stop Price** above the current market price, e.g. `520.00` | Breakout entry — triggers when price crosses up |
| 6 | Enter **Quantity** = `15` | |
| 7 | Click **Submit Order** | Confirmation dialog |
| 8 | Verify the dialog shows Type = **Stop** and stop price = `520.00` | |
| 9 | Click **Confirm** | |
| 10 | Observe the Trade Blotter | |

---

## Expected Results

- [ ] "Stop Price" input field appears when Order Type = Stop
- [ ] Confirmation dialog: symbol=QQQ, side=Buy, type=Stop, stopPrice=520.00, qty=15
- [ ] Blotter row shows QQQ Stop order with status `pending`
- [ ] Order does not fill immediately (price not at trigger level)
- [ ] Form resets after confirmation
- [ ] Limit Price field is hidden (not shown) when Stop type is selected

---

## Failure Indicators

- Stop price field hidden or not editable → order type switching logic broken
- Both limit price and stop price fields visible at the same time → UI state bug
- Order fills immediately without reaching stop → paper engine not respecting stop trigger
