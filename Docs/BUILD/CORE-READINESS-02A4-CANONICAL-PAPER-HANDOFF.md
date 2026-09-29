# CORE-READINESS-02A4 — Canonical PAPER Handoff

Status: **IMPLEMENTED / EXTERNAL REVIEW REQUIRED**. This package closes the
server-owned candidate-to-queue handoff gap and ends at
`exploratory_paper_entry_queue`. It does not consume the queue, deploy the
runtime, create a pilot, select a strategy, or establish a trading edge.
Working Jax remains `BLOCKED_BEFORE_HYPOTHESIS_DESIGN` pending the isolated
genuine-market proof in CORE-READINESS-02B and external review.

## Canonical input and portfolio state

The authenticated handler accepts only a candidate identity and an empty
request body. The service loads the World Monitor candidate, current genuine
event decision, normalized intake record, scored source-backed evidence,
append-only `CandidateEconomicInput`, and policy identities from PostgreSQL.
It rejects expiry, consumed/conflicting execution identities, missing or
conflicting economic facts, unsupported short opening, and missing provenance.
`CandidateEconomicInput` carries explicit instrument/issuer identity, risk
allocation, requested leverage, sizing/identity policy versions, content
identity, and explicit slippage allowance. Migration 000079 stores the
allowance without rewriting legacy input identities; legacy rows without it
fail closed for this path.

The configured `PAPER_ACCOUNT_ID` must already exist and be a PAPER account.
The handoff reconstructs positions from that account's append-only ledger and
requires the ledger to reconcile. Each open position is valued using the
reviewed market-observation policy: mapped instrument, allowed provider,
non-future provider timestamp, valid receipt timestamp, known price, and
freshness. Equity is derived as current cash plus freshly marked positions;
stale average cost is used only as cost basis, never as current price.

A zero-position account is represented explicitly with `KnownEmpty=true`,
known account/currency/cash/equity, durable provenance, and no synthetic
position. Unit tests prove that this semantic produces zero gross/net/long/
short exposure and cash allocation equal to cash divided by equity; an
unmarked zero-position snapshot remains invalid.

## PortfolioSnapshot v1 identity compatibility correction (CORE-READINESS-02A4R)

`KnownEmpty` adds a new explicit semantic without changing the existing
`jax.portfolio.snapshot/v1` contract or its canonical identity algorithm.
`KnownEmpty=false` is omitted from canonical JSON, preserving the exact
pre-02A4R v1 representation and the content-addressed IDs of historical
non-empty snapshots. A historical JSON payload with no `known_empty` field
decodes as false and rebuilds to the same `SnapshotID`. `KnownEmpty=true` is
serialized explicitly and remains distinct after JSON round trip. Zero
positions with false, and non-empty positions with true, remain invalid.

Regression coverage compares a current non-empty snapshot against an
independently declared pre-02A4 `PortfolioSnapshot` wire struct and legacy
canonical SHA-256, reloads a legacy payload through `PostgresSnapshotStore.Get`
and `BuildSnapshot`, verifies the canonical handoff reload helper, and
round-trips a `RiskDecision` bound to the original legacy snapshot identity.
No historical database row, SnapshotID, risk decision, or workflow identity
was rewritten.

## Requested exposure and portfolio risk

Production requires `JAX_PORTFOLIO_RISK_POLICY_FILE`, a valid versioned
`portfoliorisk.RiskPolicy`, and positive `JAX_PORTFOLIO_STATE_MAX_AGE`. There
are no default risk parameters. Policy leverage is at most 1x, candidate
requested leverage must not exceed policy, and risk allocation must respect
the configured maximum when present.

Requested risk budget is `equity * CandidateEconomicInput.RiskAllocation`;
per-unit risk is `abs(entry - stop) + explicit slippage allowance`; raw
quantity is risk budget divided by per-unit risk; raw notional is raw quantity
times entry; leverage cap is equity times requested leverage; requested
notional is the lesser of raw notional and leverage cap. This is transparent
translation of supplied economic provenance, not a strategy threshold.

Exposure analytics and the deterministic `RiskDecision` are built against the
canonical snapshot and explicit policy. Snapshot and decision use their
existing append-only persistence contracts. A REJECT persists risk only and
creates no workflow. ACCEPT/AMEND persists one risk-bound workflow with
`RISK_ACCEPTED` or `RISK_AMENDED`, then
`AWAITING_HUMAN_CONFIRMATION`, before PREPARE responds. Identical PREPARE
replay returns the same snapshot, decision, and workflow identities. A changed
snapshot/risk binding cannot replace an existing pending human decision; an
explicit future supersession path is required.

## Human decision, workflow, and intent

The explicit routes are:

- `POST /api/v1/exploratory-paper/handoff/{candidateId}/prepare`
- `POST /api/v1/exploratory-paper/handoff/{candidateId}/approve`
- `POST /api/v1/exploratory-paper/handoff/{candidateId}/reject`

All require the existing validated JWT middleware. The actor comes from
validated request-context claims; `X-User-ID` is ignored. Caller-supplied
economic/portfolio/workflow fields are rejected. The service is available only
in PAPER mode and fails closed when policy/account/market configuration is
missing.

APPROVE/REJECT require the durable pending workflow and lock the candidate and
workflow in a serializable transaction. The confirmation binds authenticated
actor, candidate/recommendation, workflow, risk decision, instrument,
direction, resulting value, and time. The canonical approval method writes the
normal candidate approval/audit facts and approved status without invoking the
legacy approval service. REJECT persists the authenticated rejection and
`HUMAN_REJECTED`, with no intent or queue.

APPROVE advances through `HUMAN_APPROVED` and durable `PAPER_INTENT_CREATED`.
The intent is paper-only, has broker execution disabled, causes no portfolio
mutation, and remains `NOT_EXECUTED`. Workflow state, immutable audit events,
candidate decision, and queue insertion commit together. Workflow recovery is
reconstructed from PostgreSQL after service recreation; identical retry returns
the established queued result. Transaction retries handle serialization and
deadlock responses; deterministic identities, row/advisory locks, and durable
unique constraints protect replay/concurrency.

## Thesis, fresh market data, quantity, and queue boundary

The exploratory thesis is projected only from persisted normalized event,
candidate, reviewed evidence, chart confirmation, economic identity, and
versioned policy facts. Missing required durable narrative/provenance is a
hard failure; no model-generated narrative is supplied.

APPROVE reads a fresh quote again through the reviewed provider/symbol/age and
receipt-provenance policy. It rejects a future/stale/wrong-provider observation
or price divergence beyond the explicit slippage allowance; it does not reuse
the PREPARE quote. The server builds the market tick and `EntryRequest`. Final
quantity is bounded by absolute resulting risk value divided by the fresh
reference price, the explicit requested quantity ceiling, available account
cash, venue minimum/fractional constraints, and the <=1x PAPER cap; it is
floored rather than rounded upward.

The service uses `NewPostgresStoreForAccount(pool, PAPER_ACCOUNT_ID)` and
`QueueApprovedEntry` semantics. It does not invoke `Runtime.RunEntryCycle`.
Expected post-approval state is one candidate approval, one final workflow,
one semantic PaperIntent, and exactly one account-scoped queue row. There are
no execution instructions, candidate paper tickets, orders, fills, ledger
events, or exploratory lifecycles before a later separately authorized runtime
consumption proof.

## Legacy isolation and proof boundary

The legacy `approvals.Service.Decide(APPROVED)` guard
`ErrCanonicalPaperHandoffRequired` remains for World Monitor candidates. The
legacy `/api/v1/approvals/{candidateId}/approve` endpoint is not an alternate
execution route. Broker execution and live trading remain disabled and outside
scope.

Disposable PostgreSQL tests exercise ACCEPT, AMEND, risk REJECT, human REJECT,
known-empty account state, JWT actor binding/spoof rejection, queue-only
boundary, replay, restart recovery, concurrent approval, and cross-account
isolation. Required validation runs against a `jax_paper02r_test*` database;
no normal Jax database is used. CORE-READINESS-02B remains the isolated
genuine-market end-to-end proof and is not started by this package.
