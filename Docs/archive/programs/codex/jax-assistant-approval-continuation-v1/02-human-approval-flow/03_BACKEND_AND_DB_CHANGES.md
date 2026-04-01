# Backend and DB Changes

## New tables
- `candidate_approvals`
- `execution_instructions`
- optional `approval_actions_audit`

## New endpoints
- `GET /api/v1/approvals/queue`
- `GET /api/v1/approvals/{candidateId}`
- `POST /api/v1/approvals/{candidateId}/approve`
- `POST /api/v1/approvals/{candidateId}/reject`
- `POST /api/v1/approvals/{candidateId}/snooze`
- `POST /api/v1/approvals/{candidateId}/reanalyze`

## Files to add
- `cmd/trader/approval_handlers.go`
- `internal/modules/approvals/service.go`
- `internal/modules/approvals/store.go`
- `internal/modules/execution/instruction_builder.go`
