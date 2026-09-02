# Safety Baseline GPT Handoff

> **HISTORICAL / SUPPORTING HANDOFF** — Current roadmap authority is
> `Docs/ROADMAP.md`. This handoff records the earlier safety-baseline sequence.

## Context

The historical roadmap pack was:

`Docs/plans/jax-trading-roadmap-pack`

The first implementation task was the Safety Baseline. The goal was to prove Jax cannot accidentally move toward live trading, leverage, or broker execution during the current phase.

No trading features were added.
No live trading was added.
No leverage support was added.
No automatic order placement was added.
No broker execution behavior was expanded.

## A. Summary Of Changes

- Added fail-closed tests for the paper execution worker env gates.
- Added ETF/instrument policy tests for unsafe products and live-mode rejection.
- Disabled leverage during the current phase by setting max leverage to `1.0`.
- Added a canonical roadmap notice to `Docs/ROADMAP.md`.
- Added a safety status report at `Docs/plans/SAFETY_BASELINE_STATUS.md`.
- Updated the reconciliation report so it reflects the fixed leverage baseline.

## B. Files Changed

- `cmd/trader/execution_instruction_worker_test.go`
- `internal/modules/instruments/policy_test.go`
- `libs/risk/policy.go`
- `libs/risk/policy_test.go`
- `config/risk-constraints.json`
- `Docs/ROADMAP.md`
- `Docs/plans/SAFETY_BASELINE_STATUS.md`
- `Docs/plans/jax-trading-roadmap-pack/RECONCILIATION_REPORT.md`

## C. Tests Added

### Execution Worker Safety

Added tests proving the paper execution worker only enables when all of these are true:

```text
JAX_RUNTIME_MODE=paper
IB_PAPER_TRADING=true
ALLOW_LIVE_TRADING=false
```

Added fail-closed coverage for:

- missing env vars
- paper mode without `IB_PAPER_TRADING=true`
- `IB_PAPER_TRADING=true` without paper runtime mode
- `ALLOW_LIVE_TRADING=true`
- non-paper modes such as `research`

### ETF / Instrument Policy

Added tests proving the ETF catalog rejects:

- `TQQQ`
- `SQQQ`
- `UVXY`
- `VXX`
- live-mode ETF trading

These cover leveraged, inverse, volatility, ETN, and live-mode product risks.

### Leverage Baseline

Added tests proving:

- `risk.DefaultPolicy()` has `MaxLeverage <= 1.0`
- `config/risk-constraints.json` has `max_leverage <= 1.0`

## D. Safety Guarantees Now Enforced

- Leverage is disabled for the current phase.
- The repo risk config now has:

```json
"max_leverage": 1.0
```

- The default Go risk policy now uses:

```go
MaxLeverage: 1.0
```

- The paper execution worker fails closed unless every required paper-mode condition is explicitly present.
- `ALLOW_LIVE_TRADING=true` prevents the paper execution worker from running during this restricted phase.
- Unsafe ETF products remain rejected by policy.
- Live-mode ETF trading remains rejected.
- The active roadmap now states that archived/autonomous/live-trading docs are historical only.

## E. Remaining Risks

- Broker-capable code still exists in:
  - `internal/modules/execution.Service.ExecuteTrade`
  - `internal/modules/execution.IBClient.PlaceOrder`
  - `cmd/trader.handleExecute`
  - `cmd/trader.startExecutionInstructionWorker`
- `EXECUTION_ENABLED=true` plus a configured IB bridge can initialize broker-capable code.
- `/api/v1/execute` remains a broker-capable route when execution is enabled and its guards pass.
- Candidate completeness and slippage-adjusted risk are not implemented yet.
- The new roadmap pack files are currently untracked unless explicitly added to git.

## F. Commands Run

Targeted tests:

```powershell
go test ./libs/risk
go test ./internal/modules/instruments
go test ./cmd/trader -run "TestExecutionInstructionWorkerSafety"
```

Repo quick verifier:

```powershell
scripts/go-verify.ps1 -Mode quick -Packages ./libs/risk,./internal/modules/instruments,./cmd/trader
```

All passed.

## G. Exact Commands To Run Locally

```powershell
go test ./libs/risk
go test ./internal/modules/instruments
go test ./cmd/trader -run "TestExecutionInstructionWorkerSafety"
scripts/go-verify.ps1 -Mode quick -Packages ./libs/risk,./internal/modules/instruments,./cmd/trader
```

## H. Current Git Status Notes

Expected modified files:

```text
Docs/ROADMAP.md
cmd/trader/execution_instruction_worker_test.go
config/risk-constraints.json
internal/modules/instruments/policy_test.go
libs/risk/policy.go
libs/risk/policy_test.go
Docs/plans/SAFETY_BASELINE_STATUS.md
Docs/plans/jax-trading-roadmap-pack/RECONCILIATION_REPORT.md
```

The roadmap pack folder may still appear as untracked:

```text
Docs/plans/jax-trading-roadmap-pack/
```

## I. Recommended Next Step

Do not implement candidate-model, scoring, paper execution, or broker changes yet.

Next safe step is to review and accept this Safety Baseline, then proceed to the next roadmap phase:

```text
Structured Trade Candidate Model
```

That next phase should still avoid:

- live trading
- leverage
- automatic order placement
- broker execution expansion
