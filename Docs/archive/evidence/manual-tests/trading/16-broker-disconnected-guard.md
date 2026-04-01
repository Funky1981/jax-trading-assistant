# MT-16 — Verify Orders Are Blocked When Broker Is Disconnected

**Area:** Trading Pilot Guard / Broker Connection  
**Type:** Negative / Safety  
**Priority:** P0 — Critical Path (Safety)  

---

## Objective

Verify that when the IB broker is not connected, all trade execution actions are blocked and the UI communicates this clearly.

---

## Background

`pilotStatus.brokerConnected` is polled from `/api/v1/pilot/status`.  
When `brokerConnected === false`, the UI should disable all order submission and position management actions.

---

## Preconditions

- [ ] IB Bridge is **stopped** or disconnected (stop the IB Gateway / TWS process)
- [ ] App is open at `/trading`
- [ ] Browser refreshed after stopping the bridge

---

## Test Steps

| Step | Action | Notes |
|------|--------|-------|
| 1 | With IB Bridge stopped, navigate to **Trading** page | |
| 2 | Check the **Order Ticket** panel | Look for a warning banner |
| 3 | Check the **Positions** panel | |
| 4 | Attempt to fill and submit an order | e.g. AAPL Buy Market 1 share |

---

## Expected Results

- [ ] An orange/yellow warning banner (AlertTriangle icon) appears in the Order Ticket panel: "Broker not connected"
- [ ] The **Submit Order** button is disabled
- [ ] The **Close** and **Protect** buttons in Positions panel are disabled
- [ ] The **Approve** button in Signals Queue is disabled
- [ ] The banner lists the reason: "Broker is not connected" or equivalent
- [ ] Watchlist still loads and displays cached or live prices (price feed may still work)
- [ ] Dashboard health panel shows broker as "unhealthy" or "disconnected"

---

## Failure Indicators

- Order submits and is accepted despite broker disconnection → no server-side broker guard
- Banner does not appear → `brokerConnected` flag not propagated to UI components
- Health panel shows broker as healthy while it's actually stopped → health check not reaching IB bridge
