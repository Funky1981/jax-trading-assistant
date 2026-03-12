# MT-19 — Place Order from Standalone Order Ticket Page

**Area:** Order Ticket Page (`/order-ticket`)  
**Type:** Functional  
**Priority:** P1 — High  

---

## Objective

Verify that the standalone Order Ticket page (`/order-ticket`) functions identically to the panel embedded in the Trading page, and that navigation to it works from the sidebar.

---

## Preconditions

- [ ] App is running and auth is satisfied
- [ ] Broker is connected / paper mode active
- [ ] Pilot is not read-only

---

## Test Steps

| Step | Action | Notes |
|------|--------|-------|
| 1 | Click **Order Ticket** in the left navigation sidebar | |
| 2 | Verify the page loads at `/order-ticket` | Page title: "Order Ticket" |
| 3 | Verify the page subtitle: "Submit broker orders through the IB bridge..." | |
| 4 | Verify the **DataSourceBadge** is visible in the panel (paper/live indicator) | |
| 5 | Fill in: Symbol = `IWM`, Side = `Buy`, Type = `Market`, Qty = `5` | |
| 6 | Add Stop Loss = `200.00` and Take Profit = `230.00` | |
| 7 | Click **Submit Order** | |
| 8 | Confirm in the dialog | |
| 9 | Navigate to **Blotter** page | |
| 10 | Verify the IWM order appears | |

---

## Expected Results

- [ ] Navigation to `/order-ticket` from sidebar works
- [ ] Page renders the full Order Ticket panel (not collapsed by default — `isOpen=true` always)
- [ ] Order submission works identically to embedded panel on Trading page
- [ ] IWM order appears in blotter after submission
- [ ] The page is constrained to `max-w-2xl` width (prevents super-wide form on large screens)

---

## Failure Indicators

- Sidebar link "Order Ticket" is missing → nav item not registered
- Page renders blank → route not registered in App.tsx or component missing export
- Panel is collapsed on standalone page → `isOpen` prop defaulting to false on page
- Order ticket does not submit → same as MT-01 failure modes
