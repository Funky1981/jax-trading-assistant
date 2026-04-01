# MT-10 — Approve a Strategy Signal and Verify Fill

**Area:** Signals Queue Panel  
**Type:** Functional  
**Priority:** P0 — Critical Path  

---

## Objective

Verify that a pending strategy signal can be approved by the user and that the resulting trade order is created and fills in paper mode.

---

## Preconditions

- [ ] At least one signal appears in the Signals Queue with status `pending`
- [ ] App is open at `/trading` or `/` (Dashboard) with Signals Queue panel visible
- [ ] Pilot is not read-only
- [ ] Broker is connected

---

## Test Steps

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to **Trading** page | Signals Queue panel is visible in the grid |
| 2 | Locate a pending signal card — note the symbol, type (BUY/SELL), entry price, stop, and target | |
| 3 | Review the confidence percentage displayed on the card | Should show e.g. "72% confidence" |
| 4 | Review the Risk:Reward ratio displayed | |
| 5 | Optionally click **Analyze** to request an AI recommendation | See MT-12 for full AI analysis test |
| 6 | Enter your name or initials in the **Approver** field | |
| 7 | Click the **Approve** button (green check) | |
| 8 | Observe the signal card status | |
| 9 | Navigate to the Trade Blotter | |

---

## Expected Results

- [ ] Signal card shows: symbol, BUY/SELL badge, strategy name, confidence %, entry/stop/target prices, R:R ratio
- [ ] Approve button is accessible and not disabled (pilot is ready)
- [ ] After approval, the signal status badge changes from `pending` to `approved` (or the card disappears from the queue)
- [ ] Trade Blotter gains a new order row corresponding to the signal's entry price and symbol
- [ ] Order fills in paper mode → status `filled`
- [ ] Positions panel gains a new position corresponding to the signal

---

## Failure Indicators

- Approve button is disabled with no explanation → check pilot status banner
- Signal card disappears but no blotter order created → signal approved but order not placed
- Signals Queue does not refresh after approval → query invalidation not firing
- Approver field is missing or not required → input not wired in the component
