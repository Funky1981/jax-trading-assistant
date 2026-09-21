# PAPER-02R — Incident Preservation & Corrected Paper-Runtime Readiness

Status: **IMPLEMENTED / VALIDATION REQUIRED / PAPER-02 HISTORICALLY
PRESERVED**

PAPER-02R is a corrective engineering package. It does not activate another
pilot, continue PAPER-02-2026-01, start formal forward paper, authorize broker
execution, or authorize live trading.

## Scope

PAPER-02R:

- preserves the frozen PAPER-02 identity and incident record;
- makes exit recommendation persistence precede any approval attempt;
- persists human exit approval or rejection with the workflow/paper-intent
  identity;
- keeps the existing workflow approval system authoritative;
- accounts for executed fill prices once and deducts commission once, while
  retaining spread/slippage as attribution;
- reconciles the paper ledger after simulated entry and exit;
- retains durable checkpoint accounting and accounting-version metadata;
- exposes worker errors, retryability, consecutive failures, and last success;
- labels candle-close and modeled-liquidity observations honestly;
- marks MFE/MAE incomplete unless coverage, cadence, provenance, and the full
  observation window are established;
- preserves World Monitor intake health and idempotent cursor/raw-payload
  provenance;
- uses isolated disposable PostgreSQL guardrails and required CI integration;
- fixes the HARNESS same-key PostgreSQL checkpoint/compaction race narrowly;
- records final durable identity/uniqueness in concurrent replay tests;
- applies only narrow HARNESS as-of temporal corrections; HARNESS-04 is not
  started.

## Historical preservation boundary

The frozen protocol remains
`Docs/PAPER_TRADING/PAPER-02-PROSPECTIVE-PILOT-PROTOCOL.md`. The incident
record is `Docs/PAPER_TRADING/PAPER-02-2026-01-INCIDENT-REVIEW.md`. The frozen
runtime SHA and historical sample remain unchanged. PAPER-02R does not rewrite
historical rows, promote restart fixtures, or convert exploratory records into
formal evidence.

## Runtime correctness contract

An exit decision first creates a durable `EXIT_RECOMMENDED` review carrying a
stable recommendation identity. No fill may be simulated while the review has
no durable human approval. A human rejection is durable and produces no exit
fill. A human approval must validate as the existing workflow's
`StatePaperIntentCreated` state and carries the approved paper intent into the
existing paper venue. The production PAPER worker constructs this runtime with
the PostgreSQL store as its `ExitApprovalSource`; it does not use an in-memory
or fixture approver. The protected operator handoff is the existing API
surface `POST /api/v1/exploratory-paper/positions/{positionId}/exit-approval`
or `.../exit-rejection`. It accepts the complete existing workflow snapshot
and an `ExitApprovalBinding` containing the position, review, recommendation,
entry workflow, and entry paper-intent identities. PostgreSQL reloads the
canonical review, validates the workflow audit stream and confirmation, and
persists the approval or rejection idempotently. The worker can therefore
resume an approved recommendation after restart without treating a
recommendation itself as approval. Closed outcomes transition the approved
review and lifecycle atomically.

Economic P&L is:

`gross executed-price P&L - entry commission - exit commission`.

Modeled spread and slippage are already embedded in executed fill prices and
are retained as attribution, not deducted a second time. The ledger cash delta
is independently reconciled with the outcome net P&L. Accounting and cost-model
identities remain explicit.

Review observations carry quote mode, liquidity mode, actual-quote availability,
observed/received timestamps, and excursion coverage. Candle-close observations
are not represented as actual bid/ask or complete path coverage. Unknown or
incomplete data remains visible and fails closed where a required input is
missing.

## PostgreSQL and CI boundary

The required CI integration job provisions a PostgreSQL 16 service database
named `jax_paper02r_test`. `PAPER02R_REQUIRED_INTEGRATION=true` is set, all
PAPER-02R database variables point to that disposable database, migrations are
applied before tests, and the focused integration and race packages run on
every push/pull request. `internal/testsupport` rejects normal Jax, `postgres`,
and other non-`jax_paper02r_test*` databases. Missing required configuration is
a failure rather than a skip.

## HARNESS boundary

HARNESS-03 remains an externally reviewed capability and is not integrated
with JaxMind or PAPER-02. PAPER-02R only fixes the PostgreSQL same-key race in
checkpoint/compaction persistence and adds narrow temporal tests: future memory
is omitted and audited, and a context build cannot carry task state created or
updated after its explicit reference time. HARNESS-04 is not started.

## Validation expectations

Validation must include repository unit tests, affected package tests, race
tests for replay/lifecycle packages, migration checks, and required PostgreSQL
CI. Passing tests establish implementation correctness only; they do not
establish a trading edge or authorize a new pilot.
