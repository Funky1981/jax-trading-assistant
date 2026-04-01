# MT-03 — Place a Limit Buy Order

**Area:** Order Ticket  
**Type:** Functional  
**Priority:** P1 — High  

---

## Objective

Verify that selecting "Limit" order type reveals the limit price field and that the order is submitted with the correct limit price.

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
| 2 | Type `SPY` in the **Symbol** field | |
| 3 | Set **Side** to `Buy` | |
| 4 | Set **Order Type** to `Limit` | The **Limit Price** field should now appear / become enabled |
| 5 | Enter a **Limit Price** slightly below current market price, e.g. `490.00` | Use a price that won't immediately fill in paper mode |
| 6 | Enter **Quantity** = `20` | |
| 7 | Click **Submit Order** | |
| 8 | Verify the confirmation dialog shows Type = **Limit** and the limit price | |
| 9 | Click **Confirm** | |
| 10 | Observe the Trade Blotter | |

---

## Expected Results

- [ ] When Order Type is changed to "Limit", a **Limit Price** input field appears
- [ ] Confirmation dialog shows: symbol=SPY, side=Buy, type=Limit, price=490.00, qty=20
- [ ] Blotter row: SPY, Buy, Limit, status=`pending`
- [ ] The order remains `pending` (limit not yet filled) since price is below market
- [ ] No form submission occurs if the Limit Price field is empty (validation)
- [ ] Form resets after successful submission

---

## Failure Indicators

- Limit price field does not appear when Limit is selected → UI conditional rendering broken
- Order submitted without price field → server may accept or reject — check blotter for correct price stored
- Order immediately fills despite price being below market → paper engine not respecting limit
