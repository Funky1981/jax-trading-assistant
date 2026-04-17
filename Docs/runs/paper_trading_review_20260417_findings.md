# Paper Trading Review Findings

- Date: 2026-04-17
- Scope: current work branch review for paper-trading production readiness
- Reviewer basis: source audit plus live Docker stack verification on the local/dev paper stack
- Verdict: not production-ready for paper trading

## Summary

The memory and research path is materially improved and operational in local mode, but the trader and broker-readiness path still has hard blockers. The main problem is that several readiness surfaces report healthy while the live trading path is not actually usable.

## Findings

### Finding 1

- Severity: P0
- Title: Trader startup allows a zero-strategy paper runtime
- Affected file: `cmd/trader/main.go`
- Source reference: lines 116-123 in the reviewed branch

Issue:

The trader logs the number of approved strategies loaded but does not fail startup when the registry is empty. In the live stack, the process came up and the container stayed healthy while the signal-generator health path later failed with `no strategies registered`.

Why it matters:

Paper trading should not run without an approved strategy set. Treating that as a deferred health error instead of a startup blocker allows an unusable runtime to be deployed and trusted by operators.

Live evidence:

- `http://localhost:8100/health` returned `503`
- Response body: `{"error":"no strategies registered","service":"jax-trader","status":"unhealthy"}`

### Finding 2

- Severity: P0
- Title: IB bridge can be "ready" while serving zero-priced quotes
- Affected file: `services/ib-bridge/ib_client.py`
- Source reference: lines 347-365 in the reviewed branch

Issue:

The bridge readiness path only checks IB socket connectivity. The quote handler waits a fixed two seconds and then returns HTTP `200` with zero-valued prices when no usable quote fields are populated.

Why it matters:

Paper trading should not treat the broker quote path as ready when the bridge is returning `price=0.0`, `bid=0.0`, and `ask=0.0`. That is a trading-truth failure, not a harmless degradation.

Live evidence:

- `http://localhost:8092/ready` returned `200`
- `http://localhost:8092/quotes/AAPL` returned `price=0.0`
- `http://localhost:8092/quotes/MSFT` returned `price=0.0`
- `http://localhost:8092/quotes/SPY` returned `price=0.0`
- `jax-trader` logs showed repeated `no market data provider available` failures caused by zero-price IB quotes

### Finding 3

- Severity: P1
- Title: Paper mode does not enforce the provider policy it declares
- Affected file: `cmd/trader/main.go`
- Source reference: lines 368-375 in the reviewed branch

Issue:

`ModePaper` is declared as strict-provider and no-synthetic mode, but event/news provider validation only hard-fails in `live` or production-like environments. In `paper` mode, missing Polygon/Finnhub credentials only produce a warning and the process continues degraded.

Why it matters:

Paper mode is supposed to behave like a pre-live proving ground. Allowing degraded provider coverage in paper mode weakens that guarantee and lets provider gaps survive until later environments.

### Finding 4

- Severity: P1
- Title: Paper-readiness reporting is historical, not operational
- Affected file: `cmd/trader/codex_api.go`
- Source reference: lines 1164-1175 in the reviewed branch

Issue:

The paper-readiness summary is derived from stored gate state and does not include live runtime probes for current strategy load, broker quote usability, or current trader health.

Why it matters:

A sign-off artifact that can remain green while the actual runtime is unhealthy is not a trustworthy release gate.

Live evidence:

- `reports/paper-readiness/latest.md` still reported `ready: true`
- That artifact was generated on `2026-03-19`
- At the same time, the live trader engine on `8100` was returning `503`

## Additional Runtime Note

The current trader deployment health contract is also misleading:

- Docker and compose use `http://localhost:8081/health`
- That endpoint currently reports healthy even when the actual trader runtime at `8100` is unhealthy

This makes the deployment surface look better than the real execution surface.

## Recommended Status

- Paper trading release status: no-go
- Do not promote this branch for paper-trading sign-off until the findings above are fixed and re-verified live
