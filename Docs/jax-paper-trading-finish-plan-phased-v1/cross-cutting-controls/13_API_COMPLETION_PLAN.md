# API Completion Plan

## Already present and useful
- signals
- recommendations
- trades
- orchestration runs
- strategies
- trading guard
- risk calc
- artifacts

## Add / complete next

### Strategy instances
- `GET /api/v1/strategy-instances`
- `GET /api/v1/strategy-instances/{id}`
- `POST /api/v1/strategy-instances`
- `PATCH /api/v1/strategy-instances/{id}`
- `POST /api/v1/strategy-instances/{id}/enable`
- `POST /api/v1/strategy-instances/{id}/disable`
- `POST /api/v1/strategy-instances/{id}/clone`

### Candidate trades
- `GET /api/v1/candidates`
- `GET /api/v1/candidates/{id}`
- `POST /api/v1/candidates/{id}/refresh`

### Approvals
- `GET /api/v1/approvals/queue`
- `GET /api/v1/approvals/{candidateId}`
- `POST /api/v1/approvals/{candidateId}/approve`
- `POST /api/v1/approvals/{candidateId}/reject`
- `POST /api/v1/approvals/{candidateId}/snooze`
- `POST /api/v1/approvals/{candidateId}/reanalyze`

### Assistant
- `POST /api/v1/chat`
- `GET /api/v1/chat/history`
- `GET /api/v1/chat/sessions/{id}`

### Gate and testing
- `GET /api/v1/gates`
- `GET /api/v1/gates/history`
- `POST /api/v1/tests/run/{gateId}`

### Streaming
- `GET /api/v1/events/stream`
