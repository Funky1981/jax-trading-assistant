# MT-22 — Human Approval Flow: Approve, Reject, Snooze, and Re-analyse

**Area:** Approvals Page (`/approvals`)  
**Type:** Functional  
**Priority:** P0 — Critical Path  

---

## Objective

Verify the full human approval lifecycle for a candidate trade: approve → execution instruction created; reject → no execution; snooze → candidate returns to queue; re-analyse → candidate flagged for re-evaluation.

---

## Preconditions

- [ ] MT-21 has been run and at least one candidate is in `status = 'awaiting_approval'`
- [ ] App is open and authenticated
- [ ] Broker / paper-trading is connected
- [ ] Navigate to `/approvals`

---

## Test Steps — Approve Path

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to `/approvals` | Approval queue should show at least one candidate |
| 2 | Review a candidate row: symbol, direction, confidence, entry/stop/target, reasoning | |
| 3 | Expand the reasoning section | Should show AI-generated rationale |
| 4 | Click **Approve** | Confirm dialog or immediate action |
| 5 | Verify the candidate row disappears from the queue (status changed to `approved`) | |
| 6 | Check DB: `SELECT status FROM candidate_trades WHERE id = '<id>'` → `approved` | |
| 7 | Check DB: `SELECT COUNT(*) FROM execution_instructions WHERE candidate_id = '<id>'` → 1 | |
| 8 | Check `execution_instructions.status` → `pending` initially | |

---

## Test Steps — Reject Path

| Step | Action | Notes |
|------|--------|-------|
| 1 | Pick a different awaiting candidate | |
| 2 | Click **Reject** | |
| 3 | Candidate disappears from queue | |
| 4 | DB: `candidate_trades.status` → `rejected` | |
| 5 | DB: no new `execution_instructions` row for this candidate | |

---

## Test Steps — Snooze Path

| Step | Action | Notes |
|------|--------|-------|
| 1 | Pick a different awaiting candidate | |
| 2 | Select a duration from the snooze dropdown (options: **1h**, **4h**, **24h**) | Default is 4h |
| 3 | Click **Snooze** | |
| 4 | Candidate remains in queue (status stays `awaiting_approval`) | |
| 5 | DB: `candidate_approvals` has a new row with `decision = 'snoozed'` and non-null `snooze_until` | |
| 6 | DB: `snooze_until ≈ NOW() + <selected hours>` | e.g. NOW() + 4 hours if 4h was chosen |

---

## Test Steps — Decision Notes

| Step | Action | Notes |
|------|--------|-------|
| 1 | Pick any awaiting candidate | |
| 2 | Click **Add notes** below the action buttons | A text area appears (toggle — clicking again hides it) |
| 3 | Type a note, e.g. "Confidence too low, retrying later" | |
| 4 | Click **Snooze** or **Reject** | Notes are submitted alongside the decision |
| 5 | DB: `candidate_approvals.notes` = the text you entered | |
| 6 | Click **Add notes** again | Text area should collapse (label changes to **Hide notes**) |

---

## Test Steps — Re-analyse Path

| Step | Action | Notes |
|------|--------|-------|
| 1 | Pick a different awaiting candidate | |
| 2 | Click **Re-analyse** | |
| 3 | Candidate remains in queue | |
| 4 | DB: `candidate_approvals.decision = 'reanalysis_requested'` for this candidate | |

---

## Test Steps — Expired Candidate Guard

| Step | Action | Notes |
|------|--------|-------|
| 1 | In the DB, set `expires_at = NOW() - interval '1 second'` on an awaiting candidate | Simulates a TTL-expired candidate |
| 2 | Attempt to approve it via the API: `POST /api/v1/approvals/<id>/approve` | |
| 3 | Verify the API returns an error (400/422 or similar) with message about expiry | |
| 4 | Verify no `execution_instructions` row was created | |

---

## Expected Results

- [ ] Approve → `candidate_trades.status = approved` + one `execution_instructions` row with `status = pending`
- [ ] Reject → `candidate_trades.status = rejected` + **zero** new `execution_instructions` rows
- [ ] Snooze → candidate stays `awaiting_approval` + approval row with `decision = snoozed` + `snooze_until` set
- [ ] Re-analyse → candidate stays `awaiting_approval` + approval row with `decision = reanalysis_requested`
- [ ] Expired candidate → API returns error; no instruction created
- [ ] Every approval action publishes an SSE event (visible in DevTools Network tab)

---

## Failure Indicators

- **Approve creates no execution instruction** → `buildInstruction` in `approvals.Service` not called or failing silently
- **Reject creates an execution instruction** → Critical bug; decision constant mismatch
- **Expired candidate is approved** → Expiry guard in `approvals.Service.Decide` is broken
- **Queue does not refresh after action** → TanStack Query invalidation not firing in `ApprovalsPage`
- **No SSE event for approval** → `publishEvent` not called in `approval_handlers.go`
