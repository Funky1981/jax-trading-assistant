# MT-17 — Collapse and Expand Trading Dashboard Panels

**Area:** Trading Page — Panel Layout  
**Type:** UI / State Persistence  
**Priority:** P2 — Medium  

---

## Objective

Verify that all panels on the Trading page can be individually collapsed and expanded, and that the state is persisted across page refreshes via localStorage.

---

## Preconditions

- [ ] App is open at `/trading`
- [ ] All panels are in their default state (all expanded)

---

## Test Steps — Individual Toggle

| Step | Action | Notes |
|------|--------|-------|
| 1 | Click the collapse toggle (chevron icon) on the **Order Ticket** panel header | Panel should animate closed |
| 2 | Verify the panel body is hidden | Only the header row should remain visible |
| 3 | Click the toggle again | Panel should expand |
| 4 | Repeat for: **Watchlist**, **Positions**, **Risk Summary**, **Trade Blotter**, **Price Chart**, **Strategy Monitor**, **Signals Queue**, **AI Assistant** | Each panel collapses independently |

---

## Test Steps — Collapse All / Expand All

| Step | Action | Notes |
|------|--------|-------|
| 5 | Click the **Collapse All** button (ChevronUp icon) at the top of the Trading page | |
| 6 | Verify all panels collapse simultaneously | |
| 7 | Click **Expand All** (ChevronDown) | |
| 8 | Verify all panels expand simultaneously | |

---

## Test Steps — State Persistence

| Step | Action | Notes |
|------|--------|-------|
| 9 | Collapse just the **Watchlist** and **Price Chart** panels | |
| 10 | Refresh the page (F5) | |
| 11 | Observe panel states after reload | |

---

## Expected Results

- [ ] Each panel collapses and expands independently when its toggle is clicked
- [ ] "Collapse All" collapses all 9 panels at once
- [ ] "Expand All" expands all 9 panels at once
- [ ] After page refresh, Watchlist and Price Chart remain collapsed (state restored from localStorage under key `jax-trading-panels`)
- [ ] All other panels remain expanded

---

## Failure Indicators

- All panels collapse together when only one toggle is clicked → shared state rather than per-panel toggle
- Expand/Collapse All buttons missing → buttons not rendered
- State not persisted after refresh → localStorage write not happening or wrong storage key
