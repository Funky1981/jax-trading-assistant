# MT-21 — Always-on Trade Watcher: Candidate Trade Detection

**Area:** Trade Watcher / Candidates  
**Type:** Functional  
**Priority:** P1 — Core Feature  

---

## Objective

Verify that the always-on trade watcher is running continuously, generates candidate trades for qualified setups, and correctly blocks disqualified setups — without any execution occurring unless a candidate is explicitly approved.

---

## Preconditions

- [ ] Backend trader service is running (`start.ps1` or `docker compose up`)
- [ ] Database migrations 000014–000016 have been applied
- [ ] At least one strategy instance is enabled in the DB / `config/strategy-instances/`
- [ ] Kill switch is **OFF**: `config_flags.global_kill_switch = false`
- [ ] IB Bridge is connected (or paper-trading stub is active)

---

## Test Steps

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to `/approvals` in the Jax UI | Approval queue page should load |
| 2 | Wait 5–10 minutes for the watcher scan cycle | Default scan interval is 5 minutes |
| 3 | Open browser DevTools → Network → filter `events/stream` | SSE connection should show `200` and remain open |
| 4 | Observe `watcher.scan` events arriving in the SSE stream | Each scan emits a `watcher.scan` event with instance list |
| 5 | Check the DB: `SELECT status, symbol, signal_type, detected_at FROM candidate_trades ORDER BY detected_at DESC LIMIT 10;` | Rows should appear as watcher scans |
| 6 | Verify no duplicate candidates for the same instance+symbol+session | `SELECT COUNT(*) FROM candidate_trades WHERE status NOT IN ('filled','cancelled','expired') GROUP BY strategy_instance_id, symbol, session_date HAVING COUNT(*) > 1` should return 0 rows |
| 7 | Check that blocked candidates have `status = 'blocked'` and a non-null `block_reason` | |
| 8 | Verify `execution_instructions` table is empty (no execution without approval) | `SELECT COUNT(*) FROM execution_instructions` should be 0 if no approvals were granted |
| 9 | Activate the global kill switch: update `config_flags` SET `global_kill_switch = true` | |
| 10 | Wait one scan interval (5 min) | No new `watcher.scan` events should fire |
| 11 | Deactivate kill switch and confirm scans resume | |

---

## Expected Results

- [ ] `watcher.scan` SSE events arrive every ~5 minutes
- [ ] `candidate_trades` rows appear with `status IN ('detected', 'qualified', 'awaiting_approval', 'blocked')`
- [ ] No two open candidates share the same `(strategy_instance_id, symbol, session_date)` 
- [ ] Blocked candidates have `status = 'blocked'` and a non-null `block_reason`; they do NOT appear in the approval queue
- [ ] `execution_instructions` has zero rows unless a candidate was explicitly approved
- [ ] Kill switch halts all scanning; re-enabling resumes within one cycle

---

## Failure Indicators

- No SSE connection established → check CORS config and that `startFrontendAPIServer` is running
- Watcher exits silently → check logs for `startTradeWatcher` goroutine panic
- Duplicate candidates detected → dedup guard in `candidates.Store.HasOpenForInstanceSymbol` is not working
- Execution instruction appears without approval → critical bug; escalate immediately
- Kill switch has no effect → `checkKillSwitch` is not being called in the watcher loop
