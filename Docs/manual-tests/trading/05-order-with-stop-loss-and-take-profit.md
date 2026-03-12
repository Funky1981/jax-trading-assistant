# MT-05 — Entry Order with Stop-Loss and Take-Profit Protection

**Area:** Order Ticket  
**Type:** Functional  
**Priority:** P0 — Critical Path  

---

## Objective

Verify that a user can attach a stop-loss and take-profit to an entry order and that both protection legs are sent to the broker correctly.

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
| 2 | Enter Symbol = `AAPL` | |
| 3 | Set Side = `Buy`, Order Type = `Market`, Quantity = `10` | |
| 4 | Enter **Stop Loss** = `195.00` | Below a typical AAPL price |
| 5 | Enter **Take Profit** = `215.00` | Above a typical AAPL price to create 2:1 R:R |
| 6 | Observe the panel summary line — it should say **"Entry with protection"** | Confirm badge/label changes |
| 7 | Click **Submit Order** | |
| 8 | Verify the confirmation dialog shows all three prices | symbol, side, qty, stopLoss=195.00, takeProfit=215.00 |
| 9 | Click **Confirm** | |
| 10 | Check Trade Blotter for **three rows**: entry, stop-loss bracket, take-profit bracket | |

---

## Expected Results

- [ ] The panel summary badge changes to "Entry with protection" when either SL or TP is filled
- [ ] Confirmation dialog lists: Stop Loss = 195.00, Take Profit = 215.00
- [ ] After fill, Trade Blotter shows the entry + at least one bracket order (stop/limit protect)
- [ ] Positions panel shows AAPL long with SL and TP annotated (if supported by UI)
- [ ] Form resets after submission (all fields including SL/TP cleared)

---

## Failure Indicators

- Confirmation dialog does not show SL/TP values → form state not passed to confirmation payload
- Only one blotter row appears instead of bracket orders → backend not creating bracket legs
- "Entry with protection" label never appears → conditional label logic broken
- Take-profit price accepted below stop-loss price with no validation → missing guard
