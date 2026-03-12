# MT-11 — Reject a Strategy Signal

**Area:** Signals Queue Panel  
**Type:** Functional  
**Priority:** P1 — High  

---

## Objective

Verify that a pending signal can be rejected and is removed from the active queue without placing any order.

---

## Preconditions

- [ ] At least one pending signal exists in the Signals Queue
- [ ] App is open at `/trading` or Dashboard
- [ ] Pilot is not read-only

---

## Test Steps

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to **Trading** page → Signals Queue panel | |
| 2 | Identify a pending signal card | Note the symbol for reference |
| 3 | Enter your name in the **Approver** field on the card | |
| 4 | Click the **Reject** button (red X icon) | |
| 5 | Observe the signals queue | |
| 6 | Navigate to the Trade Blotter | |

---

## Expected Results

- [ ] After rejection, the signal card's status badge changes to `rejected` or the card is removed from the queue
- [ ] **No** new order appears in the Trade Blotter for the rejected signal's symbol
- [ ] **No** new position appears in the Positions panel
- [ ] Signals Queue count (if displayed) decrements
- [ ] The reject action does not throw an error banner

---

## Failure Indicators

- Signal stays visible and status still shows `pending` → reject API call failed
- A buy/sell order appears in blotter after rejection → reject not preventing order creation
- Reject button is absent or disabled → check if pilot read-only is blocking it
