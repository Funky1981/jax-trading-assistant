# Jax Current Status

OPS-02A: **EXECUTED / PAPER-02 HISTORICALLY PRESERVED / DEPLOYMENT PREP**.
The artifact forensics and canonical closure are recorded in
`Docs/BUILD/OPS-02A-PAPER-02-ARTIFACT-FORENSICS-WORLD-MONITOR-DEPLOYMENT-PREP.md`.

OPS-01: **GO / EXTERNALLY REVIEWED**. Reviewed SHA:
`7776e763a38f941ab0d704619bcbe3361921b591`.

PAPER-02R: **GO / EXTERNALLY REVIEWED / HISTORICAL INCIDENT
PRESERVED**. The corrective package is routed through
`Docs/BUILD/PAPER-02R-INCIDENT-PRESERVATION-RUNTIME-READINESS.md`. It does not
continue PAPER-02-2026-01, change its frozen identity, start FORMAL_FORWARD_PAPER,
or authorize broker/live execution.

PAPER-02A3: **ACTIVATED / SUPERSEDED BY OPS-02A**. The frozen pilot was
subsequently closed through its canonical incident path.

PAPER-02-2026-01: **ABORTED / HISTORICALLY PRESERVED / 0 GENUINE
PROSPECTIVE OPPORTUNITIES**. It was an operational pilot failure, not a failed
trading strategy; no trading-edge conclusion can be drawn.

HARNESS-01: **GO / EXTERNALLY REVIEWED / FOUNDATION CONTRACTS**.
HARNESS-02: **IMPLEMENTED / EXTERNALLY REVIEWED / OFFLINE CONTEXT BUILDER**.
HARNESS-03: **IMPLEMENTED / EXTERNAL REVIEW REQUIRED / CHECKPOINT-RESUME-COMPACTION**.
The harness runtime/JaxMind is not integrated and has not entered PAPER-02.

This is the concise operational status. Roadmap sequencing is authoritative in
`Docs/ROADMAP.md`; capability maturity is in `Docs/CAPABILITY_MATRIX.md`.

## Current branch context

- Branch: `capability-reset`
- Repository HEAD at HARNESS-02 start: `ebd6be95437d5c260724a92997678634a3f2a096`
- Frozen PAPER-02 runtime code SHA: `521a4917c5e6be639ed8571cf383f22d99f1d077`
- Repository HEAD advances independently of the frozen PAPER-02 runtime SHA.
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
- OPS-01: `GO / EXTERNALLY REVIEWED` at `7776e763a38f941ab0d704619bcbe3361921b591`.
- OPS-02A: `EXECUTED / PAPER-02 HISTORICALLY PRESERVED / DEPLOYMENT PREP`.
- PAPER-02A3: `ACTIVATED / SUPERSEDED BY OPS-02A`.
- PAPER-02: `ABORTED / HISTORICALLY PRESERVED / 0 GENUINE PROSPECTIVE OPPORTUNITIES`.
- Harness architecture: `HARNESS-00 GO / EXTERNALLY REVIEWED`.
- Harness foundation contracts: `HARNESS-01 GO / EXTERNALLY REVIEWED`.
- Harness retrieval/context builder: `HARNESS-02 IMPLEMENTED / EXTERNALLY REVIEWED / OFFLINE`.
- Harness durable task state: `HARNESS-03 IMPLEMENTED / EXTERNAL REVIEW REQUIRED`.
- Harness JaxMind/evaluator runtime: `NOT IMPLEMENTED / NOT INTEGRATED`.
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
formal evidence row was created by activation. The 16 pre-activation
`position-pg-*`/`account-pg-*` PostgreSQL integration fixtures remain preserved
and excluded from the production sample.

## Next authorized work

Do not continue PAPER-02-2026-01 or reinterpret its preserved history. The next
safe step is separately reviewed corrected PAPER runtime deployment preparation
with an explicitly configured isolated `PAPER_ACCOUNT_ID`; no strategy research,
HARNESS-04, FORMAL_FORWARD_PAPER, broker execution, or live trading is
authorized by this status.
