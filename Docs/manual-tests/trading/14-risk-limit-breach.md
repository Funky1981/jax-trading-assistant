# MT-14 — Attempt Order That Breaches Risk Constraints

**Area:** Order Ticket / Risk Guards  
**Type:** Negative / Boundary  
**Priority:** P0 — Critical Path (Safety)  

---

## Objective

Verify that orders exceeding configured risk limits are blocked server-side and that the rejection is surfaced clearly in the UI.

---

## Risk Limits Reference (from `config/risk-constraints.json`)

| Constraint | Value |
|---|---|
| Max position size | $50,000 |
| Max positions | 10 |
| Max risk per trade | 2% of account |
| Max leverage | 2.0× |
| Max daily drawdown | 20% |
| Account size (default) | $10,000 |

---

## Preconditions

- [ ] App is open at `/trading` or `/order-ticket`
- [ ] Pilot is not read-only
- [ ] Broker connected / paper mode

---

## Test Steps — Oversized Position

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to **Order Ticket** | |
| 2 | Symbol = `SPY`, Side = `Buy`, Type = `Market` | |
| 3 | Quantity = `10000` | At ~$500/share this is ~$5,000,000 — far exceeds $50k limit |
| 4 | Click **Submit Order** | |
| 5 | Click **Confirm** in the dialog | |
| 6 | Observe the result | |

**Expected:** Server rejects with an error. UI displays a red error message near the form or in the blotter row (status = `rejected`).

---

## Test Steps — Too Many Open Positions

| Step | Action | Notes |
|------|--------|-------|
| 7 | Open 10 distinct positions across different symbols using MT-01 | Use minimal quantities |
| 8 | Attempt to open an 11th position | e.g. Symbol = `GLD`, qty = 1 |
| 9 | Confirm and submit | |
| 10 | Observe result | |

**Expected:** Order rejected with message indicating max positions exceeded.

---

## Expected Results (both sub-tests)

- [ ] Server returns an error response (HTTP 4xx range)
- [ ] UI shows an error message: either inline below the form, as a toast notification, or in the blotter with status `rejected`
- [ ] The error message is human-readable (not a raw JSON dump)
- [ ] No position is opened for the rejected order

---

## Failure Indicators

- Order accepted despite exceeding position limit → risk guard not active
- UI shows success but blotter shows `rejected` with no explanation → error surfacing missing
- App crashes or shows blank error page → unhandled rejection in frontend
