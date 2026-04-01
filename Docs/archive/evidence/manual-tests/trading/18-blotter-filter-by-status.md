# MT-18 — Filter Blotter by Order Status

**Area:** Trade Blotter Panel / Blotter Page  
**Type:** UI / Filtering  
**Priority:** P2 — Medium  

---

## Objective

Verify that the status filter dropdown in the Trade Blotter narrows the displayed rows correctly.

---

## Preconditions

- [ ] Multiple orders exist in different statuses: `pending`, `filled`, `cancelled`, `rejected`
  *(Create these using earlier test cases MT-01, MT-03, MT-06)*
- [ ] App is open at `/blotter` or `/trading` with the Trade Blotter panel expanded

---

## Test Steps

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to the **Blotter** page or Trade Blotter panel | |
| 2 | Locate the **Status Filter** dropdown (default: "All") | |
| 3 | Select `filled` from the dropdown | |
| 4 | Observe the blotter rows | Only filled orders should be visible |
| 5 | Change the filter to `pending` | |
| 6 | Observe the rows | Only pending orders visible |
| 7 | Change the filter to `cancelled` | |
| 8 | Observe the rows | |
| 9 | Change the filter back to `All` | |
| 10 | Observe the rows | All orders should be visible again |

---

## Expected Results

- [ ] Status filter dropdown is visible and functional
- [ ] Selecting `filled` shows only filled orders and hides pending/cancelled/rejected rows
- [ ] Selecting `pending` shows only pending orders
- [ ] Selecting `cancelled` shows only cancelled orders
- [ ] Selecting `All` restores all rows
- [ ] Row count visually matches the filter (no ghost rows from hidden items)
- [ ] Sorting still works within filtered results (click column headers)

---

## Failure Indicators

- Filter dropdown missing → Select component not rendered
- All rows visible regardless of filter → filter state not passed to table
- Wrong rows hidden → filter comparing wrong field (e.g. filtering on workflow instead of status)
- Table empties for `All` filter → all rows accidentally filtered out
