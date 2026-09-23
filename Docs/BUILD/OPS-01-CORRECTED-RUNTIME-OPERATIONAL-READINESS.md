# OPS-01 — Corrected Runtime Operational Readiness & End-to-End Proof

Status: **IMPLEMENTED / VALIDATED LOCALLY / PAPER-ONLY**

PAPER-02R reviewed SHA: `c188721af5d3fe466c7999bf3ce01859f125537a`.

OPS-01 validates that the corrected Jax runtime can operate its existing
World Monitor, candidate, approval, exploratory PAPER, persistence, restart,
and reconciliation seams as one bounded lifecycle. It does not claim a
strategy edge, continue the frozen PAPER-02 incident, activate another pilot,
authorize broker execution, or authorize live trading.

## Operational boundaries

- Runtime mode must be explicit `PAPER`.
- Execution authority is `NONE`.
- Broker execution and live trading remain disabled.
- Maximum leverage remains bounded at 1x.
- Required PostgreSQL tests use only `jax_paper02r_test` or
  `jax_paper02r_test_<suffix>` databases.
- The frozen PAPER-02 protocol and incident record remain unchanged.

## Readiness projection

`GET /api/v1/ops-01/readiness` is protected by the existing JWT API wrapper
and reports, without mutating state:

- process and database connectivity;
- PAPER mode, execution authority, broker flag, and leverage guardrails;
- World Monitor enabled/configuration state, canonical endpoint identity,
  durable cursor, last attempt/success/failure, counts, failure class, and
  consecutive failures;
- entry and review worker running/last-attempt/last-success/last-failure state;
- durable pending exit recommendations;
- a fail-closed `READY` / `NOT_READY` result.

World Monitor diagnostics distinguish disabled, invalid configuration,
upstream failure, decision failure, persistence failure, no-events, and
processed states. Diagnostic metadata is attached to the existing durable
cursor record; the frozen incident tables are not rewritten.

## End-to-end proof

The required disposable PostgreSQL proof is
`TestOPS01OperationalReadinessProof` in
`cmd/trader/ops01_operational_integration_test.go`. It exercises the actual
production seams with an isolated provider fixture and persisted substitute
market observations:

1. pull a retained provider page through the World Monitor pull worker;
2. persist raw provider bytes, normalized inbox state, and the genuine-event
   decision;
3. promote the event through the existing strategy/candidate/evidence/risk
   path;
4. preserve the immutable initial event decision and persist a versioned
   candidate-linked promotion decision after the candidate gates are ready;
5. create and replay an explicit canonical entry approval;
6. queue and consume the approved entry through the real exploratory runtime;
7. persist the simulated entry order, fill, ledger event, lifecycle, and review
   schedule with stable identities;
8. restart through the production runtime constructor, restore the durable
   paper venue, and create a durable `EXIT_RECOMMENDED` review;
9. submit an authenticated exit approval through the protected API boundary;
10. restart again, resume the approved review, simulate the exit fill, close the
    lifecycle, and reconcile the durable economic artifacts.

The proof verifies idempotent approved-entry replay, durable candidate and
workflow identities, exactly one entry and one exit order/fill/ledger event,
durable exit recommendation-before-approval ordering, and the closed outcome.

## Persistence and recovery corrections

- The initial genuine-event decision remains immutable. Candidate promotion is
  represented as a new current decision version with the candidate identity.
- Canonical evidence projections retain deterministic provenance references for
  normalized World Monitor records and market-candle observations.
- Runtime entry and exit persistence records the venue's post-fill order state,
  not the pre-fill `NEW` artifact.
- PostgreSQL reloads paper orders and fills into the paper venue at runtime
  construction, and reloads ledger events into a validated paper account.
- Database timestamp values are normalized to UTC at the runtime boundary.

## Required validation

The required CI integration job must set
`PAPER02R_REQUIRED_INTEGRATION=true`, provision an isolated disposable
PostgreSQL database, apply migrations, run the focused OPS-01 proof, and run
the Linux race proof. Repository validation also includes `go test ./...`,
focused tests, `golangci-lint`, `gofmt`, `git diff --check`, Golden Tests, and
Import Boundary Enforcement.

Passing OPS-01 proves bounded platform operation and durable lifecycle
correctness. It does not establish profitability, predictive power, or a
trading edge.
