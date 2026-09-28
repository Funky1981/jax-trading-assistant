# CORE-READINESS-02A3 — Canonical Candidate Economic Inputs

## Scope and disposition

CORE-READINESS-02A3 supplies the missing typed, durable request-level economic
inputs required before a future candidate-to-PAPER handoff. It is limited to the
economic-input prerequisite and recovery of the incomplete public queue route.
It does not establish Working Jax readiness and does not implement 02A4.

CORE-READINESS-02A was blocked because World Monitor candidates did not persist
trustworthy issuer identity or an explicit sizing-request policy. The previous
promotion path treated `RiskReviewConfig{}` defaults as production evidence and
persisted a hard-coded $100 risk budget. Neither is a valid economic fact.

## Contract and persistence

`internal/modules/candidates.CandidateEconomicInput` contains candidate and
contract identity, explicit instrument and issuer IDs with identity source and
policy version, risk allocation, requested leverage, sizing policy ID/version,
creation time, and a deterministic SHA-256 content identity. It records the
candidate's requested exposure, not account equity or a final order quantity.

Migration `000078_candidate_economic_inputs` adds an append-only typed table
with positive risk allocation and requested leverage capped at 1.0, a
restrictive candidate foreign key, unique content identity, and UPDATE/DELETE
protection. It does not backfill historical candidates.

World Monitor requires the explicit `JAX_CANDIDATE_ECONOMIC_POLICY_FILE`
configuration. Each supported symbol must explicitly map to both an
`instrument_id` and `issuer_id`, and the configuration must name identity and
sizing policy versions and provide risk allocation and requested leverage.
Missing policy, missing mappings, or incomplete sizing inputs fail closed.
Ticker, headline, event attributes, and model knowledge are never used to guess
an issuer. Fixture IDs are permitted only in disposable tests.

Identical candidate/request replays are idempotent; a conflicting second
request is rejected. Existing candidates without an artifact remain unchanged
and receive no guessed backfill. Without canonical economic input the candidate
remains `risk_pending` / `risk_not_ready`; with an explicit input, 02A3 still
stops before current-account portfolio risk. No candidate approval, PaperIntent,
paper order, or execution instruction is created by this package.

Until CORE-READINESS-02A4 implements the canonical current-account
portfolio-risk and approved-PAPER handoff, the legacy approval service rejects
every `world-monitor` `APPROVE` action, regardless of historical candidate
risk/gate fields or whether `CandidateEconomicInput` exists. The guard is in
`approvals.Service.Decide`, so protected HTTP and mobile-token approval callers
share it. `CandidateEconomicInput` is a request-level exposure contract, not
portfolio approval. World Monitor rejection, snooze, and reanalysis actions
remain available.

## Risk removal and public route boundary

The World Monitor canonical path no longer uses `RiskReviewConfig{}` or
records `proof risk-model assumption` / `existing RiskReviewConfig defaults` as
provenance. The `$100` `worldMonitorSuggestedSizing` function and its metadata
are removed. Final quantity remains dependent on the current PAPER account and
future portfolio-risk calculation; 02A3 does not implement that calculation.

The incomplete public candidate-ID entry-queue route is not registered. The
caller-supplied economic-payload route is not restored, and no stub handoff is
introduced. Internal `exploratorypaper.QueueApprovedEntry` remains available
to its existing internal integration callers. A routing test proves the public
route returns 404 until the complete 02A4 handoff exists.

## Preserved CORE-READINESS-02A work

This package preserves the existing JWT-bound candidate actor and auth tests,
quote-provider and receipt provenance, market observation safety contract,
bounded quote/chart/review queries, no-future observation rules, and review
error propagation. Direct JWT tests continue to prove that spoofed
`X-User-ID` cannot replace the authenticated actor and that missing claims do
not persist decisions.

## Validation record

Validation results, disposable PostgreSQL proof, race results, and exact-final-
SHA workflow evidence are recorded in the commit handover. The normal Jax
database is not used or mutated. No runtime deployment, strategy selection,
pilot, HARNESS-04, broker execution, or live trading is part of 02A3.

## Remaining package

CORE-READINESS-02A4 remains the separate canonical candidate → current PAPER
portfolio risk → durable workflow → authenticated approval → PaperIntent →
exploratory queue handoff. It is not implemented or authorized by this record.
