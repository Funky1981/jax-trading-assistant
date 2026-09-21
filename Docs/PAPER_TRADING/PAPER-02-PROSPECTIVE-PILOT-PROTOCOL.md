# PAPER-02 Prospective Exploratory Paper Pilot Protocol

Protocol version: `paper-02-protocol-v1`
Status: `READY_FOR_EXTERNAL_REVIEW / NOT ACTIVE`
Mode: `EXPLORATORY_PAPER`

## Purpose and status

PAPER-02 is a bounded prospective operational and product-learning pilot for
Jax's event-driven short-horizon swing trader. It tests whether the complete
event, evidence, approval, simulated execution, review, and outcome loop can
operate prospectively and honestly.

PAPER-02 IS EXPLORATORY AND CANNOT ESTABLISH TRADING EDGE.

It is not formal validation, does not authorize live or broker execution, and
cannot populate or be relabelled as `FORMAL_FORWARD_PAPER`. Its P&L,
win/loss, MFE, MAE, confidence, and other outcome statistics are descriptive
observations only.

## Start and end conditions

The package is currently `READY_FOR_EXTERNAL_REVIEW`; no pilot is active and
no prospective observation has been admitted. Activation requires a separate
external decision that freezes this protocol, its hash, the eligible universe,
all policy versions, the code SHA, the opportunity target, and the maximum
duration.

After activation, the pilot starts at the externally recorded activation time.
An opportunity is eligible only when it is first seen at or after that time.
The pilot ends at the first of:

1. 50 eligible prospective event opportunities being recorded;
2. 90 calendar days after activation; or
3. a declared abort condition.

The target is an opportunity target, not a trade target. It cannot be changed
because later outcomes look favourable or unfavourable.

## Eligible universe and intake boundary

The eligible event universe is prospective, source-backed issuer or market
events with a plausible causal connection to a resolved instrument. Eligible
instruments must be resolved, supported by the configured paper universe, and
covered by the explicit trading-session calendar. Technical-only signals,
unresolved assets, retrospective events, duplicated source identities, and
events first seen before activation are not eligible.

At the event-intake boundary, Jax assigns an `OpportunityID` before its later
decision or outcome is known. The opportunity ledger retains every eligible
case, including `CANDIDATE`, `WATCH`, `NO_TRADE`, unresolved, evidence-rejected,
risk-rejected, human-rejected, and approved exploratory trades.

## Evidence and candidate gate

Evidence must be source-backed, timestamped, provenance-preserving, relevant to
the issuer and instrument, and assessed by the canonical reviewed evidence
projection. The existing candidate policy remains authoritative. At minimum,
candidate evidence must be sufficient, fresh, non-contradictory, corroborated
by independent sources, and associated with a versioned policy.

Unknown, stale, ambiguous, contradictory, or insufficient evidence remains
visible and cannot silently become a candidate. The default decision remains
`NO_TRADE`; `WATCH` is retained when a setup is interesting but not eligible.
Technicals are supporting context only and cannot be the sole catalyst.

## Human approval and risk

Every approved exploratory entry requires explicit human approval bound to the
candidate, frozen thesis, evidence snapshot, risk decision, direction, stop,
target, horizon, uncertainty, rationale, reviewer, timestamp, and an explicit
PAPER-only acknowledgement. Human rejection is valid pilot data.

The existing deterministic portfolio-risk policy is reused. Maximum leverage
is 1x. Existing configured limits govern risk per position, aggregate
exposure, concurrent positions, concentration, and correlated or duplicate
events. A human cannot override a deterministic risk veto. Entry approval does
not grant broker or live execution authority.

## Simulated execution and lifecycle

Entries, exits, fills, ledger mutations, positions, costs, spread, slippage,
and latency use the existing isolated simulated paper venue. The runtime must
remain `Environment=PAPER`, `ExecutionAuthority=NONE`, and
`BrokerExecutionAllowed=false`.

The exploratory lifecycle is 1–5 known trading sessions. The fifth-session
hard time limit is actionable at the opening of session five. Reviews use the
explicit supported calendar; unknown or incomplete calendar state fails closed.

Supported exit reasons are `STOP`, `TARGET`, `THESIS_INVALIDATED`,
`RISK_KILL`, `TIME_LIMIT`, and `MANUAL_OPERATOR`. A closed-market exit is
deferred to the next known tradable session. Exit approval is a distinct human
decision from entry approval.

## New-evidence monitoring

Each review distinguishes:

- `ENTRY_EVIDENCE`;
- `NEW_SUPPORTING_EVIDENCE`;
- `NEW_CONTRADICTORY_EVIDENCE`;
- `NEW_INVALIDATING_EVIDENCE`;
- `NO_NEW_RELEVANT_EVIDENCE`; and
- `MISSING_EVIDENCE`.

Entry evidence replay is idempotent and cannot be counted as new. New evidence
requires a first-seen timestamp after entry and retains evidence identity,
publication time when available, ingestion time, source provenance, relevance,
signal, relationship to entry, and policy version. Conflicting reuse of an
evidence identity fails closed.

## Latency and market-data capture

The pilot records, without fabricating unknown timestamps:

`publication → ingestion → canonical assessment → candidate decision → human
review → simulated entry`.

For open positions it also records new-evidence publication/first-seen time,
ingestion, thesis reassessment, operator visibility, and exit recommendation
where known. Missing publication time produces an explicit unknown latency.

Each simulated entry, review, exit, and checkpoint records market-data source,
observation time, received time when available, quote/candle age, bid/ask
availability, spread, price used, cost-model version, and stale/missing flags.
Stale or unavailable data fails closed or becomes explicit `MISSING_DATA`.

## Metrics and calibration

Metrics are frozen before activation and include eligible opportunities,
resolution, unresolved, evidence readiness, WATCH, NO_TRADE, candidate, risk
rejection, human rejection, exploratory entry, missing data, review completion,
evidence/decision/entry/review/exit-approval latency, duplicate and
reconciliation incidents, fail-closed incidents, and descriptive outcome
fields.

Outcome metrics may describe gross/net P&L, net return, costs, direction,
event category, holding sessions, exit reasons, MFE, MAE, and mechanism versus
expectation. No PAPER-02 metric is a profitability pass/fail gate. No metric
authorizes live trading or establishes demonstrated edge.

Before an outcome is known, the record preserves confidence, uncertainty,
expected mechanism, expected horizon, expected direction, and invalidation
conditions. Confidence is not treated as a validated probability.

## Falsification and post-pilot review

Each approved thesis preserves what would make it wrong, counter-evidence,
contradictions, invalidation evidence, and whether the expected mechanism
occurred. The post-pilot review must ask what evidence suggests apparent
success was luck, selection, market drift, leakage, hindsight, or operational
bias. Outcomes cannot be used to tune the active pilot policy.

## Incidents, pause, and abort

The pilot pauses new admissions and preserves all existing records when an
incident indicates accidental broker/live routing, scientific persistence
failure, identity or reconciliation conflict, material provenance failure,
systematic stale data, repeated scheduler failure, lifecycle restore failure,
formal-firewall violation, calendar coverage failure, or accounting
inconsistency.

An abort records the incident, status, timestamp, details, and admission
boundary. Existing positions remain reviewable. Resumption requires explicit
external review; history is never rewritten.

## Operator responsibilities and audit fields

Operators review the pilot status, opportunity ledger, evidence provenance,
approval records, active positions, scheduled reviews, incidents, checkpoints,
exit approvals, and descriptive outcomes. Required fields include all stable
identities, source and market provenance, timestamps, policy/model/code
versions, missing-data reasons, decisions, human actors, rationale, and
exploratory/formal labels.

The protected read model exposes `EXPLORATORY_PAPER` and `NOT FORMAL
EVIDENCE`. It does not expose an activation control.

## Version and freeze rule

The state machine is:

`DRAFT → READY_FOR_EXTERNAL_REVIEW → ACTIVE`.

This implementation creates the DRAFT and READY control plane only. Activation
requires explicit external authorization. Once active, protocol, universe,
target, duration, metrics, risk boundaries, versions, and code SHA are frozen.
Any later change requires a new version with an explicit reason and only has
prospective effect. Existing opportunities cannot be rewritten.

Historical incident cross-reference: the preserved PAPER-02-2026-01 incident
and the corrective PAPER-02R readiness boundary are recorded in
`Docs/PAPER_TRADING/PAPER-02-2026-01-INCIDENT-REVIEW.md` and
`Docs/BUILD/PAPER-02R-INCIDENT-PRESERVATION-RUNTIME-READINESS.md`. This
cross-reference does not alter the frozen protocol, pilot identity, sample, or
activation semantics.
