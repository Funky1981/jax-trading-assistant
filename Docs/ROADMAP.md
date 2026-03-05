# Roadmap (Condensed)

## Near-Term Priorities

1. **Artifact Validation**
   - Link artifact approvals to real trust-gate execution evidence.
   - Extend artifact validation from integrity checks to golden/replay-backed validation where appropriate.
2. **Runtime/Docs Alignment**
   - Keep docs aligned to the active `cmd/trader` + `cmd/research` topology.
   - Remove stale references to superseded `services/jax-*` runtime paths.
3. **Market Data Ingestion**
   - Wire IB bridge stream into ingestion/storage.
   - Close remaining provider-method gaps in the market-data adapters.
4. **Authentication Hardening**
   - Replace placeholder login behavior with the intended auth flow.
5. **Reflection Loop**
   - Schedule and persist `memory.reflect` outputs.

## Mid-Term

- Observability (metrics, traces, log retention).
- CI + linting gates + documentation hygiene.
- Clarify Agent0/Dexter mock-vs-live boundaries and repo-local failure handling.

## Long-Term

- Full autonomous loop: ingest → orchestrate → reflect → improve strategies.
- Production hardening and infra automation.

## Reference (Archived)

Detailed historical plans and reports are preserved in `Docs/archive/root/`.
