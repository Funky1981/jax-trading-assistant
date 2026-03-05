# Roadmap (Condensed)

## Near-Term Priorities

1. **Artifact API Coverage**
   - Add focused tests for filtered listing states.
   - Add promotion edge-case coverage.
   - Add persistence-failure path tests.
2. **Runtime/Docs Alignment**
   - Keep operator docs aligned to the active `cmd/trader` + `cmd/research` topology.
   - Remove stale references to superseded `services/jax-*` runtime paths in non-archived docs.
3. **Market Data Ingestion**
   - Tighten ingestion/storage validation for IB bridge sourced data.
   - Close any remaining provider-method gaps in market-data adapters.
4. **Authentication Hardening**
   - Continue tightening token/session controls and operational defaults.
5. **Reflection Loop**
   - Schedule and persist `memory.reflect` outputs.

## Mid-Term

- Observability maturity (metrics, traces, log retention, dashboards).
- CI/lint/test gate hardening and documentation hygiene.
- Clearer Agent0/Dexter mock-vs-live boundaries and failure handling.

## Long-Term

- Full autonomous loop: ingest -> orchestrate -> reflect -> improve strategies.
- Broader production hardening and infra automation.

## Reference (Archived)

Detailed historical plans and reports are preserved in `Docs/archive/root/`.
