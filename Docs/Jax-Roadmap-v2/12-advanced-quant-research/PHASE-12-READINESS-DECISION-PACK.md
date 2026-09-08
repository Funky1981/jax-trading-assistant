# Phase 12 Readiness Decision Pack

**Date:** 2026-09-08  
**Repository:** `C:\Projects\Jax\jax-trading-assistant`  
**Branch:** `capability-reset`  
**Assessment HEAD:** `5ebb3a9c6409cd354d5859fcb81d788e40c637be`  
**Working tree at assessment start:** clean  
**Upstream:** `origin/capability-reset`; 0 behind / 105 ahead

## Decision

**PHASE 12 NOT READY.** This document records the bounded readiness stage
`PHASE 12 READINESS — HISTORICAL DATA & RESEARCH HYPOTHESIS`. It is not Phase-12
implementation and does not authorize WP-12.01, model training, a new runtime,
Qlib, RD-Agent, a feature store, a provider integration or a data purchase.

| Phase-12 prerequisite | Result | Evidence |
|---|---|---|
| Phase-07 GO | Satisfied | Phase-07 exit/evaluation evidence; scientific status remains `EXPLICIT_OOS_SINGLE_CASE_INSUFFICIENT_SAMPLE`. |
| Sufficient clean historical data | **INSUFFICIENT** | The catalog contains only four narrow fixtures: AAPL daily 22 rows, AAPL 1-minute 2,729 rows, SPY daily 120 rows and SPY 1-minute 500 rows. |
| Authorised explicit falsifiable hypothesis | **NOT FOUND** | Only a blank template, documentation examples and test fixtures were found; none is an accepted Phase-12 decision. |

The scientific conclusion remains **TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT
SAMPLE**. Actual forward-paper evidence remains **0 DAYS / 0 ORDERS**. Phase-10
and Phase-11 race-detector verification remains **OUTSTANDING** because the
available environment has no usable C/cgo compiler.

## Historical-data standard

The standard below is scoped to the recommended first experiment, not to every
future research question. A requirement becomes blocking when the selected
hypothesis depends on it; unknown is never treated as available.

### Required for the first experiment

- A pre-registered, liquid US equity or ETF universe. A practical starting
  range is 100–300 instruments, or a smaller event-defined universe only if the
  event count, issuer count and regime coverage are sufficient and frozen before
  inspection of outcomes. Thousands of instruments are not required immediately.
- Approximately ten years of daily raw and adjusted OHLCV, or a shorter period
  only when a pre-registered event-count and regime-coverage justification is
  accepted. The dataset must include enough independent events to estimate an
  effect rather than demonstrate one anecdote; an indicative technical-lead
  review target is at least 100 independent events across at least 20 issuers
  and two materially different regimes. These are proposed freeze criteria,
  not an authorization or a substitute for power analysis.
- Exchange-local session timestamps with an unambiguous UTC representation,
  trading-calendar identity, publication/availability timestamp, and explicit
  next-eligible-bar execution convention.
- Duplicate, gap, invalid-price, stale-data and outlier checks with an explicit
  missing-data policy. No silent interpolation or zero-filling.
- Separate raw and adjusted price semantics, split/dividend/corporate-action
  identity, adjustment effective time and reproducible adjustment version.
- A universe policy that addresses survivorship. Point-in-time membership and
  delisted instruments are required when the claim depends on a historical
  universe; a fixed current universe must instead be labelled as a limitation.
- Event/evidence records containing stable issuer and instrument identities,
  event type, source/provider, publication time, acquisition time, source
  vintage/revision state, raw-payload identity and corroboration/contradiction
  status.
- A benchmark and, where used, sector or peer benchmark with the same timing
  and quality controls.
- Dataset identity, content hashes, schema/version, ingestion configuration,
  source terms and retention/redistribution decision. The exact frozen dataset
  must be replayable.
- A pre-registered train/development, validation, OOS and untouched final
  holdout split. The final holdout must not be used for feature selection,
  tuning or repeated result inspection.
- Phase-07/11 execution-cost assumptions applied consistently; a zero-cost
  result may exist only as a labelled diagnostic comparison.

### Desirable for later research

Point-in-time broad and delisted universes; longer and higher-frequency history;
as-reported SEC fundamentals with filing availability timestamps; analyst
expectations; ALFRED macro vintages and complete release calendars; FX and
multi-currency valuation; richer corporate actions; borrow/shortability; quote
and order-book history; cross-vendor corroboration; formal legal review of
redistribution; and larger samples for cross-sectional ML. These are not all
required for the first event-study experiment.

Fundamental or macro-feature hypotheses promote several desirable items to
required items: as-reported values, initial publication time, revision/vintage
identity and a non-leaking expectation or baseline are mandatory for those
hypotheses.

## Current Jax data inventory

| Capability/source | Current repository status | What it can support | Readiness limitation |
|---|---|---|---|
| Dataset catalog and fixtures | Current; `data/datasets/catalog.json` | Reproducible schema, dates, counts and hashes for four OHLCV fixtures | Not a research panel: two symbols, short windows, no demonstrated delisted universe, corporate-action, licensing or point-in-time controls. |
| Alpaca | Current provider and hardened boundary under `libs/marketdata` | US equity/ETF historical bars where an authorised key, entitlement and permitted feed are available; official API documents daily and intraday bar timeframes ([historical bars](https://docs.alpaca.markets/us/reference/stockbarsingle-1)) | Credentials/entitlements are not assumed; feed completeness, corporate actions, survivorship and backtest/redistribution terms are not established for Phase 12. |
| IB bridge/paper data | Current development/paper path | Paper mechanics and small development fixtures | The `ib-bridge-paper` fixtures do not demonstrate a long, clean, point-in-time research dataset; account entitlements and history terms remain unverified. |
| Financial Datasets | Current hardened daily-EOD provider boundary | Potential US price/fundamental data through a keyed provider | Requires account/API key and potentially paid access. Its public documentation describes limited historical depth by endpoint/plan; no purchase or credential is authorized ([historical prices](https://docs.financialdatasets.ai/api/prices/historical), [pricing](https://www.financialdatasets.ai/pricing)). |
| SEC EDGAR/XBRL | Current `libs/sec` provider, normalizers and provenance | Filing/company facts, CIK identity and as-filed evidence; SEC’s public APIs expose submissions and company facts ([SEC EDGAR APIs](https://www.sec.gov/search-filings/edgar-application-programming-interfaces)) | Does not provide the required broad price panel, complete point-in-time tradable universe or guaranteed event-linked market reaction history. Restatement/as-reported and filing-time validation still need a frozen acquisition design. |
| FRED/ALFRED | Current `libs/fred` and macro evidence contracts | Macro observations, real-time periods/vintages and revision-aware evidence; ALFRED real-time periods are designed to represent what was known at a historical time ([real-time periods](https://fred.stlouisfed.org/docs/api/fred/realtime_period.html), [observations/vintages](https://fred.stlouisfed.org/docs/api/fred/series_observations.html)) | No complete experiment panel; API-key/terms and release-timestamp acquisition remain unresolved. API terms require registration and impose usage restrictions ([terms](https://fred.stlouisfed.org/docs/api/terms_of_use.html)). |
| Economic calendar/release evidence | Current `libs/calendar`, `libs/releaseevidence`, `libs/macroevidence` | Release metadata, source and acquisition provenance | Coverage and publication-time completeness are not demonstrated for a Phase-12 frozen panel; it cannot replace prices or event-universe construction. |
| Treasury, Cboe, CFTC, EIA | Current/partial supporting providers and evaluations | Supporting rates, VIX and public macro/market context | Narrow coverage; EIA/key and CFTC integration constraints remain; Cboe terms/redistribution and release-time semantics require review. Not sufficient as the primary dataset. |
| World Monitor | Current event intelligence/provenance path | Candidate event/evidence features, source corroboration and issuer/event context | Not a long clean market-history source and not sufficient alone for quantitative OOS research. |
| Replay/evaluation/walk-forward | Current `libs/replay`, `libs/walkforward`, `internal/modules/evaluation` | Frozen replay, OOS discipline, metrics and evidence-linked evaluation | Provides evaluation mechanics, not missing historical observations or authorization of a hypothesis. |
| Raw payload, dataset snapshots, cache/persistence | Current `internal/rawpayloadstore`, `libs/dataset`, Postgres migrations | Immutable acquisition references, dataset identity and durable storage | Storage does not establish source quality, licensing, point-in-time semantics or adequate coverage. |

Repository evidence for the provider constraints is consolidated in
`Docs/evidence/PHASE-03-FREE-MARKET-DATA-CLOSURE.md`,
`Docs/evidence/WP-03.01-MARKET-PRICE-OHLCV-PROVIDER-HARDENING.md`,
`Docs/evidence/WP-03.02-SEC-EDGAR-XBRL-EVIDENCE.md`,
`Docs/evidence/WP-03.03-FRED-ALFRED-MACRO-EVIDENCE.md`,
`Docs/evidence/WP-03.04-ECONOMIC-RELEASE-CALENDAR-EVIDENCE.md`,
`Docs/evidence/WP-03.05-TREASURY-EIA-CBOE-CFTC-SOURCE-EVALUATION.md`, and
`Docs/Jax-Roadmap-v2/references/DATA-SOURCE-STRATEGY.md`.

## Data-gap matrix

| Requirement | Current source | Status | Gap | Blocking for first experiment? |
|---|---|---|---|---|
| Fixed, sufficiently broad universe | Four fixtures only | Available only as fixtures | No point-in-time membership/delistings or adequate breadth | Yes, unless an event-defined universe is explicitly frozen and earns adequate sample. |
| Long daily OHLCV | Alpaca/IB/Financial Datasets boundaries | Technically obtainable, not demonstrated | No assembled, validated panel or terms decision | Yes |
| Intraday event windows | Alpaca/IB boundaries | Partial | Current minute fixtures cover only a few days | Only for intraday hypotheses; not required for daily first experiment. |
| Session/timestamp/availability semantics | Provider contracts and validators | Partial | Event publication and next-eligible-bar semantics are not unified in a frozen dataset | Yes |
| Missingness/quality validation | Provider hardening and dataset schema | Partial | No panel-level quality report | Yes |
| Corporate actions | Provider-specific/unknown | Unknown | No complete reproducible adjustment/corporate-action policy for the panel | Yes |
| Delisted/survivorship controls | No demonstrated source | Missing | Historical universe membership and delisted coverage absent | Yes for universe claims; fixed-universe limitation must be explicit otherwise. |
| SEC as-filed fundamentals | `libs/sec` | Technically obtainable | No event-linked, as-reported historical panel | Yes for SEC/fundamental candidates; not required for a price-only event study. |
| Macro vintages/release times | `libs/fred`, calendar | Partial | Key/terms and complete release timestamp panel not established | Yes for macro candidates; no for price-only candidate. |
| Benchmarks/calendars/FX | Existing supporting paths | Partial/unknown | Not assembled and frozen with the selected dataset | Benchmark/calendar yes; FX only if multi-currency is used. |
| Reproducibility and versioning | Catalog hashes/raw storage | Available as mechanics | Source terms, config and panel validation still need a release record | Yes before any external research conclusion. |
| Train/validation/OOS/final holdout | Phase-07 contracts | Available as mechanics | No adequate dataset to populate immutable splits | Yes |
| Transaction costs | Phase-11 models | Available as mechanics | Must be selected and frozen for the hypothesis | Yes for performance claims; diagnostic zero-cost remains secondary. |
| Licensing/retention/redistribution | Source-specific docs | Unknown/partial | No completed source-by-source legal/terms decision | Yes before acquisition or publication. |

## Data-source options and cost decision

No provider was purchased, no account was created, no credential was requested,
and no large dataset was downloaded.

| Route | Coverage/quality potential | Operational/licensing implication | Cost category | Decision status |
|---|---|---|---|---|
| Existing authorized Alpaca access | Daily/intraday US bars; can be suitable for a bounded first price panel if entitlements and history are verified | Key management, rate limits, feed/adjustment semantics and permitted research/redistribution terms require explicit review | FREE or LOW, depending on current entitlement | Preferred investigation route; not yet authorized for acquisition. |
| Public SEC EDGAR + existing Jax event/evidence paths | Strong filing provenance and as-filed timing; no price panel | Must respect SEC request policy and build a point-in-time event dataset; still needs market prices and universe controls | FREE | Suitable supporting route, not sufficient alone. |
| Public FRED/ALFRED + calendar | Revision-aware macro series and vintage queries | API key/terms and precise release-time handling; not equity OHLCV | FREE or LOW operationally | Supporting route for a macro hypothesis only; no key request authorized. |
| Public Treasury/Cboe data | Narrow official rates/VIX inputs | Terms, timestamps and redistribution restrictions vary; not a broad universe | FREE, with UNKNOWN terms burden | Supporting inputs only. |
| Financial Datasets | Potentially packaged US market/fundamental coverage and deeper history by plan | Account/API key, provider dependency and paid/licensing review; point-in-time quality still needs verification | MODERATE-HIGH / PAID | Not authorized. No purchase. |
| IB historical market data | Potentially useful if an existing account has entitlements | Session/account dependency, entitlement and retention terms; paper bridge is not proof of research suitability | UNKNOWN / MODERATE | Requires provider/account investigation; no new account or credential. |
| Professional survivorship-controlled vendors | Better delisted/corporate-action/fundamental coverage where licensed | Substantial cost, contracts, redistribution controls and integration burden | HIGH / PAID | Defer until a selected hypothesis demonstrates need; no purchase. |

The recommended first experiment appears possible in principle with current/free
source routes, but not with the data currently present. It requires verification
of existing authorized Alpaca access or another already-authorized/public route,
plus a completed licensing and point-in-time design. It does **not** justify
buying data at this stage. If those checks fail, the experiment is blocked until
an external technical-lead decision authorizes an acquisition route and spend.

## Universe-size decision

Thousands of securities are not an automatic Phase-12 prerequisite. A smaller
liquid universe can provide useful event-study evidence when the sample contains
independent events, multiple issuers, multiple regimes, and a pre-declared
selection policy. A single symbol, four fixtures, or one short market episode
cannot support a general claim.

For the recommended candidate, the smallest credible starting design is a
pre-registered 100–300 instrument liquid US equity/ETF universe, with an
event-defined fallback only if it produces the agreed independent-event and
issuer coverage. Generalisability, survivorship, transaction costs and event
dependence must be reported. The exact size, event-count threshold and power
analysis remain external decisions to freeze before implementation.

## Candidate hypotheses — not authorised

These are a shortlist for external selection. None is promoted into Phase-12
scope.

### HYP-EVENT-001A — Evidence-backed issuer-event reaction

**Question/rationale:** After a source-backed, issuer-resolved material event
with a pre-defined direction/surprise and corroboration quality, does the next
1/3/5 trading-day risk-adjusted excess return differ from the benchmark? The
economic rationale is delayed information absorption or underreaction.

**Target/horizon:** Forward excess return at 1, 3 and 5 trading days, using the
next eligible bar after the event becomes public; event outcome is measured
without using later evidence. **Universe:** pre-registered liquid US equities or
ETFs with stable issuer/instrument identity. **Features/evidence:** event type,
issuer, publication/availability time, source and raw identity, corroboration,
contradiction/unknown state, pre-event market reaction, volume/volatility and
benchmark/sector context.

**Baseline/metrics/costs:** Benchmark/no-signal and a simple event-direction
heuristic are mandatory baselines. Metrics include event count, mean/median
excess return, bootstrap interval, hit rate, drawdown, turnover, net result
after Phase-11 costs and calibration/coverage. Fees, spread, slippage and
latency assumptions are frozen before OOS evaluation.

**Data/minimum design:** Daily OHLCV and event/evidence records for roughly a
decade, with enough independent events across issuers and regimes to satisfy a
pre-registered power/sample rule; no intraday data unless the question is
changed. **Falsification:** shuffled event labels, placebo dates, source removal,
alternative event windows, regime splits, issuer clustering, cost stress and
contradiction/unknown exclusion sensitivity. **Reject if:** no OOS improvement
over baseline, effect disappears under costs or timing controls, sample is too
small/dependent, or evidence quality cannot be reproduced.

**Complexity/data today/risks:** Low-to-moderate deterministic event study first;
data today is insufficient. Main risks are publication-time leakage, duplicate
events, source overlap, survivorship and event dependence. Jax’s potential
advantage is issuer resolution plus provenance, contradiction and unknown
handling. This candidate is **not authorised**.

### HYP-SEC-001 — Point-in-time SEC filing surprise

**Question/rationale:** Does a point-in-time filing surprise, defined from
as-reported SEC facts available at filing time, predict forward excess return
after release? **Target/horizon:** 1, 5 and 20 trading-day excess return.
**Universe:** issuers with stable CIK-to-instrument mapping, including a policy
for delisted names. **Features:** accepted/filed timestamp, prior as-reported
facts, XBRL tags, change/surprise definition, source identity and market/sector
context.

**Baseline/metrics/costs:** Market/sector benchmark and a pre-declared simple
threshold rule; OOS excess return, confidence interval, coverage, turnover,
cost-adjusted result and stability by regime. **Data:** multi-year filings and
prices, as-reported facts, filing-time availability and historical universe.
**Falsification/rejection:** shuffled filing dates/labels, restatement-vs-as-filed
comparison, alternative tags/windows, issuer clustering, cost stress and no
improvement over the simple threshold/baseline. **Complexity:** moderate to
high. **Today:** SEC capability exists but no complete panel. **Risks:** filing
timestamp interpretation, restatements, missing expectations and survivorship.
Jax advantage: SEC provenance and issuer resolution. Not authorised.

### HYP-MACRO-001 — Vintage-safe macro surprise reaction

**Question/rationale:** Do surprises in selected macro releases, defined from
the initial release and a pre-declared expectation/baseline, produce reproducible
asset-specific excess returns? **Target/horizon:** daily or explicitly timed
release-window return in major ETFs, rates or FX proxies.

**Features/data:** ALFRED vintage, initial-release value, release timestamp,
calendar identity, benchmark and market session data. **Baseline/metrics/costs:**
unconditional event mean and a no-regime/simple directional rule; OOS event
return, interval, hit rate, regime stability and net cost-adjusted result.
**Falsification/rejection:** revised-vs-initial comparison, placebo release dates,
shuffled surprises, overlapping-release exclusion, regime splits and cost stress.
**Minimum data/complexity:** a long release history with precise timestamps and
enough independent events; moderate. **Today:** macro/vintage contracts exist,
but the keyed/timestamped panel and market history are not assembled. **Risks:**
revision/look-ahead, overlapping releases, timezone errors and low event count.
Not authorised.

### HYP-REACTION-001 — Evidence-conditioned continuation or reversal

**Question/rationale:** After the initial market reaction to an evidence-backed
event, does the next 1–5 day path show reproducible continuation or reversal
conditional on evidence quality and contradiction state? **Features:** event
reaction, gap, volume, volatility, source count, contradiction/unknowns,
issuer/sector and benchmark context. **Baseline:** naive continuation and naive
reversal rules fixed before evaluation. **Metrics/costs:** OOS path return,
drawdown, turnover, event coverage and net cost result.

**Data/falsification/rejection:** Same event-linked point-in-time price/evidence
panel as HYP-EVENT-001, with label shuffling, placebo dates, same-event
deduplication, timing perturbation, regime and cost tests. Reject if the result
depends on one event class, narrow window, a tiny issuer set or zero-cost
assumptions. Complexity is moderate; today’s data is insufficient. Jax’s
provenance/contradiction model is relevant. Not authorised.

Deferred future candidates — regime detection, GEX/options positioning, Auction
Market Theory, volume profile, order flow, session behaviour and dynamic market
structure — remain deferred and are not promoted by this pack.

## Technical recommendation — NOT AUTHORISED

The external technical lead has now selected and authorized **HYP-EVENT-001A**
for data readiness and baseline research design only. It is not authorization to
implement Phase 12. Begin with a deterministic event study and its meaningful
baselines only after the dataset gate passes; advanced ML is not warranted
unless it later demonstrates reproducible OOS value over those baselines.

This is preferred because it reuses Jax’s strongest differentiators (event
intelligence, issuer resolution, evidence provenance, contradiction/unknown
semantics and replay), requires less data than a full fundamental or intraday
ML system, is falsifiable even if rejected, and exposes the main leakage risks
clearly. It remains **NOT AUTHORISED**: no implementation, acquisition or
recommendation-logic change follows from this recommendation.

## Minimum viable research dataset

For HYP-EVENT-001A, subject to the authorized research contract and data gate:

- A frozen liquid US equity/ETF universe, initially proposed at 100–300 names,
  with explicit inclusion date, point-in-time membership or a clearly labelled
  fixed-universe limitation and a delisting policy.
- Approximately 2015–2025 daily OHLCV, raw and adjusted, with exchange calendar,
  UTC/session mapping, adjustment events and quality report. The final date
  range must be frozen before analysis, not selected after seeing results.
- Event/evidence records for the same period with stable issuer/instrument IDs,
  event type, publication/availability timestamp, acquisition timestamp, source
  identity, raw payload identity, corroboration, contradiction and revision state.
- SPY or another pre-declared broad benchmark and a sector/peer benchmark only
  where the universe mapping is trusted.
- Dataset/config/schema/version identities, hashes, validation results and
  license/retention record. Approximate record volume is the product of the
  frozen universe and sessions (roughly hundreds of thousands to low millions
  of daily rows for this range), plus event/evidence records; it must be measured
  after acquisition rather than invented.
- Validation: timestamp ordering, duplicate bars/events, gaps, invalid prices,
  corporate-action reconciliation, event-to-instrument resolution, publication
  time versus acquisition time, benchmark alignment, missingness, event
  independence and train/validation/OOS/holdout disjointness.

### Ideal later dataset

A licensed point-in-time, survivorship-controlled multi-asset dataset with
delisted names, complete corporate actions, as-reported fundamentals, SEC filing
timestamps, analyst expectations where licensed, ALFRED macro vintages, release
calendars, FX, intraday quotes/trades and reproducible costs/borrow/venue
semantics. It is not required or authorized now.

## Leakage, OOS and holdout strategy

Before any implementation, freeze dataset/version, universe, feature definitions,
knowability timestamps, target/horizon, benchmark, cost model, train/validation/
OOS dates, final holdout dates, metrics and rejection criteria. Fit transforms and
feature selection only within development/training windows. The final holdout is
sealed, accessed once for the registered final evaluation, and never used to
choose event classes, parameters, models or a narrative. All event joins use
availability time, not later acquisition or revised values. Every rerun receives
a new experiment identity; historical results are immutable.

## Relationship to forward paper trading

No forward-paper experiment starts in this readiness stage. If a later Phase-12
experiment is authorized and promoted, it receives a new hypothesis, feature,
dataset, model/recommendation-policy and configuration identity. Existing Phase-
11 paper results remain immutable. Future paper orders/results must retain the
exact generating configuration and must remain distinguishable from historical
policies. Phase-12 output remains research evidence until the accepted promotion
gate and Phase-06/09/10/11 controls are satisfied; it never receives approval or
execution authority directly.

## Race-condition check

The existing-environment attempt was:

```text
go test -race ./internal/modules/workflow ./internal/modules/papertrading -count=1
```

It failed to build at `runtime/cgo` because `gcc` is not present. Existing
inspection found no usable GCC/Clang or WSL distribution; `docker.exe` and
`docker-desktop` presence did not provide an already-authorized runnable build
environment. No toolchain or system package was installed. Status:
**RACE DETECTOR VERIFICATION OUTSTANDING**. This does not change the Phase-12
readiness result.

## Decisions required from the external technical lead/user

1. Select or reject one candidate hypothesis; do not treat this shortlist as
   authorization.
2. Approve the first-experiment universe, event definition, sample/power rule,
   date range, benchmark, horizon and cost model.
3. Decide whether existing authorized Alpaca/public routes are sufficient after
   entitlement, point-in-time, corporate-action, survivorship and licensing
   verification.
4. If they are not sufficient, decide whether to authorize a new paid/licensed
   dataset. No spend is requested or committed by this pack.
5. Decide the acceptable fixed-universe limitation versus requiring delisted and
   point-in-time membership data before the first experiment.
6. Resolve the outstanding Phase-10/11 race verification in an already-approved
   compatible environment before an unconditional Phase-11 GO or later live
   consideration.

## Verification and scope boundary

This change is documentation/governance only. No runtime code, provider code,
recommendation logic, ML infrastructure, model, dataset or migration was
changed. No Phase-12 package was started; WP-12.01 and Phase 13 remain not
started. Existing runtime verification remains governed by the previously
accepted Phase-11 evidence; this pack does not reclassify synthetic fixtures as
forward evidence or claim a trading edge.

## Recommended next roadmap action

Keep **Phase 12 NOT READY / NOT AUTHORISED** and continue the bounded readiness
stage until the two prerequisites are independently demonstrated:

- **SUFFICIENT CLEAN HISTORICAL DATA** for a selected, frozen hypothesis; and
- **AUTHORISED EXPLICIT RESEARCH HYPOTHESIS** with a falsifiable contract.

Only after both are accepted should external review authorize WP-12.01. Phase 13
remains **NOT STARTED**.
