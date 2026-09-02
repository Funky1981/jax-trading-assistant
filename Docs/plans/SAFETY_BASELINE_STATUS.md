# Safety Baseline Status

> **HISTORICAL / SUPPORTING STATUS NOTE** — The current development roadmap is
> `Docs/ROADMAP.md`. This document records an earlier safety-baseline work
> sequence and remains useful safety evidence; it does not control sequencing.

## Roadmap reference

The earlier plan was `Docs/plans/jax-trading-roadmap-pack`. Current phase and
package status are defined by `Docs/ROADMAP.md`.

Archived autonomous, live-trading, and broker-execution docs are historical references only. They must not be used to justify live trading, leverage, automatic order placement, or broker-execution expansion until later explicit roadmap gates pass.

## Current Safety Defaults

- Live trading is disabled by default because `ALLOW_LIVE_TRADING` must be explicitly set to `true` before live execution paths can pass their live-mode guard.
- Paper execution worker fails closed unless all required paper-mode env vars are present.
- ETF catalog allows only approved plain ETF symbols in paper mode.
- Leveraged, inverse, volatility, and ETN products remain excluded from the ETF paper universe.
- Risk config now sets `max_leverage` to `1.0`, which means no leverage during the safety-baseline phase.

## Execution-Related Env Vars

- `JAX_RUNTIME_MODE`: must be `paper` for the paper execution instruction worker.
- `JAX_TRADER_RUNTIME_MODE`: fallback runtime mode used only if `JAX_RUNTIME_MODE` is empty.
- `IB_PAPER_TRADING`: must be `true` for paper execution worker startup.
- `ALLOW_LIVE_TRADING`: must be `false` for paper execution worker startup. Setting it to `true` disables the paper worker during this restricted phase.
- `EXECUTION_ENABLED`: can initialize the broker-capable execution service and must not be enabled casually.
- `IB_BRIDGE_URL`: points at the broker bridge used by the execution service when execution is enabled.

## Broker-Capable Paths Not To Expand Yet

- `internal/modules/execution.Service.ExecuteTrade`
- `internal/modules/execution.IBClient.PlaceOrder`
- `cmd/trader.handleExecute`
- `cmd/trader.startExecutionInstructionWorker`
- `candidate_approvals -> execution_instructions -> trades` workflow

These paths exist for controlled paper workflows and must not be expanded toward live trading, leverage, or automatic order placement during the current phase.

## Leverage Status

- `config/risk-constraints.json` now sets `position_limits.max_leverage` to `1.0`.
- `risk.DefaultPolicy()` now sets `MaxLeverage` to `1.0`.
- Tests assert both defaults stay at or below `1.0` during the safety baseline.

## Tests Added Or Updated

- `cmd/trader/execution_instruction_worker_test.go`
  - Confirms the paper worker only enables under explicit paper-mode settings.
  - Confirms missing or unsafe env combinations fail closed.
  - Confirms `ALLOW_LIVE_TRADING=true` disables the paper worker.
- `internal/modules/instruments/policy_test.go`
  - Confirms `TQQQ`, `SQQQ`, `UVXY`, and `VXX` are rejected.
  - Confirms live-mode ETF trading is rejected.
- `libs/risk/policy_test.go`
  - Confirms default risk policy disables leverage.
  - Confirms repo risk config disables leverage.

## Remaining Risks

- `/api/v1/execute` is broker-capable when execution is enabled and its guards pass.
- `EXECUTION_ENABLED=true` plus a configured IB bridge can initialize broker-capable code.
- Existing archived docs contain autonomous and live-trading language. They remain historical only and should not drive implementation.
- Candidate completeness and slippage-aware risk fields are still incomplete relative to the new roadmap.

## Local Test Commands

```powershell
go test ./libs/risk
go test ./internal/modules/instruments
go test ./cmd/trader -run "TestExecutionInstructionWorkerSafety"
```

For the standard repo workflow:

```powershell
scripts/go-verify.ps1 -Mode quick -Packages ./libs/risk,./internal/modules/instruments,./cmd/trader
```
