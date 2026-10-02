# Jax Current Status

PAPER-02-2026-01: **ABORTED / HISTORICALLY PRESERVED / 0 GENUINE
PROSPECTIVE OPPORTUNITIES**. Its artifact forensics and canonical closure are
recorded in `Docs/BUILD/OPS-02A-PAPER-02-ARTIFACT-FORENSICS-WORLD-MONITOR-DEPLOYMENT-PREP.md`.

PAPER-02R: **GO / EXTERNALLY REVIEWED**. Historical incident preservation and
corrected PAPER-runtime readiness are complete; the package did not continue
PAPER-02 or demonstrate an edge.

OPS-01: **GO / EXTERNALLY REVIEWED** at
`7776e763a38f941ab0d704619bcbe3361921b591`.

OPS-02B: **GO / EXTERNALLY REVIEWED**. Its bounded PAPER runtime proof and
authorised disablement of twelve historical strategy fixtures are recorded in
`Docs/BUILD/OPS-02B-CONTROLLED-PAPER-RUNTIME-DEPLOYMENT-PROOF.md`.

OPS-02C1: **GO / EXTERNALLY REVIEWED**. The five confirmed historical
`pilot-restart-*` fixtures were closed through the canonical admission-blocking
incident path; historical evidence and PAPER-02 were preserved. See
`Docs/BUILD/OPS-02C1-PILOT-TEST-FIXTURE-CLOSURE.md`.

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

## Historical branch context at HARNESS-02 start

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
- OPS-02A: `EXECUTED / PAPER-02 HISTORICALLY PRESERVED / SUPERSEDED BY OPS-02B AND OPS-02C1`.
- OPS-02B: `GO / EXTERNALLY REVIEWED`.
- OPS-02C1: `GO / EXTERNALLY REVIEWED`.
- PAPER-02A3: `ACTIVATED / SUPERSEDED BY OPS-02A`.
- PAPER-02: `ABORTED / HISTORICALLY PRESERVED / 0 GENUINE PROSPECTIVE OPPORTUNITIES`.
- CORE-READINESS-01: `COMPLETE / BLOCKED_BEFORE_HYPOTHESIS_DESIGN`.
- CORE-READINESS-02: `IN PROGRESS`; 02A4 awaits external review and 02B remains.
- CORE-READINESS-02A3: `GO / EXTERNALLY REVIEWED`; typed candidate economic
  input contract.
- CORE-READINESS-02A4 / 02A4R: `GO / EXTERNALLY REVIEWED`; the
  narrow PortfolioSnapshot v1 identity correction preserves legacy non-empty
  SnapshotIDs by omitting false `KnownEmpty`, while true remains explicit.
  No historical database rows or identities were rewritten. Canonical
  current-account portfolio risk, JWT-bound human decision, durable PaperIntent,
  and account-scoped queue. Queue consumption is outside scope.
- CORE-READINESS-02B1: `GO / EXTERNALLY REVIEWED`; technical-proof policy,
  split market freshness, candle completion semantics, and guarded disposable
  routing fixture only.
- CORE-READINESS-02B2: `IMPLEMENTED / EXTERNAL REVIEW REQUIRED`; separate Alpaca
  quote/trade timestamps, nullable persistence, bounded clock-skew, and strict
  availability validation. Its evidence is preserved in the package record.
- CORE-READINESS-02B2R2: `IMPLEMENTED / EXTERNAL REVIEW REQUIRED`; durable
  pending PAPER entry/exit orders wait for distinct post-latency persisted
  provider quotes; approval-time evidence remains unchanged.
- CORE-READINESS-02B2R3: `IMPLEMENTED / EXTERNAL REVIEW REQUIRED`; post-latency
  quote lookup resolves canonical instrument IDs through the shared explicit
  candidate-economic identity policy; quote provenance records the resolved
  symbol while order/fill identity remains canonical.
- CORE-READINESS-02B2R3R1: `GO / EXTERNALLY REVIEWED` at
  `555384f48b2d0f62d68e0fd2fb1a04efee5a9fbb`; canonical market identity and
  policy-drift closure across approval, entry, review, and exit.
- CORE-READINESS-02B2R4: `IMPLEMENTED / EXTERNAL REVIEW REQUIRED`; the
  canonical exploratory runtime waits for sufficient full-order liquidity and
  only persists entry/exit lifecycle transitions after one complete fill.
  Generic PaperVenue partial fills remain supported; superseded for final
  acceptance by R4R1.
- CORE-READINESS-02B2R4R1: `GO / EXTERNALLY REVIEWED` before the genuine proof;
  canonical approvals are exclusive per PAPER account/instrument, runtime
  repeats the durable exposure check, and exit quantity is bound to the
  lifecycle entry fill. See
  `Docs/BUILD/CORE-READINESS-02B2R4R1-SINGLE-INSTRUMENT-EXCLUSIVITY.md`.
- CORE-READINESS-02B: `PARTIAL GENUINE PROOF EXECUTED / BLOCKED_NO_ELIGIBLE_GENUINE_EVENT`.
  Attempts 1 and 2 passed their session/clock gates and processed bounded
  genuine World Monitor pages. Attempt 2 processed 25 events after disposable
  cursor `72795`, producing 20 `NO_TRADE`, 5 `WATCH`, no `CANDIDATE`, and 24
  unresolved asset resolutions. No candidate or economic lifecycle resulted.
  This is technical integrity evidence, not a strategy failure or edge claim.
  The next sequential continuation boundary is disposable cursor `76220`; the
  normal Jax database and cursor were not accessed. See
  `Docs/BUILD/CORE-READINESS-02B-GENUINE-MARKET-END-TO-END-PROOF.md`.
- Working Jax remains `BLOCKED_BEFORE_HYPOTHESIS_DESIGN` pending completion and
  external review of the full isolated genuine-market proof. Both recorded
  attempts passed their host-clock gates; Attempt 2 stopped at zero eligible
  candidates.
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

OPS-02B proved bounded PAPER runtime operation against real PostgreSQL and
genuine World Monitor input with execution disabled, broker execution skipped,
an isolated PAPER account, and zero unexpected economic writes. This was a
controlled proof; it does not claim that the runtime is running now or that
every component is proven. Live execution remains unauthorized. PAPER-02
created zero genuine prospective opportunities and remains aborted. The
16 pre-activation `position-pg-*`/`account-pg-*` PostgreSQL integration
fixtures remain preserved and excluded from the production sample.

The last reviewed database checkpoints were six ABORTED exploratory pilots and
zero ACTIVE pilots after OPS-02C1, and zero enabled strategy instances after
OPS-02B fixture disablement. These are the recorded package results, not a new
database query in this documentation update.

## Next recommended work (not yet authorized)

CORE-READINESS-01 found that the reusable core is not yet trustworthy enough
to begin hypothesis design. CORE-READINESS-02A4 implements the canonical
server-owned candidate-to-queue seam. CORE-READINESS-02B has a partial genuine
provider proof but stopped at a WATCH-only event with no eligible candidate.
The continuation is sequential from disposable World Monitor cursor `72795`;
the first attempt's current-tail anchoring was a technical-proof protocol
deviation and is not economic or strategy evidence. No strategy selection, normal
database mutation, new pilot, prospective evidence collection, runtime
deployment, or broker use is authorized here. PAPER-02 remains aborted and
historically preserved; HARNESS-04 is deferred, FORMAL_FORWARD_PAPER is not
started, and broker/live execution is not authorized.
