# Frontend: Data services to add

Pattern reference: `frontend/src/data/strategy-service.ts` and `http-client.ts`.

Create:
- `frontend/src/data/instances-service.ts`
- `frontend/src/data/backtest-service.ts`
- `frontend/src/data/research-service.ts`
- `frontend/src/data/testing-service.ts`

Each should:
- call `apiClient.get/post/put`
- define minimal TS types in `frontend/src/data/types.ts` (or a new `types-research.ts` if you prefer).

Minimum TS types:
- `StrategyInstance` { instanceId, strategyId, enabled, configJson, updatedAt }
- `BacktestRunSummary` { runId, instanceId, from, to, createdAt, stats }
- `BacktestRunDetail` extends summary + trades[]
- `ResearchProject` { id, name, instanceId, configJson, createdAt }
- `TestingGateStatus` { gateId, status, lastRunAt, summary, artifactUrl? }

Acceptance criteria:
- Type-safe fetch in pages.
- Errors displayed via existing toast/alert pattern (if present).
