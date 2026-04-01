# MT-13 — Add and Remove Symbols from the Watchlist

**Area:** Watchlist Panel  
**Type:** Functional  
**Priority:** P1 — High  

---

## Objective

Verify that a user can add new symbols to the watchlist and remove existing ones, with live price data reflecting the change.

---

## Preconditions

- [ ] App is open at `/trading` or Dashboard with Watchlist panel visible
- [ ] Market data is connected (prices should display for existing symbols)

---

## Test Steps — Add

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to the **Watchlist** panel | |
| 2 | Locate the symbol input field (text box with "+" / Add button) | |
| 3 | Type `NVDA` in the symbol input | Uppercase or lowercase accepted |
| 4 | Click the **Add** (+) button or press Enter | |
| 5 | Observe the watchlist table | |

**Expected (Add):**
- [ ] `NVDA` row appears in the watchlist table
- [ ] Price and Change columns start populating within a few seconds (market data connected)
- [ ] The input field clears after adding
- [ ] Duplicate symbols are handled — adding NVDA twice does not create two rows

---

## Test Steps — Remove

| Step | Action | Notes |
|------|--------|-------|
| 6 | Locate the `NVDA` row just added | |
| 7 | Click the **Remove** (🗑 Trash) icon on the NVDA row | |
| 8 | Observe the watchlist table | |

**Expected (Remove):**
- [ ] `NVDA` row is immediately removed from the table
- [ ] No error appears
- [ ] Other symbols remain unaffected

---

## Failure Indicators

- Add button disabled or input not responding → hook mutation not wired
- Symbol added but no price data appears after 10s → market data subscription not updated
- Remove icon absent → column config missing delete button
- Removing one symbol removes another → wrong symbol passed to delete mutation
