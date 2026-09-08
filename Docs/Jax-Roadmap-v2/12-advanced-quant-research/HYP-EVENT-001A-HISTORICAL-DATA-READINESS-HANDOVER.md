# HYP-EVENT-001A Historical Data Readiness Handover

**Date:** 2026-09-08  
**Repository:** `C:\Projects\Jax\jax-trading-assistant`  
**Branch:** `capability-reset`  
**Starting HEAD:** `06cd57e16f4d0057a9d3c24723fc51bc916dc9dd`
**Working tree at start:** clean
**Upstream:** `origin/capability-reset`; 107 ahead / 0 behind

## Decision and authorization boundary

`HYP-EVENT-001A — AUTHORISED FOR DATA READINESS AND BASELINE RESEARCH DESIGN`
is recorded. The hypothesis remains research-only. Phase 12 implementation,
WP-12.01, Qlib, RD-Agent, ML training, recommendation changes and Phase 13
were not started and remain unauthorized.

The bounded historical-data gate is now:

**SUFFICIENT CLEAN HISTORICAL DATA — READY FOR EXTERNAL PHASE-12 AUTHORISATION**

This is readiness for the initial event-study research only. It is not Phase
12 authorization, recommendation evidence, profitability evidence or a trading
edge claim.

## Registered hypothesis

**ID:** `HYP-EVENT-001A`  
**Title:** SEC evidence-backed issuer-event reaction

**Research question:** For timestamped material SEC 8-K events that Jax can
resolve to an issuer and classify directionally using only information
available at the time, does subsequent benchmark-relative return move more
consistently in the predicted direction when evidence quality is stronger?

The initial event family is exactly `FORM 8-K`; amendments (`/A`) and duplicate
semantics are excluded. Event records retain CIK, accession, form, filing date,
SEC acceptance timestamp, filing/issuer identity, raw evidence identity and
retrieval timestamp. The acceptance timestamp is retained as the available
public-availability proxy; acquisition time is not event time.

## Exact frozen research design

- Study period: `2016-01-01` through `2025-12-31`; partial 2026 excluded.
- Universe: frozen 61-symbol US-listed common-equity seed plus SPY benchmark;
  event-defined inclusion with explicit current-ticker survivorship limitation.
- Liquidity rule: prior close at least USD 5.00 and trailing 60-session
  average dollar volume at least USD 10,000,000.
- Entry: next regular US equity-session open strictly after event availability.
- Primary horizon: 5 trading days.
- Secondary diagnostics: 1 and 3 trading days.
- Exit: close of the relevant future session; no same-bar or unknowable close.
- Benchmark: SPY over identical timestamps.
- Baseline A: null/no systematic benchmark-relative effect.
- Baseline B: direction-only event assessment without evidence-quality
  conditioning.
- Direction and evidence quality must be versioned and knowable at event time;
  no return-derived labels were generated in this readiness stage.
- Phase-11 cost-model identity must be frozen before outcome evaluation.
- Required floors before exposing final-holdout outcomes: 300 qualifying
  issuer-events, 50 issuers, maximum 5% per issuer, and 60 final-holdout
  observations.
- Frozen partitions: development 2016–2021, validation 2022–2023, OOS 2024,
  sealed final holdout 2025.
- Final-holdout performance was not calculated or displayed.

## Secret/environment configuration

The repository-established root `.env` convention was used. `start.ps1` and
Compose expect that file; no second env file was created and the file was not
modified. Values were loaded only in memory for the qualification process.

| Setting | Status |
|---|---|
| `ALPACA_API_KEY` | PRESENT |
| `ALPACA_API_SECRET` | PRESENT |
| `SEC_USER_AGENT` | PRESENT |
| `SEC_CONTACT` | PRESENT |

No secret value was printed, logged, persisted, included in artifacts or
committed.

## Alpaca entitlement qualification

**PASS — Level 1 and bounded acquisition.** Using the existing configured
account and the established in-memory loading path:

- Representative symbols AAPL, MSFT, AMZN, JPM, XOM, NVDA and SPY returned
  January 2016 daily SIP/raw bars with successful pagination and rate-limit
  metadata.
- AAPL and SPY raw, split, dividend and all-adjustment requests succeeded.
- The bounded panel acquired 2016–2025 daily data for the frozen seed and SPY,
  with SIP feed identity and raw/all adjustment variants.
- No paid Alpaca plan or new credential was introduced.

The local/private dataset is at
`data/datasets/hyp-event-001a/dataset-2016-2025-sip-sec-v1/` and is ignored by
Git. Raw provider payloads are not redistributed. IEX-only history would not
be sufficient for a promotable result.

## SEC qualification and acquisition

**PASS — Level 1 and bounded acquisition.** The existing bounded `libs/sec`
EDGAR integration was used. A seven-issuer qualification sample returned
730 unique exact 8-K records across 2016–2025 with acceptance timestamps.
The bounded acquisition used official SEC submissions/history data, exact
8-K filtering, no amendments, accession deduplication and retained raw/source
provenance identities. The SEC client identity was loaded in memory from the
existing local convention; its values are not reported.

## Temporal cross-check

Three real 2024 AAPL 8-K records were cross-checked against Alpaca SIP daily
bars. Each had a parseable UTC acceptance timestamp, a next-session bar on a
strictly later session date, and a valid entry reference. No same-day or
same-bar fill was used.

## Dataset identity and coverage

**Dataset ID:** `hyp-event-001a-sec-8k-alpaca-sip-2016-2025-v1`
**Content-manifest SHA-256:** `db2b454793e50f28c34c9c2a7d91f798741936528752de7bd27cb52072a4864d`
**Local dataset size:** approximately 136 MB; 418 files
**Feed:** Alpaca SIP
**Benchmark:** SPY
**Candidate events:** 1,166
**Excluded by frozen liquidity/coverage qualification:** 107
**Qualifying deduplicated events:** 1,059
**Distinct issuers:** 60
**Maximum issuer contribution:** 19 events / approximately 1.79%
**Raw market rows:** 154,674
**All-adjusted market rows:** 154,674
**Selected-event next-session coverage:** PASS for all selected events
**Acceptance timestamp completeness:** PASS for all selected events

Partition counts in the local normalized event panel are:

| Partition | Dates | Qualifying events | Status |
|---|---|---:|---|
| Development | 2016-01-01 – 2021-12-31 | 583 | available for later research |
| Validation | 2022-01-01 – 2023-12-31 | 236 | available for later research |
| OOS | 2024-01-01 – 2024-12-31 | 120 | available for later research |
| Final sealed holdout | 2025-01-01 – 2025-12-31 | 120 | **SEALED; no outcomes calculated** |

The versioned validation artifact is
`data/datasets/hyp-event-001a/dataset-2016-2025-sip-sec-v1/normalized/validation.json`.

## Survivorship, corporate actions and quality

- **Survivorship/delisted:** Limited. The seed is current-ticker based; point-in-
  time inactive, renamed and delisted coverage was not demonstrated. This is
  acceptable for initial research only, not promotion or an edge claim.
- **Corporate actions:** Raw and all-adjustment requests were acquired. Field-
  level reconciliation remains required before any promotion decision.
- **Event duplicates/amendments:** Exact 8-K and accession deduplication was
  applied; `/A` amendments were excluded.
- **Issuer mapping:** SEC CIK identity was retained; current-ticker mapping
  and historical instrument limitations remain explicit.
- **Bars:** Selected event coverage, market-row validity and benchmark coverage
  passed the bounded validation. No outcome or direction label was generated.
- **Calendar/timing:** Next-session selection was cross-checked on real AAPL
  examples. The current adapter does not provide a separately authoritative
  public-availability field beyond the retained SEC acceptance timestamp.
- **Readiness floors:** All four floors passed: 1,059 events, 60 issuers,
  approximately 1.79% maximum issuer share, and 120 final-holdout events.

## Licensing, cost and credentials

- Spend: **0**; no paid source, subscription or account was added.
- Credentials: existing local configuration only; no new credential requested;
  values remain private and are not included here.
- Data use: local/private research use; raw Alpaca data is not redistributed.
  SEC source/provenance records are retained. Publication or commercial use
  requires a separate rights review against the applicable provider terms.

## Verification

Passed or recorded:

- Alpaca and SEC live qualification contract tests with configured local
  secrets loaded in memory.
- SEC/market-data package tests and full `go test ./... -count=1`.
- `go vet ./...`.
- Dataset validation, event coverage, temporal/provenance and provider checks.
- Reproducible dataset content-manifest validation.
- Roadmap/manifest validation and `git diff --check`.

No migration, runtime, recommendation or execution code was changed. The only
repository data-readiness changes are documentation and the Git ignore rule for
the private local dataset. The final-holdout performance remains sealed.

## Adversarial data review

The bounded review attacked future-derived labels, acquisition-vs-event time,
same-day look-ahead, calendar selection, amendment/duplicate handling, issuer
resolution, symbol changes, survivorship, delisted coverage, SIP/IEX identity,
invalid bars, holdout access, outcome-derived thresholds and license omission.

**Data-readiness blocking findings remaining: 0**
**Adversarial data review: PASS**

Non-blocking limitations remain explicitly recorded: current-ticker
survivorship, incomplete demonstrated delisted/rename reconstruction,
acceptance timestamp as the available public-availability proxy, and pending
field-level corporate-action reconciliation before promotion.

## Race-detector status

`RACE DETECTOR VERIFICATION OUTSTANDING` remains unchanged. The existing
environment lacks the required C/cgo toolchain; no compiler or system tooling
was installed. This does not block this data-readiness assessment.

## Scientific and safety status

`TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE` remains preserved.
No return, effect size, OOS performance, significance or profitability claim
was produced. Actual forward-paper evidence remains `0 DAYS / 0 ORDERS`;
`SOAK INFRASTRUCTURE DEMONSTRATED` remains separate.

`ALLOW_LIVE_TRADING=false`, `BROKER_EXECUTION_ALLOWED=false`, the live worker
and broker execution remain disabled, and no broker order, paper order, trade,
fill or portfolio mutation was created by this task.

## Readiness decision

**SUFFICIENT CLEAN HISTORICAL DATA — READY FOR EXTERNAL PHASE-12 AUTHORISATION**

This result authorizes no implementation by itself. External technical-lead
review is required before Phase 12 begins. Until then, HYP-EVENT-001A remains
research-only; WP-12.01 and all other Phase-12 implementation remain not
started. Phase 13 remains not started.
