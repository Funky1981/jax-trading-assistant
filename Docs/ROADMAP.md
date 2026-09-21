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
| PAPER-02A3 | **ACTIVATED / EXTERNALLY REVIEWED** |
| PAPER-02 | **ACTIVE / EXPLORATORY_PAPER / FROZEN** |
| PAPER-02R | **IMPLEMENTED / VALIDATION REQUIRED / INCIDENT PRESERVED** |
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
operator read models. It does not activate a pilot.

## PAPER-02

PAPER-02-2026-01 is active in `EXPLORATORY_PAPER` mode under
`paper-02-protocol-v1`. Its target is 50 genuine prospective opportunities over
the frozen 90-calendar-day window ending 2026-12-17. The pilot remains
exploratory and cannot establish demonstrated edge. Its sample is `0 / 50` at
the HARNESS-00 handover, and only observations first seen at or after the
activation timestamp may be admitted.

PAPER-02A2 defines the genuine prospective World Monitor intake boundary and
PAPER-02A3 records activation. The source identity, calendar, universe, risk,
entry, candidate/evidence, trader, thesis, exit, and cost identities remain
frozen. Retain WATCH, NO_TRADE, unresolved, rejected, missing-data, and approved
cases. No historical replay may be used as a prospective observation.

Runtime safety remains `PAPER`, `ExecutionAuthority=NONE`, broker execution
disabled, execution disabled, maximum leverage 1x, and no live/broker/IB order
path. The trader intake worker was not started during activation and no genuine
opportunity, order, fill, position, or formal evidence row was created by
activation. Existing restart-test fixtures cannot contribute to the production
pilot sample.

The canonical protocol remains
`Docs/PAPER_TRADING/PAPER-02-PROSPECTIVE-PILOT-PROTOCOL.md`; the activation and
current runtime state are recorded in the durable handover and current status
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

## Current development sequence

```text
PAPER-01
GO / EXTERNALLY REVIEWED

↓

PAPER-02
PROSPECTIVE EXPLORATORY PAPER PILOT
READY FOR EXTERNAL REVIEW / NOT ACTIVE

↓

PAPER-03
EXPLORATORY OUTCOME ANALYSIS
PLANNED

↓

PAPER-04
PLAYBOOK SELECTION / PREREGISTRATION
PLANNED

↓

FORMAL-01
FORMAL FORWARD-PAPER VALIDATION
PLANNED

↓

LIVE-READINESS GATE
BLOCKED

↓

PHASE 13
OPTIONAL PERSONAL LIVE EXECUTION
NOT STARTED / BLOCKED
```

## Deferred work

VAL-04B, FORMAL-01, live-readiness, Phase 13, and commercialisation are not
current implementation work. PAPER-02 is a readiness package only; an external
activation decision is still required. Old plans, phase contracts, and
ProjectOS templates cannot authorize it.

## Current success criterion

The current success criterion is a coherent, externally reviewable PAPER-01
control plane and implementation that preserves safety and makes the next
decision unambiguous. It is not profitability.

## North Star

Jax should become a trustworthy evidence-first research and decision partner:
source-backed, causally explicit, skeptical by default, human-controlled, and
honest about uncertainty.
