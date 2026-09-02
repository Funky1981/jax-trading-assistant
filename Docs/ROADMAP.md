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
| Phase 00 — Event / Issuer Foundation | ✅ Accepted / complete |
| Phase 01 — Canonical Contracts / Provenance / Replay | ✅ Accepted / complete |
| Phase 02 — Provider / Data Platform | ✅ Accepted / complete, including durable raw storage closure |
| Phase 03 — Core Financial Evidence | **IN PROGRESS** |
| Current accepted package | WP-03.03 — FRED / ALFRED macro observations + vintages (**COMPLETE / GO**) |
| Next package | **WP-03.04 — Economic Release / Calendar Ingestion (NEXT; not started)** |

The current package context is the accepted `capability-reset` sequence at
repository HEAD `952c2eec92a05c728e4ec6c74da6c09886ae18e6`. Package acceptance is
recorded in the relevant evidence and review handovers. The next package must be
authorized separately; this reconciliation does not begin WP-03.04.

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

### Phase 03 — Core Financial Evidence — IN PROGRESS

Phase 03 adds trustworthy market, company and macro evidence to the accepted
platform. Its phase exit requires a source-linked evidence packet for a
representative US equity/ETF using real market, company and macro evidence
without relying on model memory.

| Package | Status | Evidence / closure |
| --- | --- | --- |
| WP-03.01 — Market price / OHLCV | **COMPLETE / GO** | `Docs/evidence/WP-03.01-MARKET-PRICE-OHLCV-PROVIDER-HARDENING.md`; timestamp/freshness closure in `Docs/evidence/WP-03.01A-MARKET-EVIDENCE-CLOSURE.md` |
| WP-03.02 — SEC / EDGAR / XBRL | **COMPLETE / GO** | `Docs/evidence/WP-03.02-SEC-EDGAR-XBRL-EVIDENCE.md`; SEC temporal-semantics closure is included in its accepted result |
| WP-03.03 — FRED / ALFRED macro observations + vintages | **COMPLETE / GO** | `Docs/evidence/WP-03.03-FRED-ALFRED-MACRO-EVIDENCE.md`; point-in-time/vintage leakage protection and macro-evidence closure are included |
| WP-03.04 — Economic release / calendar ingestion | **NEXT — NOT STARTED** | `Docs/Jax-Roadmap-v2/03-core-financial-evidence/WP-03.04-economic-release-calendar-ingestion.md` |
| WP-03.05 — Treasury / EIA / CBOE / CFTC source evaluation and first approved integrations | **PLANNED** | `Docs/Jax-Roadmap-v2/03-core-financial-evidence/WP-03.05-treasury-eia-cboe-cftc-source-evaluation-and-first-approved-integrations.md` |
| WP-03.06 — Evidence-quality / cross-source checks | **PLANNED** | `Docs/Jax-Roadmap-v2/03-core-financial-evidence/WP-03.06-evidence-quality-cross-source-checks.md` |

Corrective packages such as WP-03.01A are closure history under their parent
package, not new permanent roadmap phases.

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
phase requires its own package reviews, evidence and technical-lead GO.

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

The next roadmap package is **WP-03.04 — Economic Release / Calendar
Ingestion**. Do not begin it as part of this documentation reconciliation.
