# MT-08 — Close an Open Position with a Limit Exit

**Area:** Portfolio / Positions Panel  
**Type:** Functional  
**Priority:** P1 — High  

---

## Objective

Verify that a user can close a position using a limit order exit price, and that the limit price is stored and respected by the broker.

---

## Preconditions

- [ ] An open long position exists (e.g. MSFT 5 shares)
- [ ] App is open at `/portfolio` or `/trading`
- [ ] Pilot is not read-only

---

## Test Steps

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to the **Positions** panel | |
| 2 | Locate the `MSFT` position and click **Close** | |
| 3 | In the Close Position dialog, set **Quantity** = `5` | Full close |
| 4 | Set **Order Type** to `LMT` (Limit) | Limit Price field should appear |
| 5 | Enter **Limit Price** = `1.00` above current ask, e.g. `500.00` | Set an ambitious price that won't fill immediately |
| 6 | Check the **Confirm close** checkbox | |
| 7 | Click **Submit** | |
| 8 | Check Trade Blotter | |

---

## Expected Results

- [ ] When LMT is selected, a "Limit Price" input becomes visible in the dialog
- [ ] Submission blocked if Limit Price field is empty when LMT is selected
- [ ] Blotter row: MSFT, Sell, Limit, price=500.00, workflow=Close, status=`pending`
- [ ] Position remains in place (not closed yet — limit not filled)
- [ ] Notice message shown: "position stays open until IB fills the exit order"

---

## Failure Indicators

- Limit Price field hidden even after selecting LMT → conditional UI broken
- Order submitted without limit price → API may reject or accept with no price
- Blotter shows Market type instead of Limit → order type not passed to API
