# Current Work Branch Capability Baseline

## Implemented strongly
- Two authoritative runtimes:
  - `cmd/trader`
  - `cmd/research`
- Shadow validator runtime:
  - `cmd/shadow-validator`
- Frontend API in trader runtime:
  - signals, recommendations, trades, strategies, orchestrate runs, trading guard, risk calc
- Research runtime orchestration and backtest endpoint
- Artifact domain/store/handlers and migrations
- Backtest module:
  - `internal/modules/backtest/engine.go`
- Execution module:
  - `internal/modules/execution/engine.go`
- Strategy instance loading from JSON into DB
- Auth shell and login flow in frontend
- Strong docs / ADR / ops / migration structure

## Partial
- strategy instances:
  - backend bootstrap exists
  - management workflow/UI missing
- signal approval:
  - signal approve/reject exists
  - richer candidate approval queue missing
- AI/orchestration:
  - orchestration runs exist
  - full AI audit/replay model missing
- trust and validation:
  - artifact validation exists
  - complete gates and no-fake-data enforcement missing

## Missing
- real event-data completion
- candidate trade model
- always-on watcher
- approval queue
- execution instruction layer
- Research / Analysis / Testing / Approvals / Assistant pages
- assistant chat backend + frontend
- RAG research-only module integration
- ai_decisions / acceptance model
- provenance enforcement as hard platform rule
- flatten proof / reconciliation / gate dashboard
