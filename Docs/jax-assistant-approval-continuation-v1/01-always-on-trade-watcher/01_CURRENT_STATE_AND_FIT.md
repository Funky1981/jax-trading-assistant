# Current State and Fit

## Current usable pieces in `work`
- `cmd/trader/frontend_api.go`
  - already exposes signal/recommendation/trade APIs
- `cmd/trader/strategy_instances_loader.go`
  - already loads JSON strategy instances into `strategy_instances`
- `cmd/research/main.go`
  - already provides orchestration and backtest runtime
- `config/strategy-instances/*.json`
  - existing path for bootstrapping instance definitions

## What is missing for always-on scanning
- background watcher loop in `cmd/trader`
- polling/stream scheduling for enabled instances
- candidate-trade generation contract
- instance mode separation (`research`, `paper`, `live`)
- SSE or websocket event stream for frontend live updates
- candidate status model distinct from final executed trade
