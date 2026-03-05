# Current Work Branch Baseline

## Strengths on work branch
- cmd/trader runtime + frontend API
- cmd/research runtime
- cmd/shadow-validator
- artifacts domain/store/handlers
- EJ layer modules and migrations
- backtest and execution modules
- expanded CI/docs/ADR work

## Key risks / gaps
- Fake UTCP local backtest path still exists (`libs/utcp/backtest_local_tools.go`)
- Event-trading-specific pipeline incomplete
- AI audit/replay not yet fully enforced
- Research/Analysis/Testing route wiring/integration may still be incomplete
