# Implementation tasks (Codex)

## Backend
1) Add DB migrations for:
   - `ai_decisions`, `ai_decision_acceptance`
   - `runs`, `run_artifacts`, `test_runs`, `gate_status`
2) Add an `AuditService` interface with:
   - `LogAuditEvent`
   - `LogAIDecision`
   - `LogAIAcceptance`
3) In the pipeline:
   - wrap every AI call with `LogAIDecision`
   - schema-validate outputs; on failure log `invalid` and continue deterministically
4) Add endpoints:
   - `GET /api/v1/runs`
   - `GET /api/v1/runs/{id}`
   - `GET /api/v1/runs/{id}/timeline`
   - `GET /api/v1/ai-decisions?runId=&instanceId=`
   - `GET /api/v1/ai-decisions/{id}`
   - `GET /api/v1/gates`
   - `GET /api/v1/test-runs?type=`
5) Ensure every request has `correlation_id` and is stored.

## Frontend
1) Analysis page:
   - add Run selector
   - add Decision Timeline tab
2) Add AI Decision drawer/modal
3) Testing page:
   - show gate statuses with history
   - show last test run artifacts

## Determinism rules
- AI output never directly becomes an order.
- AI output must be schema validated and then accepted/rejected by deterministic rules.
- Every acceptance/rejection is stored.

Definition of done:
- You can click a trade and see:
  - upstream AI decisions (if any)
  - deterministic rules applied
  - order intent
  - broker status
  - fills and P/L
