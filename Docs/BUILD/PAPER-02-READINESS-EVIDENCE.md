# PAPER-02 Readiness Evidence

Status: **READY_FOR_EXTERNAL_REVIEW / NOT ACTIVE**.

This document records the design/readiness implementation only. It does not
activate PAPER-02, admit a prospective observation, establish an edge, or
authorize formal or live execution.

## Protocol

`Docs/PAPER_TRADING/PAPER-02-PROSPECTIVE-PILOT-PROTOCOL.md` defines the frozen
pre-activation protocol: 50 eligible event opportunities, maximum 90 calendar
days, explicit event/evidence/candidate/risk/human gates, five-session paper
lifecycle, missing-data semantics, incidents, metrics, calibration, and
falsification review.

## Pilot identity

`internal/modules/exploratorypaper/pilot.go` defines the versioned
`PilotProtocol`, `PilotIdentity`, content hash, policy/model/cost/calendar/code
versions, and the `DRAFT → READY_FOR_EXTERNAL_REVIEW → ACTIVE` state machine.
`ActivateWithExternalAuthorization` requires an explicit external reviewer
acknowledgement; this package never invokes it.

## Opportunity ledger

Migration `000073_paper02_pilot_readiness.up.sql` adds a durable pilot identity,
opportunity projection, append-only opportunity events, evidence records,
market-observation records, incidents, formal-firewall constraints, and
identity/append-only database triggers. The opportunity projection is created
at the prospective intake boundary before later decision or outcome.

Non-trades are retained by classification: `WATCH`, `NO_TRADE`, `UNRESOLVED`,
`REJECTED_EVIDENCE`, `REJECTED_RISK`, `REJECTED_HUMAN`, and
`APPROVED_EXPLORATORY_TRADE`.

## Evidence, latency, and market quality

Evidence records preserve first-seen, publication when available, ingestion,
source, relevance, signal, phase, and policy version. Entry replay is
idempotent; post-entry supporting, contradictory, invalidating, irrelevant,
and missing evidence are distinct phases. Unknown publication latency remains
unknown. Market observations require source, timestamps, bid/ask, spread,
price, cost model, and explicit stale/missing treatment.

## Risk, human decisions, and outcomes

The pilot layer links approved opportunities to the existing PAPER-01
lifecycle. It does not duplicate the paper venue or ledger. Human decisions
remain explicit and risk vetoes cannot be overridden. Descriptive metrics
include non-trades and outcomes but hard-code `formalEvidenceEligible=false`
and `demonstratedEdge=false`.

## Operator surface

Protected read-only routes expose the pilot identity, protocol hash/status,
opportunity ledger, incidents, metrics, active exploratory positions, and
`NOT FORMAL EVIDENCE`. No activation route is registered.

## Restart and reconciliation

The Postgres integration test persists a test-only active fixture, records an
opportunity, candidate decision, PAPER-01 lifecycle link, entry evidence, and
post-entry invalidating evidence, then reconnects through a new Postgres pool.
It verifies identity, opportunity, lifecycle link, new-evidence count, and
evidence idempotency survive restart. The fixture is isolated and cleaned up;
no real pilot is activated.

## Remaining limitation

The control plane is ready for external review but remains inactive. A real
pilot still requires external activation, live operational monitoring, and a
separately reviewed decision about the eventual FORMAL-01 design. No
prospective sample has been collected by this package.
