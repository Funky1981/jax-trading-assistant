# Jax Current Status

PAPER-02A3: **ACTIVATED**. PAPER-02-2026-01 is **ACTIVE / EXPLORATORY_PAPER**
under the frozen `paper-02-protocol-v1` contract. The pilot target is 50
genuine prospective opportunities; the sample is `0 / 50` at the HARNESS-00
handover. PAPER-02 remains exploratory and cannot establish demonstrated edge.

HARNESS-00: **IMPLEMENTED / EXTERNAL REVIEW REQUIRED / DOCUMENTATION ONLY**.
The harness runtime is not implemented and has not entered PAPER-02.

This is the concise operational status. Roadmap sequencing is authoritative in
`Docs/ROADMAP.md`; capability maturity is in `Docs/CAPABILITY_MATRIX.md`.

## Current branch context

- Branch: `capability-reset`
- Current code SHA: `521a4917c5e6be639ed8571cf383f22d99f1d077`
- PAPER-02 pilot: `PAPER-02-2026-01`
- Current build routing: `Docs/BUILD/CURRENT_PACKAGE.md`

## Runtime architecture

Jax is an ADR-0012 modular monolith with `cmd/trader` for deterministic
runtime/API paths, `cmd/research` for bounded research/orchestration paths, the
frontend, Postgres-backed state, and an explicit `services/ib-bridge` boundary.

## Accepted foundation

Phases 00–12 and CR-02G are accepted technical capability foundations at their
documented gates. They do not prove a trading edge or authorize live execution.

## Current scientific and package state

- VAL-03R4B: `COMPLETE / FAILED_VALIDATION / EXTERNALLY_REVIEWED`.
- `ma_crossover_v1`: `CLOSED / FAILED HISTORICAL VALIDATION`.
- Formal forward paper: `NOT STARTED`; historical formal result is NO-GO.
- VAL-04A: `COMPLETE / GO`.
- VAL-04B: `DEFERRED / NOT CURRENT NEXT STEP`.
- PAPER-01: `GO / EXTERNALLY REVIEWED`.
- PAPER-01A: `COMPLETE / SUPERSEDED BY PAPER-01B`.
- PAPER-01B: `IMPLEMENTED / ACCEPTED DIRECTIONALLY`.
- PAPER-01C: `ACCEPTED / EXTERNALLY REVIEWED`.
- PAPER-02A2: `READINESS VERIFIED`.
- PAPER-02A3: `ACTIVATED`.
- PAPER-02: `ACTIVE / EXPLORATORY_PAPER / FROZEN`.
- Harness architecture: `HARNESS-00 IMPLEMENTED / EXTERNAL REVIEW REQUIRED`.
- Harness runtime: `NOT IMPLEMENTED`.
- Trading edge: `NOT DEMONSTRATED`.
- Phase 13: `NOT STARTED / BLOCKED`.
- Optional future commercialisation: `DEFERRED / NOT A CURRENT OBJECTIVE`.

## Current trader identity

Event-driven short-horizon swing trader; event/news/source-backed catalyst;
technicals supporting only; human-approved exploratory entries; 1–5 trading
sessions, typical 2–3, hard maximum 5.

## Safety state

Live trading, broker execution, autonomous execution, and real order mutation
are disabled. Exploratory paper uses `PAPER`, `ExecutionAuthority=NONE`, and
maximum leverage 1x. No genuine PAPER-02 opportunity, order, fill, position, or
formal evidence row was created by activation. Five pre-existing
`pilot-restart-*` integration fixtures remain excluded from the production
sample.

## Next authorized work

Collect genuine prospective observations only. Admit an opportunity only when
its immutable first-seen timestamp is at or after the activation timestamp
recorded in the durable PAPER-02 handover. Retain all eligible outcomes,
including WATCH, NO_TRADE, unresolved, rejected, missing-data, and approved
cases. Do not tune policies, alter the frozen pilot identity, replay historical
events as prospective observations, start FORMAL_FORWARD_PAPER, implement
HARNESS-01, implement Context Engineering, or begin Phase 13.
