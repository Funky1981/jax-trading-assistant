# Paper Trading Review Fix Plan

- Date: 2026-04-17
- Scope: remediation plan for the paper-trading review findings recorded in `paper_trading_review_20260417_findings.md`
- Goal: make paper-trading readiness operationally truthful, not just historically green

## Remediation Order

### 1. Fail trader startup when no approved strategies are loaded

Target:

- `cmd/trader/main.go`

Required change:

- After `LoadApprovedStrategies`, fail startup in `paper` and `live` when `registry.List()` is empty
- Keep the failure explicit in logs so operators can distinguish "no approved artifacts" from generic startup failure

Acceptance check:

- In `JAX_RUNTIME_MODE=paper`, trader must exit non-zero if zero approved strategies are loaded
- Container must not become healthy in that condition

### 2. Add a real trader readiness endpoint and move deployment health to it

Targets:

- `cmd/trader/main.go`
- `cmd/trader/frontend_api.go`
- `cmd/trader/Dockerfile`
- `docker-compose.yml`
- `start.ps1`

Required change:

- Add a trader `/ready` endpoint that checks at least:
  - strategy registry is non-empty
  - signal-generator health passes
  - broker dependency is usable when execution is enabled in paper mode
- Stop using frontend API `/health` as the container/runtime health gate for trader readiness

Acceptance check:

- Docker health, compose health, and startup script must all agree on the same readiness endpoint
- A trader with zero strategies or unusable broker quotes must not report ready

### 3. Make IB bridge readiness reflect quote usability, not just socket connectivity

Targets:

- `services/ib-bridge/ib_client.py`
- `services/ib-bridge/main.py`

Required change:

- Harden quote retrieval to wait for usable market data instead of returning `200` with all-zero values
- Consider fallback price fields IB may populate when `last` is unavailable
- Make readiness fail when the bridge is connected but cannot produce usable quotes for a configured verification symbol

Acceptance check:

- `/ready` must return non-`200` when quote data is connected but unusable
- `/quotes/<symbol>` must not return a success response with `price=0.0`, `bid=0.0`, and `ask=0.0` unless the API explicitly marks that state as unusable
- Trader market ingester must stop logging zero-price quote failures in the normal paper stack

### 4. Enforce strict provider policy in paper mode

Target:

- `cmd/trader/main.go`

Required change:

- Apply event/news provider hard-fail rules to `paper` mode, not only `live` and production-like envs
- Keep the error message explicit about which provider requirement is missing

Acceptance check:

- A paper-mode runtime with enabled event-dependent strategies and no required provider credentials must fail startup

### 5. Make paper-readiness reporting operational

Targets:

- `cmd/trader/codex_api.go`
- `reports/paper-readiness/latest.md`
- `reports/paper-readiness/latest.json`

Required change:

- Extend paper-readiness generation to include live runtime probes for:
  - trader readiness
  - strategy availability
  - IB bridge quote usability
  - research readiness where required by the runtime path
- Refuse to publish a `ready` summary when live runtime checks fail
- Include `checkedAt` and live probe results directly in the output artifact

Acceptance check:

- A stale green artifact must not survive a red runtime
- If live trader readiness is failing, the generated readiness summary must be `not_ready`

## Verification Plan

After the fixes are implemented, re-run this sequence against the actual running stack:

1. Rebuild and recreate the paper stack from the current branch
2. Verify trader startup fails when no approved strategies exist
3. Verify trader `/ready` is the health gate used by Docker, compose, and scripts
4. Verify `ib-bridge /ready` fails when quotes are zero or unusable
5. Verify `ib-bridge /quotes/<symbol>` returns usable prices in the paper stack
6. Verify paper-mode provider validation fails fast for missing required event/news inputs
7. Re-generate the paper-readiness report and confirm it matches live runtime truth

## Exit Criteria

The paper-trading blocker set is closed only when all of the following are true:

- Trader cannot start ready with zero approved strategies
- Trader deployment health matches real trader readiness
- IB bridge readiness proves quote usability, not just connectivity
- Paper mode enforces strict provider policy
- Paper-readiness artifacts are generated from live operational truth
- A fresh live verification run produces a `go` outcome for paper trading
