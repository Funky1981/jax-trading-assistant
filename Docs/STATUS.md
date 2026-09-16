# Jax Current Status

This is the concise operational status. Roadmap sequencing is authoritative in
`Docs/ROADMAP.md`; capability maturity is in `Docs/CAPABILITY_MATRIX.md`.

## Current branch context

- Branch: `capability-reset`
- PAPER-01 baseline: `cc512d7df7d258a4630d39ff64270390b55fb1f7`
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
- PAPER-01: `IMPLEMENTED / EXTERNAL REVIEW REQUIRED`.
- PAPER-02: `NOT STARTED / NOT AUTHORIZED`.
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
maximum leverage 1x. No pilot trade was started by DOC-01.

## Next decision required

External review of PAPER-01 against the canonical documentation and the
implementation. Do not start PAPER-02 without external GO.
