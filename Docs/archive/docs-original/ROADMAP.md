# Roadmap (Condensed)

## Near-Term Priorities

1. **Orchestration API**
   - Implement HTTP endpoints expected by the frontend.
   - Add run tracking + retrieval endpoints.
2. **Strategy Signals**
   - Background job for signal generation.
   - Storage + performance tracking endpoints.
3. **Market Data Ingestion**
   - Wire IB bridge stream into ingestion/storage.
4. **Agent0 HTTP Service**
   - Provide `/v1/plan` and `/v1/execute` endpoints.
5. **Reflection Loop**
   - Schedule and persist `memory.reflect` outputs.

## Mid-Term

- Observability (metrics, traces, log retention).
- CI + linting gates + documentation hygiene.

## Long-Term

- Full autonomous loop: ingest → orchestrate → reflect → improve strategies.
- Production hardening and infra automation.

## Reference (Archived)

Detailed historical plans and reports are preserved in `Docs/archive/root/`.
