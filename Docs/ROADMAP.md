# Jax Roadmap

## What this document is

This is the authoritative, human-readable roadmap for current Jax development.
It defines the active architecture sequence, package status, promotion gates and
safety boundaries. Detailed package briefs and evidence documents support this
roadmap; they do not replace it.

Jax is an evidence-first market research and trading decision assistant. It is
not currently a live trading bot or an autonomous execution system. The default
decision remains `NO_TRADE`.

## Current position

| Area | Status |
| --- | --- |
| Phase 00 — Event / Issuer Foundation | **COMPLETE / GO** |
| Phase 01 — Canonical Contracts / Provenance / Replay | **COMPLETE / GO** |
| Phase 02 — Provider / Data Platform | **COMPLETE / GO**, including durable raw storage closure |
| Phase 03 — Core Financial Evidence | **COMPLETE / GO** |
| Phase 04 — World Monitor Intelligence | **COMPLETE / GO** |
| Phase 05 — Deterministic Quant Core | **COMPLETE / GO** |
| Phase 06 — Research & Recommendation Engine | **COMPLETE / GO** |
| Phase 07 — Evaluation, Replay & Backtesting | **COMPLETE / GO** |
| Phase 08 — Controlled AI Tools & Durable Research Agents | **COMPLETE / GO** |
| Phase 09 — Portfolio Intelligence & Deterministic Risk | **COMPLETE / GO** |
| Phase 10 — Workflow, HITL & Operational Safety | **COMPLETE / CONDITIONAL GO** |
| Phase 11 — High-Fidelity Paper Trading | **COMPLETE / CONDITIONAL GO** |
| Current implementation package | **PHASE 12 READINESS — HISTORICAL DATA & RESEARCH HYPOTHESIS** |
| Next package | **Historical data qualification for HYP-EVENT-001A — bounded readiness only** |

The current package context is the `capability-reset` sequence. Package
acceptance is recorded in the relevant evidence and review handovers.
WP-03.04 and WP-03.05 have received independent technical-lead **FINAL GO**.
WP-03.06 has received independent technical-lead FINAL GO and is now COMPLETE /
GO. The Phase-03 exit condition was demonstrated using a deterministic,
source-linked AAPL/Apple evidence packet containing real market, SEC/company
and Treasury macro/context evidence, immutable raw provenance and no
model-memory dependency. Phase 03 is COMPLETE / GO. Phase 04 is implemented
with its exit demonstrated and its external gate review pending.

## Roadmap status vocabulary

- `PLANNED` — intended, not yet designed or authorized.
- `DESIGNED` — architecture/specification exists; implementation is not implied.
- `IMPLEMENTED` — code or workflow exists.
- `TESTED` — automated tests exist; this is not proof of trading value.
- `PROVEN` — validated with appropriate real, paper, research or replay evidence.
- `ACCEPTED` / `COMPLETE / GO` — the technical-lead roadmap gate for the stated
  phase or package passed. It does not promote every related capability to
  `PROVEN` and does not authorize later work automatically.

Capability maturity and roadmap acceptance are related but separate. A legacy
component can be `IMPLEMENTED` and `TESTED` while remaining outside the accepted
current decision pipeline until it is revalidated against the current contracts,
provenance, provider and validation architecture.

## Active roadmap

### Phase 00 — Event / Issuer Foundation — ACCEPTED / COMPLETE

The accepted causal-attribution foundation provides typed causal attribution,
deterministic `DIRECT` / `PROXY` / `UNRESOLVED` policy, canonical issuer/asset
resolution, and no generic relevance-based causal inference.

Evidence: `Docs/evidence/PHASE-00-ISSUER-RESOLUTION-CLOSEOUT.md` and the Phase 00
package evidence under `Docs/evidence/`.

### Phase 01 — Canonical Contracts / Provenance / Replay — ACCEPTED / COMPLETE

The accepted foundation provides canonical contracts for Instrument, Issuer,
Event, Evidence, Observation, ResearchRun, QuantResult and Recommendation;
immutable provenance; SHA-256 content identity; canonical serialization;
replay/audit semantics; and compatibility rules. Research remains decision
support only and execution authority remains `NONE`.

Evidence: `Docs/evidence/WP-01.01-JAX-DOMAIN-CONTRACT-INVENTORY.md` through
`Docs/evidence/WP-01.04-REPLAY-AUDIT-COMPATIBILITY.md`.

### Phase 02 — Provider / Data Platform — ACCEPTED / COMPLETE

The accepted data platform provides provider registry/capabilities, exact raw
payload handling, deterministic normalization, freshness and provider health,
retry/rate-limit policy, source qualification, and separate content/acquisition
identities. Durable PostgreSQL `RawPayloadStore` is part of this completed
foundation; it preserves exact bytes and append-only acquisition identity.

Evidence: `Docs/evidence/WP-02.01-PROVIDER-REGISTRY-CAPABILITY-CONTRACT.md` through
`Docs/evidence/WP-02.07-DURABLE-RAW-PAYLOAD-STORAGE.md`.

### Phase 03 — Core Financial Evidence — COMPLETE / GO

Phase 03 adds trustworthy market, company and macro evidence to the accepted
platform. Its phase exit requires a source-linked evidence packet for a
representative US equity/ETF using real market, company and macro evidence
without relying on model memory.

| Package | Status | Evidence / closure |
| --- | --- | --- |
| WP-03.01 — Market price / OHLCV | **COMPLETE / GO** | `Docs/evidence/WP-03.01-MARKET-PRICE-OHLCV-PROVIDER-HARDENING.md`; timestamp/freshness closure in `Docs/evidence/WP-03.01A-MARKET-EVIDENCE-CLOSURE.md` |
| WP-03.02 — SEC / EDGAR / XBRL | **COMPLETE / GO** | `Docs/evidence/WP-03.02-SEC-EDGAR-XBRL-EVIDENCE.md`; SEC temporal-semantics closure is included in its accepted result |
| WP-03.03 — FRED / ALFRED macro observations + vintages | **COMPLETE / GO** | `Docs/evidence/WP-03.03-FRED-ALFRED-MACRO-EVIDENCE.md`; point-in-time/vintage leakage protection and macro-evidence closure are included |
| WP-03.04 — Economic release / calendar ingestion | **COMPLETE / GO** | `Docs/evidence/WP-03.04-ECONOMIC-RELEASE-CALENDAR-EVIDENCE.md` |
| WP-03.05 — Treasury / EIA / CBOE / CFTC source evaluation and first approved integrations | **COMPLETE / GO** | `Docs/evidence/WP-03.05-TREASURY-EIA-CBOE-CFTC-SOURCE-EVALUATION.md` |
| WP-03.06 — Evidence-quality / cross-source checks | **COMPLETE / GO** | `Docs/evidence/WP-03.06-EVIDENCE-QUALITY-CROSS-SOURCE-CHECKS.md` |

The paid Financial Datasets option remains accepted but is not required for
Phase-03 development acceptance. Its configured external credential returned
HTTP 401, so the bounded closure uses explicit Alpaca Basic SIP evidence as a
zero-cost development source. Alpaca is not thereby approved for production or
serious backtesting; feed completeness, corporate-action handling and licensing
remain future source-qualification questions. See
`Docs/evidence/PHASE-03-FREE-MARKET-DATA-CLOSURE.md` and
`Docs/evidence/PHASE-03-EXIT-GATE.md`.

Corrective packages such as WP-03.01A are closure history under their parent
package, not new permanent roadmap phases.

### Phase 04 — World Monitor Intelligence — COMPLETE / GO

Phase 04 builds corroborated event intelligence from the existing World Monitor
boundary while preserving raw evidence, provenance, temporal semantics,
deterministic replay and downstream safety boundaries. Its seven authorised
work packages are complete. The exact exit condition was accepted by external
GPT-5.6 Sol with `GO PHASE 04`. The reproducible internal verification record is at
`Docs/Jax-Roadmap-v2/04-world-monitor-intelligence/PHASE-04-INTERNAL-VERIFICATION.md`.

From Phase 04 onward, Autonomous Development Mode permits Codex to complete one
bounded package at a time with internal verification and self-review, then
continue within the authorised phase. External technical-lead review occurs at
the phase exit gate or on a hard stop. See
`Docs/Jax-Roadmap-v2/governance/AUTONOMOUS-DEVELOPMENT-MODE.md`.

### Phase 05 — Deterministic Quant Core — COMPLETE / GO

Phase 05 provides deterministic numerical context for research and risk using
frozen canonical datasets. It begins with
`WP-05.01 — Quant service/library boundary and versioned request/response contract`.
All nine authorised work packages are implemented and the exit condition is
demonstrated by the frozen-fixture proof in
`internal/modules/quant/phase05_exit_test.go`: every core quant result is
deterministic, versioned, tested against known values and independently
reproducible. External GPT-5.6 Sol accepted the Phase-05 handover and returned
`GO PHASE 05`; Phase 05 is therefore COMPLETE / GO. Phase-05 work created no recommendations, candidates, approvals,
orders, trades, fills or live execution authority. External technical-lead
review accepted the demonstrated exit. The full internal record is
`Docs/Jax-Roadmap-v2/05-deterministic-quant-core/PHASE-05-INTERNAL-VERIFICATION.md`.

### Phase 06 — Research & Recommendation Engine — COMPLETE / GO

Phase 06 combines accepted evidence and deterministic quant context with
bounded, provenance-preserving research reasoning. Its outputs remain
research-only `WATCH`, `NO_TRADE`, or `CANDIDATE` classifications with no
approval, order, trade, fill, or live execution authority. The first package is
`WP-06.01 — Define evidence packet contract` through `WP-06.07 — UI/API read
model for useful recommendations` are implemented. External GPT-5.6 Sol
accepted the demonstrated exit condition with `GO PHASE 06`. The full internal
record is `Docs/Jax-Roadmap-v2/06-research-recommendation-engine/PHASE-06-INTERNAL-VERIFICATION.md`.

### Phase 07 — Evaluation, Replay & Backtesting — COMPLETE / GO

Phase 07 is validation-first: it evaluates whether research and candidate logic
survive historical knowability, outcome-leakage, cost, walk-forward and
falsification checks. It must be capable of disproving an apparent edge. Its
outputs remain evaluation artifacts and do not create approvals, orders, trades,
fills or live execution authority. WP-07.01 through WP-07.08 are implemented and
internally verified. External GPT-5.6 Sol accepted the demonstrated exit with
`GO PHASE 07`; trading edge remains not demonstrated because the evidence is
`EXPLICIT_OOS_SINGLE_CASE_INSUFFICIENT_SAMPLE`. Canonical status remains:
`TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE`.

### Phase 08 — Controlled AI Tools & Durable Research Agents — COMPLETE / GO

Phase 08 adds bounded read-only research tools, durable checkpointed tasks,
explicit budgets, adaptive gap finding, critic/reflection, provenance-safe
memory and evaluation. It must preserve deterministic recommendation, risk and
execution gates. WP-08.01 through WP-08.08 are implemented and internally
verified; the exact Phase-08 exit condition is demonstrated by
`internal/modules/harness/evaluation_test.go`. External GPT-5.6 Sol review
accepted Phase 08 as COMPLETE / GO. The inability to run Go race detection
because `gcc` is unavailable remains a NON-BLOCKING ENVIRONMENTAL LIMITATION.

### Phase 09 — Portfolio Intelligence & Deterministic Risk — COMPLETE / GO

Phase 09 evaluates Phase-06 recommendations against canonical portfolio state
and explicit deterministic risk policy. It does not grant approval or
execution authority. WP-09.01 through WP-09.07 are implemented and internally
verified. The exact exit condition is demonstrated by
`internal/modules/portfoliorisk/phase09_exit_test.go` and was accepted by
external GPT-5.6 Sol as `GO PHASE 09`. The approved migration-number collision
remediation retained historical migrations unchanged, moved the Phase-09
migrations to `000059` and `000060`, and added complete-stream registry
validation. `TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE` remains the
scientific status. The approved migration-number collision remediation is
recorded in `Docs/Jax-Roadmap-v2/ROADMAP-DECISION-LOG.md`. Phase 10 is
`COMPLETE / CONDITIONAL GO`; Phase 11 is `COMPLETE / CONDITIONAL GO` with race
verification outstanding. Phase 12 is not authorised for WP implementation
because its dataset prerequisite is not satisfied. HYP-EVENT-001A is authorised
only for bounded historical-data readiness and baseline research design.

### Phase 11 — High-Fidelity Paper Trading — COMPLETE / CONDITIONAL GO

Phase 11 provides an isolated deterministic paper venue, versioned execution
costs, partial-fill and session semantics, event-derived paper ledger,
reconciliation, outcome attribution and a six-month soak protocol. The exact
capability gate is demonstrated by
`internal/modules/papertrading/phase11_exit_test.go` using accelerated frozen
fixtures. This is not real elapsed forward-paper evidence and does not prove
trading edge. Phase 10 remains **COMPLETE / CONDITIONAL GO** with
`RACE DETECTOR NOT VERIFIED — ENVIRONMENTAL LIMITATION`; race detection is
still unavailable in the current environment. Actual forward-paper evidence
remains 0 days / 0 orders. Phase 12 prerequisites are not satisfied and Phase
12 is not authorised.

### Phase 12 Readiness — Historical Data & Research Hypothesis — NOT READY

Phase-07 GO is satisfied and **HYP-EVENT-001A** is authorised for data readiness
and baseline research design only. The available historical data is still
**INSUFFICIENT** for leakage-safe advanced quantitative research. No paid data
has been purchased and WP-12.01 has not started. The bounded readiness
assessment is recorded in
`Docs/Jax-Roadmap-v2/12-advanced-quant-research/PHASE-12-READINESS-DECISION-
PACK.md`; the resulting handover is in
`Docs/Jax-Roadmap-v2/12-advanced-quant-research/HYP-EVENT-001A-HISTORICAL-
DATA-READINESS-HANDOVER.md`. Phase 13 is not started.

### Later roadmap — planned capability progression

The detailed package material is retained in `Docs/Jax-Roadmap-v2/`. These are
future stages, not current authorization to implement them.

| Stage | Intended outcome | Main promotion concern |
| --- | --- | --- |
| Phase 04 — Corroborated Event Intelligence | Reuse the World Monitor boundary with canonical evidence, source triangulation, event clustering and event/market correlation. | Existing event ingestion is not automatic acceptance of the new evidence architecture. |
| Phase 05 — Deterministic Quant Core | Produce reproducible returns, volatility, liquidity, correlation, benchmark and risk primitives. | Deterministic calculations and versioned inputs; no model-made facts. |
| Phase 06 — Evidence Packet, Research and Recommendations | Build evidence packets; combine event, market, company and macro evidence; represent bull/bear cases, contradictions, unknowns, freshness and eligibility; provide calibrated decision support. | `ALLOW`, `REVIEW` and `ABSTAIN` guardrails must measure calibration, uncertainty, downside/tail risk, evidence quality, regime familiarity and distribution shift. |
| Phase 07 — Evaluation, Replay, Backtesting and Falsification | Evaluate frozen recommendations through historical replay and realistic backtests. | Development/train vs holdout separation where applicable, out-of-sample and walk-forward testing, frozen configuration, predicted-vs-observed comparison, costs/slippage, placebo dates, unrelated assets, alternative periods and randomized/null signals. Apparent edge must face attempts to disprove it. |
| Optional Shadow Mode — Forward Market Observation | Only after strategy validation, observe live markets and compare frozen recommendations with actual movement while keeping order/execution authority disabled. | Shadow results are prospective evidence, not proof by themselves; no orders, fills, approvals or automatic promotion. |
| Phase 08 — Controlled AI Tools and Durable Research Agents | Add bounded read-only tools, durable research checkpoints, context/cost controls and safe model routing where justified. | AI must not bypass deterministic evidence, eligibility, risk or execution gates. |
| Phase 09 — Portfolio Intelligence and Deterministic Risk | Add canonical portfolio state, exposure/concentration/correlation, position proposals, budgets and stress/tail-risk analysis. | Deterministic risk and explicit unknowns; no leverage expansion by roadmap implication. |
| Phase 10 — Workflow, Human Approval and Operational Safety | Harden recommendation, review, paper-intent and audit state machines with explicit confirmation, kill switches, circuit breakers and observability. | Human approval remains explicit and record-only until a later gate says otherwise. |
| Phase 11 — High-Fidelity Paper Trading and Forward Validation | Implement or harden a paper venue with realistic fees, spread, slippage, latency, fills, market hours, account/position ledger, reconciliation and long-duration soak. | Separate `paper workflow implemented` from `paper trading evidence sufficient for promotion`; require substantial forward evidence and performance monitoring. |
| Phase 12 — Advanced Quant Research (optional) | Explore factors, model selection, ensembles, drift and rolling research only where statistically justified. | Experiment registry, rejection criteria, leakage-safe protocols and reproducible evidence. |
| Phase 13 — Optional Live Execution | Consider a separately gated, human-approved live capability only after all prior evidence and safety gates pass. | Live is optional, not the assumed destination. Any constrained automation requires a later explicit decision and demonstrated guardrail performance. |

No later phase is accepted merely because related code already exists. Each
phase requires its own package evidence, demonstrated exit condition and
external technical-lead GO.

## Existing capabilities awaiting later integration/revalidation

Jax already contains higher-level components from earlier implementation
sequences. They are retained and may be reused, hardened, adapted or replaced;
they are not automatically accepted into the current architecture programme.

| Area | Current repository-derived status |
| --- | --- |
| Decision Core | **EXISTING — IMPLEMENTED / TESTED; REVALIDATION / PROMOTION PENDING** against canonical contracts and current evidence providers |
| Event Intelligence | **EXISTING — IMPLEMENTED / TESTED; REVALIDATION / PROMOTION PENDING**; genuine World Monitor ingestion has bounded `PROVEN` evidence, but this is not proof of the full future intelligence phase |
| Swing Brain | **EXISTING — IMPLEMENTED / TESTED; REVALIDATION / PROMOTION PENDING**; swing-first remains the current product direction |
| Risk Veto / risk structures | **EXISTING — IMPLEMENTED / TESTED; REVALIDATION / PROMOTION PENDING**; this is not the future portfolio-risk phase or live authority |
| Research / backtest evidence structures | **EXISTING — IMPLEMENTED / TESTED; VALIDATION / PROMOTION PENDING**; structures and checks do not by themselves prove strategy edge or profitability |
| Paper approval / paper workflow | **EXISTING — IMPLEMENTED / TESTED; FORWARD EVIDENCE PENDING**; approval and ticket/read-model paths exist, but code existence is not sufficient paper-trading evidence |
| Review / replay / feedback / operator workflows | **EXISTING — IMPLEMENTED / TESTED** in bounded areas; genuine event replay is `PROVEN` as deterministic system behaviour, not as threshold, predictive or profitability validation |

See `Docs/CAPABILITY_MATRIX.md` for capability-level evidence and
`Docs/STATUS.md` for the older implementation-status snapshot. Neither changes
the active roadmap sequence in this document.

## Promotion gates

```text
trusted evidence
→ research and judgment
→ validated strategy
→ historical replay / backtest / falsification
→ forward paper validation
→ demonstrated edge and calibrated guardrails
→ human-approved live (optional)
→ constrained automation only if later explicitly justified
```

Before any autonomous live trading or live capital is considered, validation
must include realistic transaction costs/slippage, holdout and out-of-sample
discipline, walk-forward testing, frozen strategy/model configuration for
forward validation,
predicted-versus-observed monitoring, falsification/null tests, and a measured
decision guardrail that can `ALLOW`, `REVIEW` or `ABSTAIN` under uncertainty and
distribution shift. Paper trading must provide substantial forward evidence;
an implemented simulator or approval workflow is not enough. Optional Shadow
Mode may contribute forward evidence only after the historical gates and remains
non-executing.

## Safety state

Jax remains in a non-live, paper-safe/research state. The current safety contract
is unchanged:

- `ALLOW_LIVE_TRADING=false`.
- Broker/execution authority is disabled; the execution worker is disabled in
  the current restricted state.
- `BROKER_EXECUTION_ALLOWED=false` and recommendation execution authority is
  `NONE` unless a later explicit gate changes the state.
- Maximum leverage remains restricted to 1x; no leverage expansion is implied.
- No automatic creation or mutation of approvals, execution instructions,
  order intents, orders, trades or fills is authorized by this roadmap.
- Human approval remains mandatory for any paper workflow that is later enabled.

Live trading is **not currently authorized**. This document does not change
runtime settings, broker behaviour, trading rules, strategies or model selection.

## Supporting and historical roadmap sources

- **ACTIVE / AUTHORITATIVE:** this file, `Docs/ROADMAP.md`.
- **SUPPORTING:** `Docs/Jax-Roadmap-v2/`, which contains the detailed current
  architecture package briefs, gates, decision log and evidence navigation;
  its status pages support this roadmap and do not outrank it.
- **SUPPORTING / HISTORICAL:** `Docs/plans/jax-trading-roadmap-pack/`, whose
  candidate/risk/paper concepts remain useful but whose old phase sequence is
  superseded for current development. Its README and roadmap now direct readers
  here.
- **SUPPORTING:** `Docs/JAX_PRODUCT_CHARTER.md`, `Docs/CAPABILITY_MATRIX.md`,
  phase contracts and accepted evidence under `Docs/evidence/`.
- **HISTORICAL / ARCHIVED:** `Docs/archive/` and completed plan packs. They
  explain existing implementation history and must not be used as current
  authorization for live trading or autonomous execution.

The current position is **Phase 12 historical data readiness — IN PROGRESS**
after external `CONDITIONAL GO PHASE 11`. HYP-EVENT-001A is authorised for data
readiness and baseline research design only. Phase 10 remains complete/
conditional GO with its race-detector condition open because the current
environment lacks cgo/gcc. Phase 11 remains paper-only; actual forward-paper
evidence is 0 days / 0 orders. Phase 12 implementation is not authorised and
has not started.
