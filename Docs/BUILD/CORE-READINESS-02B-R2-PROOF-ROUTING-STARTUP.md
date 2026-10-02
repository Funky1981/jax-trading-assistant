# CORE-READINESS-02B-R2 Proof-Mode Technical Routing Fixture Startup Correction

## Result

**Status: `IMPLEMENTED / TECHNICAL VALIDATION PASSED`**

R2 corrects proof-mode startup so the technical routing identity required by
the World Monitor promoter is present in the guarded disposable database while
the generic PAPER event-provider contract remains unchanged for ordinary
runtime. The fixture is installed only after the generic readiness check. On
restart, the proof-mode readiness query excludes only the reserved fixture
name; the post-gate fixture validator then checks its full identity and
semantics and fails closed on any incompatibility.

This package is routing infrastructure only. It does not consume a World
Monitor page, alter the promoter threshold or candidate logic, select a
strategy or hypothesis, establish an edge, or perform broker/live execution.

## Authority and preserved Attempt 2 result

- Starting authority: branch `capability-reset`, clean HEAD at
  `210be8c3b3264825a29c22f7b0c1daa88be7fd79`, equal to
  `origin/capability-reset` with zero divergence.
- Attempt 2 remains **valid at its reached zero-promoter-input boundary**:
  page `72795 → 76220`, `NO_TRADE=20`, `WATCH=5`, `CANDIDATE=0`, with zero
  promoter input rows at the unchanged `0.55` minimum. The later routing issue
  did not cause its zero-candidate result.
- The startup incompatibility was discovered after Attempt 2: the enabled
  technical routing fixture triggered generic PAPER event-provider readiness
  because `POLYGON_API_KEY` was empty. The fixture was removed and Attempt 2
  then started with Alpaca only.
- The correction is complete before any attempt from frozen cursor `76220`.
  R2 did not inspect or consume that continuation page. Attempt 3 was not
  started.

## Startup and isolation contract

Startup validates `CORE_READINESS_02B_PROOF_MODE=true`, the PAPER-only
configuration and Alpaca/World Monitor prerequisites, then connects to the
configured PostgreSQL database and verifies the dedicated PAPER account. It
loads normal strategy-instance config rows and runs generic event-provider
readiness before creating the fixture on first startup. Only afterward does it
ensure the routing fixture, then continue with approved artifact loading. The
fixture is not an approved artifact and the approved strategy registry remains
empty (`0`).

For restarts, an already-persisted row with the exact reserved fixture name is
excluded from the generic event-dependent-provider scan only while proof mode
is explicitly enabled. The fixture helper then revalidates it after that gate.
This proof-mode restart exception does not apply to ordinary PAPER startup;
ordinary enabled event-dependent strategies still require the normal Polygon
provider contract.

Before any fixture write, the helper verifies that both the configured URL and
the actual connection select `jax_paper02r_test` or `jax_paper02r_test_*`.
It serializes creation with a transaction advisory lock. An existing fixture
is reused only when its name, strategy type and ID, enabled state, session
timezone, flatten time, policy-derived symbols, four classification labels,
boolean flags and canonical config hash match. Duplicate semantic fixtures,
unexpected enabled rows, broader-than-policy symbol sets, and incompatible
existing fixtures fail closed. No unexpected row is disabled or changed.

The symbol set is derived from
`config/core-readiness-02b-candidate-economic-policy.json`: SPY, QQQ, DIA, IWM,
XLK, XLF, XLE, SMH, SOXX, TLT, and GLD. The fixture sets
`strategy_evidence=false`, `hypothesis_selection=false`, and `disposable=true`.
It provides promoter routing identity only; it does not represent strategy or
economic evidence.

## Proof and regression coverage

The guarded disposable-PostgreSQL proof checks first creation, second-startup
reuse with exactly one fixture, promoter routing for every policy ETF, and no
routing identity for non-policy symbol AAPL. It also rejects an unrelated
enabled strategy and a duplicate semantic fixture. Semantic validation tests
reject wrong strategy identity, missing labels, positive evidence/selection
flags, `disposable=false`, unexpected config fields, and a broader symbol set.
The helper refuses proof mode off, non-PostgreSQL targets, and normal database
names before mutation.

The readiness regression inserts an ordinary enabled event-dependent strategy
in a disposable database and verifies PAPER startup still fails with an empty
`POLYGON_API_KEY`. The corresponding proof-mode startup gate passes without
Polygon when the only enabled row is the guarded fixture; Alpaca and World
Monitor proof prerequisites remain independently validated. No generic
provider requirement was removed or relaxed for normal PAPER mode.

## Validation

- `go test ./cmd/trader`: passed, including required disposable PostgreSQL
  routing and normal-PAPER provider-readiness regression.
- `go test ./...`, `golangci-lint run ./...`,
  `scripts/golden-check.ps1 -Mode verify`, `go list -mod=readonly -m all`,
  `docker compose config --quiet`, scoped gofmt verification, and
  `git diff --check` passed.
- Windows local `go test -race ./cmd/trader` was unavailable because CGO is
  disabled and no compiler is installed; exact-SHA Linux CI race coverage is
  required for final closeout.
- The runtime process was not launched for this correction package: its genuine
  World Monitor worker can request a page at startup, while R2 explicitly
  preserves cursor `76220` and consumes no new page.

## Safety state and next step

- World Monitor continuation cursor: `76220`, unchanged by R2.
- Normal database mutation: none; normal Jax database/cursor not accessed.
- Attempt 3: not started.
- Candidates, approvals, queues, orders, fills, lifecycles, and outcomes:
  none created by R2.
- Broker calls: `0`; IB Bridge: not used; live trading: not authorized.
- Polygon requirement in proof mode: no, provided the guarded proof-mode
  configuration independently validates Alpaca and World Monitor prerequisites.
- Strategy or trading-edge claim: none.

The next package after external GO is **CORE-READINESS-02B Attempt 3**,
sequentially from cursor `76220`, during a valid regular US PAPER session.
