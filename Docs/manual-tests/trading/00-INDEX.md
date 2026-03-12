# Manual Trading Test Plan — Index

All tests below are **paper-trading scenarios** performed by a human tester in the Jax UI.  
Run these after every release or whenever trading-path code changes.

| # | File | Scenario |
|---|------|----------|
| 01 | [01-market-order-buy.md](01-market-order-buy.md) | Place a market buy order |
| 02 | [02-market-order-sell-short.md](02-market-order-sell-short.md) | Place a market sell (short) order |
| 03 | [03-limit-order-buy.md](03-limit-order-buy.md) | Place a limit buy order |
| 04 | [04-stop-order-entry.md](04-stop-order-entry.md) | Place a stop-entry order |
| 05 | [05-order-with-stop-loss-and-take-profit.md](05-order-with-stop-loss-and-take-profit.md) | Entry order with stop-loss and take-profit protection |
| 06 | [06-cancel-pending-order.md](06-cancel-pending-order.md) | Cancel a pending order from the blotter |
| 07 | [07-close-position-market.md](07-close-position-market.md) | Close an open position at market |
| 08 | [08-close-position-limit.md](08-close-position-limit.md) | Close an open position with a limit exit |
| 09 | [09-protect-position.md](09-protect-position.md) | Add stop-loss and take-profit to an open position |
| 10 | [10-signal-approve-and-fill.md](10-signal-approve-and-fill.md) | Approve a strategy signal and verify fill |
| 11 | [11-signal-reject.md](11-signal-reject.md) | Reject a strategy signal |
| 12 | [12-signal-ai-analyze.md](12-signal-ai-analyze.md) | Request AI analysis on a pending signal |
| 13 | [13-watchlist-add-remove.md](13-watchlist-add-remove.md) | Add and remove symbols from the watchlist |
| 14 | [14-risk-limit-breach.md](14-risk-limit-breach.md) | Attempt order that breaches risk constraints |
| 15 | [15-pilot-readonly-guard.md](15-pilot-readonly-guard.md) | Verify trading is blocked when pilot is read-only |
| 16 | [16-broker-disconnected-guard.md](16-broker-disconnected-guard.md) | Verify orders blocked when broker is disconnected |
| 17 | [17-panel-collapse-expand.md](17-panel-collapse-expand.md) | Collapse and expand trading dashboard panels |
| 18 | [18-blotter-filter-by-status.md](18-blotter-filter-by-status.md) | Filter blotter by order status |
| 19 | [19-order-ticket-standalone.md](19-order-ticket-standalone.md) | Place order from standalone Order Ticket page |
| 20 | [20-portfolio-risk-review.md](20-portfolio-risk-review.md) | Review portfolio exposure and risk metrics |

## Prerequisites (apply to all tests)

- App is running at `http://localhost:5173` (dev) or the deployed URL
- Backend trader service is running on port `8081`
- IB Bridge is running (or paper-trading mode is active)
- Auth is disabled (`{ enabled: false }`) **or** you are logged in with a valid account
- All tests assume **paper trading mode** — never run on a live-funded account without explicit intent

## Pass / Fail Convention

Each test ends with an **Expected Result** block.  
Mark a test **PASS** if every expected result is met exactly.  
Mark it **FAIL** if any expected result is not met — capture a screenshot and note the deviation.
