# MT-02 — Place a Market Sell (Short) Order

**Area:** Order Ticket  
**Type:** Functional  
**Priority:** P0 — Critical Path  

---

## Objective

Verify that a user can submit a market sell order (short entry or exit) and that it is correctly represented in the blotter and positions panel.

---

## Preconditions

- [ ] App is open at `/trading` or `/order-ticket`
- [ ] Broker is connected or paper mode is active
- [ ] Trading Pilot is NOT in read-only mode
- [ ] To test a closing sell: an existing long position in `MSFT` exists in the Positions panel

---

## Test Steps

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to **Order Ticket** panel | |
| 2 | Type `MSFT` in the **Symbol** field | |
| 3 | Set **Side** to `Sell` | Dropdown must switch to Sell |
| 4 | Set **Order Type** to `Market` | |
| 5 | Enter **Quantity** = `5` | |
| 6 | Leave Stop Loss and Take Profit blank | |
| 7 | Click **Submit Order** | Confirmation dialog appears |
| 8 | Verify the summary shows Side = **Sell** | Confirm the order is a sell, not buy |
| 9 | Click **Confirm** | |
| 10 | Check Trade Blotter | |

---

## Expected Results

- [ ] Confirmation dialog: symbol=MSFT, side=Sell, type=Market, qty=5
- [ ] After confirm, form resets
- [ ] New blotter row: `MSFT`, `Sell`, `Market`, qty=`5`, status=`pending`
- [ ] Status transitions to `filled`
- [ ] If a long MSFT position existed, it is reduced by 5 shares in the Positions panel
- [ ] If no prior position, a short MSFT position of -5 appears in the Positions panel

---

## Failure Indicators

- Side shows "Buy" in blotter despite selecting Sell → UI state not bound correctly
- Position panel not updated → position refresh not triggered after fill
