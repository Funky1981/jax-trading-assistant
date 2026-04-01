# Phase 3 — Approval Queue and Paper Execution

## Objective
Turn candidate trades into controlled paper executions only after explicit human approval.

## Current state
- signal approve/reject exists
- execution module exists
- missing: candidate approval model and execution instruction flow

## Required deliverables

### 3.1 Approval queue model
New tables:
- `candidate_approvals`
- `execution_instructions`

### 3.2 Approval actions
Support:
- approve
- reject
- snooze
- reanalyze

Endpoints:
- `GET /api/v1/approvals/queue`
- `GET /api/v1/approvals/{candidateId}`
- `POST /api/v1/approvals/{candidateId}/approve`
- `POST /api/v1/approvals/{candidateId}/reject`
- `POST /api/v1/approvals/{candidateId}/snooze`
- `POST /api/v1/approvals/{candidateId}/reanalyze`

### 3.3 Execution instruction layer
Execution engine must consume explicit instructions, not raw candidates.

Add:
- `internal/modules/execution/instruction_builder.go`
- `internal/modules/approvals/service.go`
- `internal/modules/approvals/store.go`

### 3.4 Paper execution path
Implement paper execution states:
- submitted
- accepted
- partially_filled
- filled
- cancelled
- expired

### 3.5 Flatten-by-close
Add:
- same-day flatten scheduler
- paper flatten proof artifact
- residual position/order detection

### 3.6 Reconciliation
At minimum for paper:
- candidate -> approval -> execution instruction -> trade -> fill linkage
- P/L summary by day
- mismatch detection
