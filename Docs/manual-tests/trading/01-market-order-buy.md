# MT-01 — Place a Market Buy Order

**Area:** Order Ticket  
**Type:** Functional  
**Priority:** P0 — Critical Path  

---

## Objective

Verify that a user can submit a market buy order for a valid symbol and see it appear in the Trade Blotter with status `pending` → `filled`.

---

## Preconditions

- [ ] App is open at `/trading` or `/order-ticket`
- [ ] Broker status banner shows "Connected" (green) or paper mode is active
- [ ] Trading Pilot is NOT in read-only mode (no red shield banner)
- [ ] Watchlist has at least one symbol (e.g. `AAPL`)

---

## Test Steps

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to **Trading** → locate the **Order Ticket** panel | Panel should default to expanded |
| 2 | In the **Symbol** field, type `AAPL` | Symbol field accepts uppercase and lowercase |
| 3 | Set **Side** to `Buy` | Default is Buy — verify the dropdown shows "Buy" |
| 4 | Set **Order Type** to `Market` | Price field should be hidden or disabled |
| 5 | Enter **Quantity** = `10` | Must be a positive integer |
| 6 | Leave Stop Loss and Take Profit fields **blank** | Protection is optional |
| 7 | Click **Submit Order** | A confirmation dialog should appear |
| 8 | Review the confirmation summary: symbol=AAPL, side=Buy, type=Market, qty=10 | Confirm details are correct |
| 9 | Click **Confirm** in the dialog | |
| 10 | Observe the Trade Blotter panel | |

---

## Expected Results

- [ ] Confirmation dialog shows "AAPL — Buy — Market — 10 shares" (or equivalent wording) before submission
- [ ] After confirm, the form resets (all fields cleared)
- [ ] A new row appears in the **Trade Blotter** with symbol `AAPL`, side `Buy`, type `Market`, qty `10`
- [ ] The order status badge shows `pending` initially
- [ ] Within a few seconds (paper fill), the status updates to `filled`
- [ ] No error banner appears on the page
- [ ] The **Positions** panel gains or increases an `AAPL` long entry

---

## Failure Indicators

- Form submits but no order row appears in the blotter → backend may be down
- Order stays in `pending` indefinitely → IB paper fill not working
- Red "Trading is read-only" banner appears → pilot guard is blocking (see MT-15)
- Dialog does not appear → confirmation flow regression
