# Jax Roadmap

`Docs/ROADMAP.md` is the single authoritative roadmap. Product meaning comes
from `Docs/JAX_PRODUCT_CHARTER.md`; current status comes from `Docs/STATUS.md`;
capability maturity comes from `Docs/CAPABILITY_MATRIX.md`; current build
routing comes from `Docs/BUILD/CURRENT_PACKAGE.md`.

## Purpose

Jax is developed as a bounded, evidence-first decision system. This roadmap
defines sequencing and promotion gates without rewriting scientific evidence.

## Product objective

Build a trustworthy event-driven trading research and decision system that
explains events, resolves affected assets, tests causal theses, rejects weak
setups, requires human approval, learns from outcomes, and remains paper-only
until a separate scientific and safety decision authorizes anything later.

## Initial trader model

Jax's initial trader is the **EVENT-DRIVEN SHORT-HORIZON SWING TRADER**.

- Primary catalyst: event/news/source-backed evidence.
- Technicals: supporting, confirming, or veto capability; never the sole catalyst.
- Entry: human approval required.
- Exploratory hold: 1–5 trading sessions; typical 2–3; hard maximum 5.
- Default decision: `NO_TRADE`.

## Open-position behaviour

At entry, Jax freezes the thesis, causal rationale, stop, target, hard time
exit, and policy versions. Open positions receive relevant-evidence
reassessment and at least one scheduled review per trading session. Generic or
irrelevant news must not change a thesis.

## Exit policy

The supported exit reasons are `STOP`, `TARGET`, `THESIS_INVALIDATED`,
`RISK_KILL`, `TIME_LIMIT`, and `MANUAL_OPERATOR`. A closed-market invalidation
is recorded as `EXIT_AT_NEXT_TRADABLE_SESSION`. The session calendar is
explicit; unknown calendar state fails closed.

## Human control

Every new exploratory entry requires explicit human approval. Risk, execution
environment, and paper-only safety checks remain deterministic.

## Exploratory vs formal paper

`EXPLORATORY_PAPER` is a product-learning mode. It may use prospective event
and news evidence, may evolve through versioned rules, and does not prove an
edge. `FORMAL_FORWARD_PAPER` is a later, separately authorized scientific mode
with frozen rules, future-only identity, and no tuning. Exploratory records may
not populate or be relabelled as formal evidence. `DEMONSTRATED_EDGE` is a
separate evidence conclusion and is not implied by either paper mode.

## Current scientific position

VAL-03R4B is `COMPLETE / FAILED_VALIDATION / EXTERNALLY_REVIEWED`.
`ma_crossover_v1` is `CLOSED / FAILED HISTORICAL VALIDATION`.
The formal forward paper is `NO-GO`; trading edge remains `NOT DEMONSTRATED`.
VAL-04A is `COMPLETE / GO`. VAL-04B is `DEFERRED / NOT CURRENT NEXT STEP`.

## Accepted capability foundation

Phases 00–12 and CR-02G are accepted technical capability foundations at their
documented gates. Their evidence remains supporting material; acceptance does
not promote a strategy or prove a trading edge.

## Current position

| Area | Status |
| --- | --- |
| PAPER-01 | **GO / EXTERNALLY REVIEWED** |
| PAPER-01A | **COMPLETE / SUPERSEDED BY PAPER-01B REMEDIATION** |
| PAPER-01B | **IMPLEMENTED / ACCEPTED DIRECTIONALLY** |
| PAPER-01C | **ACCEPTED / EXTERNALLY REVIEWED** |
| PAPER-02A1 | **GO / EXTERNALLY REVIEWED** |
| PAPER-02A2 | **IMPLEMENTED / EXTERNAL CONFIGURATION REQUIRED** |
| PAPER-02A3 | **ACTIVATED / EXTERNALLY REVIEWED / SUPERSEDED BY OPS-02A** |
| PAPER-02 | **ABORTED / HISTORICALLY PRESERVED / 0 GENUINE PROSPECTIVE OPPORTUNITIES** |
| PAPER-02R | **GO / EXTERNALLY REVIEWED / INCIDENT PRESERVED** |
| OPS-01 | **GO / EXTERNALLY REVIEWED** |
| OPS-02A | **EXECUTED / PAPER-02 HISTORICALLY PRESERVED / SUPERSEDED BY OPS-02B AND OPS-02C1** |
| OPS-02B1 | **IMPLEMENTED / VALIDATED / SUPERSEDED BY OPS-02B3** |
| OPS-02B3 / OPS-02B | **GO / EXTERNALLY REVIEWED** |
| OPS-02C1 | **GO / EXTERNALLY REVIEWED** |
| CORE-READINESS-01 | **COMPLETE / BLOCKED_BEFORE_HYPOTHESIS_DESIGN** |
| CORE-READINESS-02A3 | **GO / EXTERNALLY REVIEWED / CANDIDATE ECONOMIC INPUT CONTRACT** |
| CORE-READINESS-02A4 / 02A4R | **GO / EXTERNALLY REVIEWED / CANONICAL QUEUE HANDOFF** |
| CORE-READINESS-02B1 | **GO / EXTERNALLY REVIEWED / POLICY + TEMPORAL PREREQUISITES** |
| CORE-READINESS-02B2 | **IMPLEMENTED / EXTERNAL REVIEW REQUIRED / ALPACA QUOTE TEMPORAL SEMANTICS** |
| CORE-READINESS-02B | **NOT YET EXECUTED / ISOLATED GENUINE-MARKET END-TO-END PROOF** |
| FORMAL_FORWARD_PAPER | **NOT STARTED** |
| Trading edge | **NOT DEMONSTRATED** |
| Phase 13 | **NOT STARTED / BLOCKED** |
| HARNESS-00 | **GO / EXTERNALLY REVIEWED** |
| HARNESS-01 | **GO / EXTERNALLY REVIEWED / FOUNDATION CONTRACTS** |
| HARNESS-02 | **IMPLEMENTED / EXTERNAL REVIEW REQUIRED / OFFLINE CONTEXT BUILDER** |
| Harness runtime/JaxMind/evaluator | **NOT IMPLEMENTED / NOT INTEGRATED** |
| Optional future commercialisation | **DEFERRED / NOT A CURRENT OBJECTIVE** |

## PAPER-01

PAPER-01 is the implemented event-driven exploratory paper-trader loop. Its
contracts, lifecycle tests, paper-workflow proof, and operator context are
documented in `Docs/BUILD/PAPER-01.md`,
`Docs/BUILD/PAPER-01-CAPABILITY-MAP.md`,
`Docs/TRADING_BRAIN/JAX_TRADER_MODEL_V1.md`, and `Docs/PAPER_TRADING/`.
PAPER-01 is GO / EXTERNALLY REVIEWED. PAPER-01B adds durable exploratory
identity, restart/reconciliation checks, canonical evidence gating, frozen
provenance, idempotent reviews, deterministic accounting, and a minimal
operator read model. PAPER-01C closes the runtime entry/review loop through the
approved-entry queue and scheduled worker. PAPER-02 readiness adds a frozen
prospective protocol, pilot identity, non-trade opportunity ledger, new-
evidence classification, latency capture, incidents, descriptive metrics, and
operator read models. That readiness work preceded the historical PAPER-02
activation and subsequent OPS-02A abort.

## PAPER-02

PAPER-02-2026-01 was activated in `EXPLORATORY_PAPER` mode under
`paper-02-protocol-v1`, with a target of 50 genuine prospective opportunities
over the frozen 90-calendar-day window ending 2026-12-17. OPS-02A canonically
aborted it after proving that the 16 persisted economic rows were pre-activation
PostgreSQL integration-test fixtures. The final genuine prospective sample is
`0 / 50`; the pilot is historically preserved and cannot establish a trading
edge.

PAPER-02A2 defined the genuine prospective World Monitor intake boundary and
PAPER-02A3 recorded activation. OPS-02A records the closure incident. The source identity, calendar, universe, risk,
entry, candidate/evidence, trader, thesis, exit, and cost identities remain
frozen. Retain WATCH, NO_TRADE, unresolved, rejected, missing-data, and approved
cases. No historical replay may be used as a prospective observation.

During the OPS-02B controlled proof, runtime safety was `PAPER`,
`ExecutionAuthority=NONE`, broker execution and execution were disabled, maximum
leverage was 1x, and the broker was skipped. No unexpected economic write
occurred. This proves operational integrity for that bounded run; it does not
prove a strategy, predictive skill, profitability, an economic edge, or
formal-forward readiness. The PAPER-02 activation created no genuine
opportunity, order, fill, position, or formal-evidence row; the pilot was later
aborted. Existing integration-test fixtures are excluded from its production
sample.

The canonical protocol remains
`Docs/PAPER_TRADING/PAPER-02-PROSPECTIVE-PILOT-PROTOCOL.md`; the activation and
closure state are recorded in the durable database and current status
documents.

PAPER-02R is the corrective incident-preservation and paper-runtime-readiness
package. It preserves the frozen PAPER-02 history, corrects durable exit
approval, economic accounting, ledger reconciliation, data-quality semantics,
and required PostgreSQL validation. It has prospective engineering effect only;
it does not continue the pilot or create a demonstrated-edge claim.

## Exploratory learning programme

The exploratory programme preserves rejected, watched, and no-trade cases;
captures latency, thesis changes, evidence, costs, MFE/MAE, and session-based
outcomes; and does not convert learning outcomes into decision inputs without a
separate reviewed policy change.

## Future playbooks

Future playbooks may be added only after the event-driven initial trader has
been reviewed. No generic multi-week technical swing policy is active.

## Future formal validation

FORMAL-01 is planned only after exploratory observations, playbook selection,
preregistration, calibration, and a separate authorization. Historical
validation artifacts remain immutable.

## Calibration / uncertainty

Confidence is a bounded, versioned uncertainty field. It is not a promise of
probability, profitability, or promotion. Unknown, stale, contradictory, or
insufficient evidence must remain visible.

## Risk philosophy

Risk is a deterministic veto and sizing boundary. Maximum leverage remains 1x;
human approval cannot override invalid safety invariants.

## Paper safety

Live trading, broker mutation, real orders, real fills, and real positions
remain disabled. Paper execution uses the existing isolated venue with
`Environment=PAPER` and `ExecutionAuthority=NONE`.

## Phase 13

Phase 13 optional personal live execution is **NOT STARTED / BLOCKED**. It is
not part of the current implementation sequence.

## Scientific validation principles

No look-ahead, retrospective relabelling, uncontrolled tuning, or selection
from outcome knowledge. Scientific claims require frozen identities, preserved
provenance, adequate samples, falsification, and external review.

## Architecture principles

Keep event resolution, evidence, quant, risk, workflow, approval, paper
execution, monitoring, and outcomes behind their existing contracts. Do not
create a parallel ledger or bypass the two-runtime modular-monolith boundary.

## Harness architecture programme

HARNESS-00 is **GO / EXTERNALLY REVIEWED**. HARNESS-01 is **GO / EXTERNALLY
REVIEWED** and adds versioned, deterministic foundation contracts in
`internal/modules/harnesscontracts`. HARNESS-02 is **IMPLEMENTED / EXTERNAL
REVIEW REQUIRED** and adds only an offline, deterministic retrieval/context
builder in `internal/modules/contextbuilder`, with no model or runtime
integration.
The two separate concerns remain the Codex engineering harness and the Jax
runtime/research harness around JaxMind. Their canonical architecture,
evaluation specification, detailed roadmap, gap analysis, and HARNESS build
evidence are in `Docs/HARNESS/` and `Docs/BUILD/`.

HARNESS-01 and HARNESS-02 must not enter the active PAPER-02 runtime, modify its context or
evidence selection, alter policies or thresholds, affect the 50-opportunity
sample, or begin FORMAL_FORWARD_PAPER. HARNESS-03 and all later packages remain
gated. The harness runtime and JaxMind are not integrated.

## Deferred Context Engineering

**Status: RECORD / RECONCILED BY HARNESS-00, HARNESS-01, AND HARNESS-02 — DO NOT IMPLEMENT AS A SEPARATE TRACK.** Context
Engineering is a future cross-cutting capability spanning the existing Phase 06
Research & Recommendation Engine and Phase 08 Controlled AI Tools & Durable
Research Agents. Its retained planning record is
`Docs/CONTEXT_ENGINEERING_ROADMAP_RECORD.md`; the canonical detailed
architecture and future implementation gates are now in
`Docs/HARNESS/ARCHITECTURE.md`, `Docs/HARNESS/EVALUATION.md`, and
`Docs/HARNESS/ROADMAP.md`.

The governing principle is **context is not storage**: durable stores retain
raw evidence, research artifacts, task state and memory, while a future
Context Builder assembles a bounded, provenance-aware package for the current
research objective and JaxMind invocation. Evidence, Memory, State and Context
remain distinct concepts.

Implementation is deferred until trustworthy evidence acquisition and
normalisation, provenance-aware retrieval, durable checkpoint/task state,
forecasting/uncertainty/calibration, BlackBox/auditability, and
Experience/Judgement dependencies are suitably available and externally
reviewed. The future contract must include explicit context budgets,
objective-relevant selection including contradictions and unknowns,
reconstructable provenance, structured checkpoint/compaction, relevant-only
memory retrieval, and integration with existing architecture rather than a
second AI system.

Context Quality Evaluation is mandatory. It must force checkpoint/compaction
and resume, test objective/constraint retention, omitted-evidence retrieval,
distinct evidence/memory/state handling, contradiction preservation,
non-repetition, non-invention, non-premature completion, context
reconstruction and stability under changed selection, while measuring bounded
context quality, cost, retrieval volume, unnecessary context and evidence
coverage where practical.

This record changes no runtime behaviour and authorizes no PAPER-02 activation,
formal paper, live execution or Phase 13 work.

## AI role

AI may interpret and propose within evidence and tool boundaries. It cannot
grant execution authority, bypass human approval, change risk policy, or turn a
paper observation into scientific proof.

## Data/evidence principles

Evidence is source-backed, timestamped, immutable by identity, and separated
from inference. Canonical issuer, instrument, event, and evidence references
are preferred to copied model memory.

## Outcome integrity

Outcomes describe what happened after a decision. They do not silently become
features, thresholds, or selection criteria for the same policy.

## Versioning

Trader model, thesis contract, candidate, risk, entry, exit, cost, and formal
run versions remain attached to records. Old records retain their versions.

## Promotion vocabulary

`PLANNED` means future intent; `IMPLEMENTED` means code or documentation
exists; `TESTED` means automated coverage; `PROVEN` means evidence supports a
specific claim; `COMPLETE / GO` is a technical-lead gate. None of these terms
alone means a demonstrated trading edge.

## Working Jax v1 readiness criterion

CORE-READINESS-01 audited the current repository and reviewed runtime evidence
against this minimum end-to-end loop:

```text
data
→ analysis
→ decision
→ risk
→ human approval
→ paper trade
→ monitor
→ exit
→ outcome
```

The audit verdict is `BLOCKED_BEFORE_HYPOTHESIS_DESIGN`: candidate approval
actor identity is caller-header controlled; candidate risk/approval does not
produce the portfolio-risk-bound exploratory entry request; and genuine
market-price provenance/freshness/no-lookahead semantics have not been proven
in the corrected runtime. OPS-02B did not prove a strategy, predictive skill,
profitability, an economic edge, or formal-forward readiness. Do not revive
`ma_crossover_v1` or reinterpret its failed frozen recovery validation.

## Current development sequence

```text
PAPER-01
ACCEPTED FOUNDATION

↓

PAPER-02
ABORTED / HISTORICALLY PRESERVED /
0 GENUINE PROSPECTIVE OPPORTUNITIES

↓

PAPER-02R / OPS-01 / OPS-02B / OPS-02C1
CORRECTED RUNTIME AND OPERATIONAL INTEGRITY
GO / EXTERNALLY REVIEWED

↓

CORE-READINESS-01
WORKING JAX V1 / SCIENTIFIC READINESS AUDIT
COMPLETE / BLOCKED BEFORE HYPOTHESIS DESIGN

↓

CORE-READINESS-02A4
CANONICAL CANDIDATE → PORTFOLIO RISK → HUMAN APPROVAL → PAPER QUEUE
GO / EXTERNALLY REVIEWED

↓

CORE-READINESS-02B1
TECHNICAL-PROOF POLICY + MARKET TEMPORAL PREREQUISITES
IMPLEMENTED / EXTERNAL REVIEW REQUIRED

↓

CORE-READINESS-02B
ISOLATED GENUINE-MARKET END-TO-END PROOF
REMAINING / NOT STARTED

↓

ONLY AFTER THAT AUDIT:
A SEPARATELY SELECTED AND PREREGISTERED
EXPLORATORY ECONOMIC HYPOTHESIS

↓

FUTURE EXPLORATORY PROSPECTIVE PILOT
NOT STARTED / NOT AUTHORIZED

↓

PAPER-03
OUTCOME ANALYSIS
FUTURE

↓

PAPER-04
PLAYBOOK SELECTION + PREREGISTRATION
FUTURE

↓

FORMAL-01
FORMAL FORWARD PAPER
FUTURE / SEPARATE AUTHORIZATION
```

## Deferred work

CORE-READINESS-01 is complete and blocks hypothesis design. CORE-READINESS-02A4
is implemented through the account-scoped queue and requires external review.
CORE-READINESS-02B1 is GO / EXTERNALLY REVIEWED for technical proof policy,
temporal market semantics, and disposable routing. CORE-READINESS-02B2 corrects
quote/trade provenance and remains subject to external review. CORE-READINESS-02B
remains the isolated genuine-market end-to-end proof and has not started. No strategy is selected by
this roadmap update. PAPER-03, PAPER-04,
and FORMAL-01 remain future work; FORMAL-01 requires separate authorization.
VAL-04B, HARNESS-04, live-readiness, Phase 13, and commercialisation are not
current implementation work. No new prospective pilot is authorized; live
execution remains unauthorized.

## Current success criterion

CORE-READINESS-01 remains `BLOCKED_BEFORE_HYPOTHESIS_DESIGN`. CORE-READINESS-02A4
and 02A4R are `GO / EXTERNALLY REVIEWED`; 02B1 is `GO / EXTERNALLY REVIEWED`
and 02B2 requires external review. CORE-READINESS-02B must still provide isolated
genuine-market end-to-end proof and external review.
Neither package is strategy selection or a profitability claim.

## North Star

Jax should become a trustworthy evidence-first research and decision partner:
source-backed, causally explicit, skeptical by default, human-controlled, and
honest about uncertainty.
