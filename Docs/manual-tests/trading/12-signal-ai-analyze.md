# MT-12 — Request AI Analysis on a Pending Signal

**Area:** Signals Queue Panel — AI Assistant  
**Type:** Functional  
**Priority:** P2 — Medium  

---

## Objective

Verify that the AI analysis feature on the Signals Queue panel can be triggered and that the result is displayed on the signal card.

---

## Preconditions

- [ ] At least one pending signal in the Signals Queue
- [ ] AI service / Dexter integration is configured and reachable
- [ ] App is open at `/trading` or Dashboard

---

## Test Steps

| Step | Action | Notes |
|------|--------|-------|
| 1 | Navigate to the **Signals Queue** panel | |
| 2 | Locate a pending signal card | |
| 3 | Click the **Analyze** button (✨ Sparkles icon) on the signal card | A loading state should begin |
| 4 | Wait for the AI analysis to complete | May take several seconds |
| 5 | Observe the signal card for the AI analysis result | |

---

## Expected Results

- [ ] Clicking Analyze triggers a loading/spinner state on the button
- [ ] After completion, an AI analysis section appears below the signal details
- [ ] The analysis includes a recommendation (e.g. "Proceed with caution", "Strong signal") and reasoning text
- [ ] The signal status dot reflects the analysis state: green (completed), grey (not run), red (failed)
- [ ] If AI analysis fails, a failure message or "failed" status appears — not a blank card
- [ ] The Approve/Reject buttons remain functional after analysis loads

---

## Failure Indicators

- Analyze button triggers nothing → `onAnalyze` callback not wired
- Button spins forever → API call hanging, no timeout or error handling
- Analysis status shows "completed" but no text appears → response shape mismatch
- Card layout breaks after analysis text renders → CSS overflow issue
