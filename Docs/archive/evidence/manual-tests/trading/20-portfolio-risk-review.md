# MT-20 — Review Portfolio Exposure and Risk Metrics

**Area:** Portfolio Page / Risk Summary Panel  
**Type:** Functional / Read-Only Review  
**Priority:** P1 — High  

---

## Objective

Verify that the Portfolio page correctly displays current exposure, daily P&L, drawdown, and per-position risk metrics, including progress bars that change colour under stress.

---

## Preconditions

- [ ] At least 2–3 open positions exist (create using MT-01, MT-02)
- [ ] App is open at `/portfolio`
- [ ] Market data is connected (prices updating)

---

## Test Steps — Risk Summary Panel

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to **Portfolio** page | |
| 2 | Observe the **Risk Summary** panel | |
| 3 | Check the **Exposure** row: current value, limit, and utilisation % | |
| 4 | Observe the **Exposure progress bar** colour | Green = safe, Yellow = warning (>60%), Red = danger (>80%) |
| 5 | Check the **Daily P&L** figure | Should reflect current unrealised + realised P&L for the session |
| 6 | Check the **Drawdown** metric | Should be 0% or positive if profitable session |
| 7 | Check the **DataSourceBadge** in the panel header | Should show PAPER or LIVE |

---

## Test Steps — Positions Panel

| Step | Action | Notes |
|------|--------|-------|
| 8 | Observe the **Positions** panel below Risk Summary | |
| 9 | Verify columns: Symbol, Qty, Avg Price, Market Price, Unrealised P&L, % Change | |
| 10 | Click the **Sort** arrow on the "Unrealised P&L" column | Orders by P&L ascending → descending |
| 11 | Verify winning positions show green P&L, losing positions show red | |
| 12 | Verify the **Close** and **Protect** action buttons are visible on each row | |

---

## Expected Results

- [ ] Risk Summary displays: Exposure ($), Daily P&L ($), and at least one progress bar
- [ ] Progress bar is green when utilisation < 60%, yellow 60–80%, red > 80%
- [ ] Positions table shows all current open positions with live market prices
- [ ] P&L values update as prices move (or on next poll cycle)
- [ ] Sorting by any column works without error
- [ ] DataSourceBadge correctly reflects paper vs. live mode
- [ ] No "—" placeholders in columns that should have data (market price is live)

---

## Failure Indicators

- Exposure shows $0 with positions open → portfolio aggregation not summing positions
- Progress bar always green regardless of utilisation → progress value not calculated
- P&L shows `—` for all positions → market price not being fetched or mapped
- Sorting a numeric column sorts lexicographically (e.g. "9" before "10") → wrong column type definition in table config
- DataSourceBadge missing → component not imported or prop not passed
