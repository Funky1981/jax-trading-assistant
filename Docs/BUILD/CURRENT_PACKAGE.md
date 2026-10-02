# Current Build Package

Current routing: **CORE-READINESS-02B — SEQUENTIAL GENUINE-MARKET PROOF
CONTINUATION**. The first bounded genuine attempt is recorded as
`PARTIAL GENUINE PROOF EXECUTED / BLOCKED_NO_ELIGIBLE_GENUINE_EVENT`; its event
remained WATCH and no candidate was created. Continue only from frozen
disposable World Monitor cursor `72795`, at a valid regular US PAPER session.
The current host-clock blocker was cleared during the recorded proof. See
`Docs/BUILD/CORE-READINESS-02B-GENUINE-MARKET-END-TO-END-PROOF.md`.

CORE-READINESS-02B2R4R1 — SINGLE-INSTRUMENT LIFECYCLE EXCLUSIVITY — was
externally reviewed before the genuine proof and is included in its proof code;
it is no longer the active package.
CORE-READINESS-02B2R3R1 remains **GO / EXTERNALLY REVIEWED** at
`555384f48b2d0f62d68e0fd2fb1a04efee5a9fbb`. CORE-READINESS-02B2R4 implemented
full-fill atomicity and is superseded for final acceptance by R4R1. R4R1 adds
account+instrument-scoped approval serialization, runtime exposure checks, and
pre-exit ledger-to-lifecycle quantity binding. Generic PaperVenue partial fills
remain available; the canonical v1 lifecycle remains single-exposure and
full-fill only. See
`Docs/BUILD/CORE-READINESS-02B2R4R1-SINGLE-INSTRUMENT-EXCLUSIVITY.md`.
CORE-READINESS-02B remains blocked because no eligible genuine candidate was
produced in the prior bounded sample. This WATCH-only infrastructure result is
not a strategy failure, hypothesis-readiness result, or trading-edge claim.
CORE-READINESS-02A4 and 02A4R remain **GO / EXTERNALLY REVIEWED**. The 02A4
queue remains account-scoped and is not consumed by 02A4. CORE-READINESS-01
remains **BLOCKED_BEFORE_HYPOTHESIS_DESIGN** until CORE-READINESS-02B is
completed and externally reviewed. No strategy is selected, and no new pilot
or economic activity is authorized.

CORE-READINESS-02A4R is a narrow identity-compatibility correction to the
PortfolioSnapshot v1 wire representation: false `KnownEmpty` is omitted to
preserve historical non-empty SnapshotIDs, while true remains explicit. The
correction does not rewrite historical records and remains within the 02A4
canonical handoff boundary.

CORE-READINESS-02A3 established the typed append-only candidate economic-input
contract. CORE-READINESS-02A4 connects persisted candidates and explicit
economic inputs to current PAPER portfolio risk, authenticated human decision,
durable workflow/PaperIntent, and one account-scoped queue row. See
`Docs/BUILD/CORE-READINESS-02A3-CANDIDATE-ECONOMIC-INPUTS.md` and
`Docs/BUILD/CORE-READINESS-02A4-CANONICAL-PAPER-HANDOFF.md`. The legacy World
Monitor APPROVE route remains blocked; only the explicit canonical PAPER
handoff can approve that candidate path. The queue is not consumed by 02A4.

Read `Docs/BUILD/CORE-READINESS-01-WORKING-JAX-V1-AUDIT.md`. 02A4 addresses the
server-owned candidate/risk/approval-to-queue seam and JWT-bound decision actor.
CORE-READINESS-02B1 is **GO / EXTERNALLY REVIEWED** as the prerequisite
policy/temporal contract. CORE-READINESS-02B2R2 corrected post-latency runtime
liveness. CORE-READINESS-02B2R3R1 closes canonical market identity across
approval, entry, review and exit and is externally reviewed at the SHA above.
CORE-READINESS-02B2R4R1's reviewed accounting/lifecycle exclusivity contract is
part of the accepted proof-code chain. CORE-READINESS-02B is the current package:
continue at most one new sequential event-page attempt from cursor `72795` and
only during a valid market session. Do not scan backwards, cherry-pick pages,
or change the reviewed policy.
OPS-01 remains synthetic/disposable full-loop evidence;
OPS-02B remains real-runtime event-intake evidence with zero economic activity.

OPS-02A: **EXECUTED / PAPER-02 HISTORICALLY PRESERVED / SUPERSEDED BY OPS-02B AND OPS-02C1**.
The artifact-forensics, canonical closure, World Monitor verification, and
Compose account-wiring record is in
`Docs/BUILD/OPS-02A-PAPER-02-ARTIFACT-FORENSICS-WORLD-MONITOR-DEPLOYMENT-PREP.md`.

OPS-02B1: **IMPLEMENTED / VALIDATED / SUPERSEDED BY OPS-02B3**.
Read `Docs/BUILD/OPS-02B1-SAFE-DEPLOYMENT-PREFLIGHT-CORRECTION.md`. The
process-level worker gates were a prerequisite for the later controlled
`jax-trader` deployment proof.

OPS-02B3 / OPS-02B: **GO / EXTERNALLY REVIEWED**. Read
`Docs/BUILD/OPS-02B-CONTROLLED-PAPER-RUNTIME-DEPLOYMENT-PROOF.md`. The twelve
historical World Monitor integration-test fixtures were disabled in the normal
database with explicit operator approval, historical evidence was preserved,
and the bounded PAPER runtime passed with execution disabled and the broker
skipped. The proof recorded an isolated PAPER account and zero unexpected
economic writes. It proves operational integrity for that controlled run, not
strategy quality or an economic edge. No new pilot or live execution is
authorized.

OPS-02C1: **GO / EXTERNALLY REVIEWED**. Read
`Docs/BUILD/OPS-02C1-PILOT-TEST-FIXTURE-CLOSURE.md`. The five
confirmed historical `pilot-restart-*` fixtures were transitioned from ACTIVE
to ABORTED through admission-blocking `RecordIncident` calls. All dependent
evidence was preserved and PAPER-02 remains unchanged.

At the OPS-02C1 closure checkpoint, the normal database held six ABORTED
exploratory pilots and zero ACTIVE pilots. OPS-02B verified zero enabled
strategy instances after disabling the twelve confirmed test fixtures. These
are the last reviewed database results recorded by those packages, not a fresh
database query for this routing update.

OPS-01: **GO / EXTERNALLY REVIEWED** at
`7776e763a38f941ab0d704619bcbe3361921b591`.

PAPER-02R: **GO / EXTERNALLY REVIEWED / INCIDENT PRESERVATION & CORRECTED
PAPER-RUNTIME READINESS**. Reviewed SHA: `c188721af5d3fe466c7999bf3ce01859f125537a`.
Read
`Docs/BUILD/PAPER-02R-INCIDENT-PRESERVATION-RUNTIME-READINESS.md` for the
reviewed package record. The exact pre-existing reviewed PostgreSQL/test
files are incorporated as input and are not discarded merely to obtain a clean
worktree.

HARNESS-03: **IMPLEMENTED / EXTERNAL REVIEW REQUIRED / CHECKPOINT, RESUME & COMPACTION**.
The historical PAPER-02-2026-01 exploratory pilot is frozen and remains
unchanged after its OPS-02A abort.
No model call, JaxMind integration, evaluator execution, task-state
persistence, compaction, full memory behavior, orchestration, or trading
integration is authorized by this current package.

PAPER-02A3 activated the frozen prospective protocol. OPS-02A closes
`PAPER-02-2026-01` as `ABORTED / HISTORICALLY PRESERVED / 0 GENUINE
PROSPECTIVE OPPORTUNITIES` after classifying all 16 pre-activation economic
rows as PostgreSQL integration-test fixtures. Runtime remains `PAPER`,
`ExecutionAuthority=NONE`, broker execution disabled, execution disabled, and
maximum leverage 1x. Formal forward paper, demonstrated edge, live trading,
and Phase 13 remain unauthorized.

## HARNESS-03 — checkpoint, resume, and compaction package

Read in this order:

1. `Docs/JAX_PRODUCT_CHARTER.md`
2. `Docs/ROADMAP.md`
3. `Docs/STATUS.md`
4. `Docs/ARCHITECTURE.md`
5. `Docs/HARNESS/README.md`
6. `Docs/HARNESS/ARCHITECTURE.md`
7. `Docs/HARNESS/EVALUATION.md`
8. `Docs/HARNESS/ROADMAP.md`
9. `Docs/HARNESS/ENGINEERING-HARNESS-GAP-ANALYSIS.md`
10. `Docs/BUILD/HARNESS-01-FOUNDATION-CONTRACTS.md`
11. `Docs/BUILD/HARNESS-02-RETRIEVAL-CONTEXT-BUILDER.md`
12. `Docs/BUILD/HARNESS-03-CHECKPOINT-RESUME-COMPACTION.md`

HARNESS-03 remains a separately reviewed capability reference. Its runtime is
not integrated with JaxMind or PAPER-02, and HARNESS-04 remains unimplemented
and unauthorized.

## PAPER-02R — incident preservation and corrected runtime readiness

The externally reviewed corrective package is documented in
`Docs/BUILD/PAPER-02R-INCIDENT-PRESERVATION-RUNTIME-READINESS.md`. It includes
the exact previously reviewed dirty test/integration work as pre-existing
input, validates and completes it, and preserves the frozen PAPER-02 incident
boundary. It does not continue the pilot, start HARNESS-04, enable broker
execution, or enable live trading.

## OPS-01 — corrected runtime operational readiness

The operational readiness and end-to-end proof is documented in
`Docs/BUILD/OPS-01-CORRECTED-RUNTIME-OPERATIONAL-READINESS.md`. OPS-01
validates bounded PAPER runtime operation, durable World Monitor diagnostics,
approval identity, restart recovery, and accounting reconciliation. It does
not establish a strategy edge or authorize a new pilot, broker execution, or
live trading.

## PAPER-01 / PAPER-02 — event-driven exploratory paper-trader reference

Status: **PAPER-02 ABORTED / HISTORICALLY PRESERVED / 0 GENUINE PROSPECTIVE OPPORTUNITIES**.

PAPER-01C is accepted and externally reviewed. PAPER-02A2 supplied the frozen
prospective exploratory protocol, durable pilot identity, non-trade opportunity
ledger, new-evidence/latency capture, market quality, incidents, descriptive
metrics, and protected operator read models; PAPER-02A3 activated the pilot and
OPS-02A later recorded its canonical abort.
The pilot was an operational failure, not a failed trading strategy. It cannot
demonstrate an edge, does not create formal evidence, and does not permit live
execution.

Read in this order:

1. `Docs/JAX_PRODUCT_CHARTER.md`
2. `Docs/ROADMAP.md`
3. `Docs/STATUS.md`
4. `Docs/BUILD/PAPER-01.md`
5. `Docs/BUILD/PAPER-01-CAPABILITY-MAP.md`
6. Relevant paper-trading contracts under `Docs/PAPER_TRADING/`

The PAPER-02 activation decision was historically recorded, then the pilot was
canonically aborted by OPS-02A. Prospective observation collection is not
active. Its frozen protocol and intake implementation remain historical
records; they do not authorize a new pilot. No formal forward-paper run,
strategy selection, Phase 13 work, broker execution, or live trading is
authorized by this package.

The historical incident and preservation boundary are documented in
`Docs/PAPER_TRADING/PAPER-02-2026-01-INCIDENT-REVIEW.md`. PAPER-02R does not
rewrite the frozen protocol or historical sample.
