# Current State and Fit

## Current approval-related pieces
- signal approve/reject endpoints already exist
- trades API already exists
- execution engine module already exists
- trading guard and risk calc endpoints already exist

## Gaps
- approvals tied to `strategy_signals`, not richer candidate-trade objects
- no approval queue page/API model
- no expiry rules for stale approvals
- no explicit audit trail for approval decision metadata
- no separation between:
  - model/orchestration analysis
  - candidate trade
  - approved execution instruction
