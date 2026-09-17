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
| PAPER-01 | **NO-GO / PENDING PAPER-01B EXTERNAL RE-REVIEW** |
| PAPER-01A | **COMPLETE / SUPERSEDED BY PAPER-01B REMEDIATION** |
| PAPER-01B | **IMPLEMENTED / EXTERNAL RE-REVIEW REQUIRED** |
| PAPER-02 | **NOT AUTHORIZED** |
| FORMAL_FORWARD_PAPER | **NOT STARTED** |
| Trading edge | **NOT DEMONSTRATED** |
| Phase 13 | **NOT STARTED / BLOCKED** |
| Optional future commercialisation | **DEFERRED / NOT A CURRENT OBJECTIVE** |

## PAPER-01

PAPER-01 is the implemented event-driven exploratory paper-trader loop. Its
contracts, lifecycle tests, paper-workflow proof, and operator context are
documented in `Docs/BUILD/PAPER-01.md`,
`Docs/BUILD/PAPER-01-CAPABILITY-MAP.md`,
`Docs/TRADING_BRAIN/JAX_TRADER_MODEL_V1.md`, and `Docs/PAPER_TRADING/`.
PAPER-01 remains NO-GO pending external re-review of PAPER-01B. PAPER-01B
adds durable exploratory identity, restart/reconciliation checks, canonical
evidence gating, frozen provenance, idempotent reviews, deterministic
accounting, and a minimal operator read model. It does not authorize a pilot.

## PAPER-02

PAPER-02 is not started and not authorized. Its future bounded exploratory
pilot definition is retained in `Docs/BUILD/PAPER-02.md`, but no pilot may
begin from this package. The exact PAPER-02 versus FORMAL-01 sequencing remains
subject to an external decision after PAPER-01B passes re-review.

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
IMPLEMENTED / EXTERNAL REVIEW REQUIRED

↓

PAPER-02
PROSPECTIVE EXPLORATORY PAPER PILOT
NOT STARTED / NOT AUTHORIZED

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

VAL-04B, PAPER-02, FORMAL-01, live-readiness, Phase 13, and commercialisation
are not current implementation work. Old plans, phase contracts, and ProjectOS
templates cannot authorize them.

## Current success criterion

The current success criterion is a coherent, externally reviewable PAPER-01
control plane and implementation that preserves safety and makes the next
decision unambiguous. It is not profitability.

## North Star

Jax should become a trustworthy evidence-first research and decision partner:
source-backed, causally explicit, skeptical by default, human-controlled, and
honest about uncertainty.
