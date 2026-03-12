# MT-06 — Cancel a Pending Order from the Blotter

**Area:** Trade Blotter  
**Type:** Functional  
**Priority:** P1 — High  

---

## Objective

Verify that a user can cancel a pending order directly from the Trade Blotter and that the order transitions to `cancelled` status.

---

## Preconditions

- [ ] A pending limit or stop order exists in the blotter (use MT-03 or MT-04 to create one)
- [ ] App is open at `/trading` or `/blotter`
- [ ] Pilot is not read-only

---

## Test Steps

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to **Blotter** page or expand Trade Blotter panel on Trading page | |
| 2 | Identify a row with status `pending` | |
| 3 | Click the **Cancel** button (🗑 / X / Cancel) on that row | A confirmation dialog should appear |
| 4 | Verify the dialog shows the order details (symbol, qty, type) | Prevents accidental cancellation |
| 5 | Click **Confirm Cancel** in the dialog | |
| 6 | Observe the blotter row | |

---

## Expected Results

- [ ] A confirmation dialog appears before cancellation
- [ ] After confirming, the blotter row status changes from `pending` to `cancelled`
- [ ] The cancel button is no longer clickable on the now-cancelled row (disabled or hidden)
- [ ] No error toast or banner appears
- [ ] If a position was contingent on this order, the Positions panel is unaffected (position not already open)

---

## Failure Indicators

- Cancel button visible on `filled` or `cancelled` rows → action guard missing
- Order transitions to `cancelled` without confirmation dialog → safety dialog removed
- Row stays `pending` after cancellation → backend cancel call failed; check network tab
- Error banner: "Cannot cancel filled order" → wrong order selected for cancel
