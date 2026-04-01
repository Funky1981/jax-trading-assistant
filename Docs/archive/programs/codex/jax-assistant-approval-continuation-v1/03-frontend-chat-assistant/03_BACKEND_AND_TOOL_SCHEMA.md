# Backend and Tool Schema

## New endpoints
- `POST /api/v1/chat`
- `GET /api/v1/chat/history`
- `GET /api/v1/chat/sessions/{id}`

## Suggested transport
- WebSocket for chat streaming
- SSE optional for assistant progress updates
- keep live dashboard updates separate from chat transport

## Tool calls the assistant should support
1. `get_candidate_trade(candidateId)`
2. `get_signal(signalId)`
3. `get_trade(tradeId)`
4. `get_strategy(strategyId)`
5. `get_strategy_instance(instanceId)`
6. `get_orchestration_run(runId)`
7. `search_research_runs(filters)`
8. `query_rag_research(query, filters)`
9. `explain_trade_blockers(candidateId)`

## Files to add
- `cmd/trader/chat_handlers.go`
- `internal/modules/chat/service.go`
- `internal/modules/chat/tool_router.go`
- `internal/modules/chat/session_store.go`
