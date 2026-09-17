# PAPER-01B Implementation Audit

Status: **IMPLEMENTED / EXTERNAL RE-REVIEW REQUIRED**. This document is an
implementation audit, not authorization for PAPER-02, FORMAL_FORWARD_PAPER,
Phase 13, or live/broker execution.

## Reused repository facilities

- Event/evidence identity and reviewed candidate gates remain owned by
  `internal/modules/eventdecisions` and the canonical `candidate_evidence_scores`
  projection. `exploratorypaper.GenerateCandidate` accepts only a reviewed
  assessment from that provider and binds its evidence fingerprint and policy
  version.
- Human approval remains the existing `internal/modules/workflow` state
  machine. `workflow_instances` and `workflow_audit_events` are reused; no
  second approval system was introduced.
- Simulated execution remains `internal/modules/papertrading`. Existing
  `paper_orders`, `paper_fills`, `paper_accounts`, and `paper_ledger_events`
  are reused; no second ledger or broker path was introduced.
- Postgres migration ordering and append-only conventions are reused in
  `db/postgres/migrations/000069_exploratory_paper_lifecycle.*`.
- Existing `cmd/trader` route protection and scheduler startup are reused for
  the minimum exploratory read model and due-review worker.

## Gaps closed

1. `exploratory_paper_lifecycles` durably binds event/candidate/thesis,
   frozen hashes, approval IDs, paper order/fill IDs, position state, policy
   versions, and outcome references.
2. Reassessments, reviews, checkpoints, and outcomes have stable identities,
   uniqueness constraints, append-only triggers, and explicit missing-data
   states.
3. Restart reads rehydrate and validate the frozen thesis, binding, workflow,
   paper intent, paper order, paper fill, reviews, checkpoints, and outcome.
   Missing or divergent required rows fail closed.
4. Candidate quality/corroboration is derived from reviewed evidence facts;
   unknown, stale, contradictory, irrelevant, insufficient, or ambiguous
   evidence remains WATCH rather than becoming CANDIDATE.
5. Thesis and evidence content hashes detect tampering. Repeated evidence is
   idempotent; conflicting reuse of an evidence identity is rejected.
6. Exit precedence, explicit session windows/calendar coverage, deterministic
   fill-based accounting, direction-aware MFE/MAE, and checkpoint arithmetic
   are encoded and tested.
7. `GET /api/v1/exploratory-paper/positions` and the position detail route
   expose the reviewable read model with `NOT FORMAL EVIDENCE`.

## Known boundary

The scheduled worker records a visible `MISSING_DATA` review when required
market/evidence input is unavailable; it never fabricates a review or starts a
pilot. PAPER-01B remains exploratory control-plane and simulated-paper
capability only. PAPER-02 sequencing versus FORMAL-01 remains for an external
decision after re-review.
