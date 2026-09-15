# VAL-03R4A — Pre-Start Data-Integrity Validator Closure

## Status

`VAL-03R4A = COMPLETE / EXTERNAL REVIEW REQUIRED`.

The single prior VAL-03R4 attempt was a pre-start abort. It did not create the
formal recovery run marker, and no strategy outcome was exposed. The old R4
authorization remains historical and is not reusable. R4B authorization is
absent; this package does not authorize recovery execution.

## Starting state and preserved boundaries

The package began from clean, synchronized `capability-reset` at
`fd74fd7fe570eab9dd4c5435671b00af68936f27`, after the attempt-preservation
commit. The selected candidate, recovery boundary, parameters, costs, sample
floors, 2025 seal and safety policy were unchanged.

`VAL-03` remains `CONTAMINATED_FOR_FORMAL_OOS / CLOSED`; the original
2023–2024 run is not rerun. Recovery performance remains unstarted.

## Attempt #1 forensic finding

The first `--execute` invocation aborted before `STARTED_ONCE` with:

`raw/split factor inconsistency XLK 2016-02-11`.

The exact XLK row is present in the previously accepted local payloads. The
canonical `marketdata.DeriveVAL03BSplitFactor` accepts the rounded OHLC frame
with factor `0.5` and tolerance `5e-4`; the old duplicate absolute-price
validator rejected it. This is validator drift, not a dataset defect.

The permanent forensic and preflight artifacts contain the payload identities,
row values, ratios, complete instrument matrix, synchronization result and
structural-only flags. The preflight covered 2,688 sessions per instrument,
including the recovery range, without calculating signals, episodes or returns.

## Corrective closure

The formal loader now uses the canonical shared factor and boundary helpers:

- `marketdata.DeriveVAL03BSplitFactor`
- `marketdata.IsVAL03BStructuralFactorBoundary`

The known rounded XLK regression is covered by a deterministic test. The R4A
freeze binds the corrected runner source blobs and requires the two prior
attempt-absence flags to be true. Contract audit and preflight-only modes pass.

The corrected `--execute` path is locked behind the future
`VAL-03R4B-EXECUTION-AUTHORIZATION.json`; that file was not created.

## Outcome and safety state

Formal recovery run count is `0`; `STARTED_ONCE` is absent; result artifacts and
formal run state are absent. No recovery signals, episodes, returns, P&L,
2025/2026 outcomes, orders, fills, broker calls, forward paper or Phase 13
activity occurred.

Execution remains `ExecutionAuthority=NONE`, `CreatesFill=false`, live trading
disabled, broker execution disabled and maximum leverage `1x`.
