# Phase 5 — Assistant and AI Audit

## Objective
Add a true assistant layer that can answer questions and explain scenarios without becoming the trading authority.

## Required deliverables

### 5.1 Chat backend
Add endpoints:
- `POST /api/v1/chat`
- `GET /api/v1/chat/history`
- `GET /api/v1/chat/sessions/{id}`

Files:
- `cmd/trader/chat_handlers.go`
- `internal/modules/chat/service.go`
- `internal/modules/chat/tool_router.go`
- `internal/modules/chat/session_store.go`

### 5.2 Tool router
Assistant tools should support:
- get candidate
- get signal
- get trade
- get strategy
- get strategy instance
- get orchestration run
- search research runs
- query research-only RAG
- explain blockers

### 5.3 Assistant boundary enforcement
Rules:
- no order placement
- no approval on behalf of user
- no config mutation without explicit action
- no risk override
- no trust gate override

### 5.4 AI audit model
Add tables:
- `ai_decisions`
- `ai_decision_acceptance`

Store:
- purpose
- prompt/template version
- model/provider
- input payload reference
- raw output
- structured output
- acceptance decision
- linked candidate/run/trade IDs
