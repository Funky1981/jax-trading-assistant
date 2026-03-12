# MT-15 — Verify Trading Is Blocked When Pilot Is Read-Only

**Area:** Trading Pilot Guard / Order Ticket / Positions  
**Type:** Negative / Safety  
**Priority:** P0 — Critical Path (Safety)  

---

## Objective

Verify that when the Trading Pilot is in read-only mode, all order submission and position action buttons are disabled and a clear warning banner is displayed.

---

## Background

The Pilot Status Banner appears whenever `pilotStatus.readOnly === true`.  
Read-only mode is set by the backend when safety conditions aren't met (e.g. max consecutive losses reached, flatten required, or manual override).

---

## Preconditions

- [ ] The backend is configured to respond with `readOnly: true` on `/api/v1/pilot/status`  
  *(Trigger this by: reaching max consecutive losses, or temporarily modifying the pilot config)*
- [ ] App is open at `/trading`
- [ ] Browser has refreshed after the read-only condition is active

---

## Test Steps

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to **Trading** page | |
| 2 | Observe the **Order Ticket** panel | Look for a red or orange banner |
| 3 | Observe the **Positions** panel | |
| 4 | Observe the **Watchlist** panel | |
| 5 | Try to fill in the Order Ticket form and click **Submit Order** | |

---

## Expected Results

- [ ] A red Pilot Status Banner appears at the top of the Order Ticket panel (ShieldAlert icon, red text)
- [ ] The **Submit Order** button is disabled (`disabled` attribute set) when `pilotStatus.readOnly` is true
- [ ] The banner lists the reason(s) why trading is blocked (e.g. "Max consecutive losses reached")
- [ ] The **Close** and **Protect** buttons in the Positions panel are also disabled
- [ ] A similar warning appears on the Watchlist panel (add/remove may still work — watchlist is not a trade action)
- [ ] Signal Approve button is disabled in the Signals Queue

---

## Failure Indicators

- Order submits successfully despite read-only mode → client guard not checking `pilotStatus.readOnly`
- Banner does not appear → pilot status not polled or not rendered in panel
- Banner appears but Submit button is still clickable → disabled state not tied to pilot status
- Reason list is empty even when reasons are provided → reasons prop not rendered
