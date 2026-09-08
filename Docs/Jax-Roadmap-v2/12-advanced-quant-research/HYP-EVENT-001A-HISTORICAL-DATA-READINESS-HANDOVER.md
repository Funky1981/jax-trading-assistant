# HYP-EVENT-001A Historical Data Readiness Handover

**Date:** 2026-09-08  
**Repository:** `C:\Projects\Jax\jax-trading-assistant`  
**Branch:** `capability-reset`  
**Starting HEAD:** `ea7f92b3af86695f68be36b0099f7bdad6691f07`  
**Working tree at start:** clean  
**Upstream:** `origin/capability-reset`; 106 ahead / 0 behind

## Decision and authorization boundary

`HYP-EVENT-001A — AUTHORISED FOR DATA READINESS AND BASELINE RESEARCH DESIGN`
is recorded. The hypothesis is research-only. Phase 12 implementation remains
not authorized: WP-12.01, Qlib, RD-Agent, ML training, recommendation changes
and Phase 13 were not started.

Readiness result:

**HISTORICAL DATA INSUFFICIENT — PHASE 12 REMAINS NOT READY**

The dataset gate did not pass because no authorized Alpaca account/feed was
configured and Jax’s SEC live adapter requires an operator-supplied automated
client identity that is not configured. Consequently no Level-1 live source
qualification or bounded historical acquisition was performed.

## Registered hypothesis

**ID:** `HYP-EVENT-001A`  
**Title:** SEC evidence-backed issuer-event reaction

**Research question:** For timestamped material SEC 8-K events that Jax can
resolve to an issuer and classify directionally using only information available
at the time, does subsequent benchmark-relative return move more consistently
in the predicted direction when evidence quality is stronger?

The initial event family is restricted to `FORM 8-K`. Amendments and duplicate
semantics are excluded unless explicitly modelled later. Required event identity
fields are CIK, accession number, form, filing date, SEC acceptance/publication
timestamp where available, filing identity, issuer identity, raw evidence
identity and retrieval timestamp. Public availability time is event time;
acquisition time is not event time.

## Frozen baseline research design

- Study period: `2016-01-01` through `2025-12-31`; partial 2026 is excluded.
- Primary horizon: 5 trading days.
- Secondary diagnostics: 1 and 3 trading days.
- Entry: next regular US equity-session open after SEC public availability.
- Exit: close of the relevant future trading session.
- Return: instrument return minus SPY return over identical timestamps, with
  predicted-direction signing where applicable.
- Benchmark: SPY only; no post-result sector benchmark is permitted.
- Baseline A: null/no systematic benchmark-relative effect.
- Baseline B: direction-only event assessment without evidence-quality
  conditioning.
- Primary test: evidence-quality conditioning must improve reproducibly over
  Baseline B at the registered 5-day horizon.
- Direction must be generated from versioned information available at event
  time. Future returns cannot create event labels.
- Evidence quality may use only already-defined or separately versioned Jax
  dimensions: source authority, provenance completeness, corroboration,
  contradiction/unknown state and issuer-resolution confidence.
- Phase-11 execution-cost model identity must be frozen; gross and
  cost-adjusted results must be reported separately.
- No final-holdout result was calculated or displayed.

Required falsification set: shuffled directional labels, constrained shuffled
event dates, placebo non-event dates, evidence-quality permutation, removal of
the strongest source category, issuer-cluster sensitivity, event-cluster and
deduplication sensitivity, an alternative predeclared liquidity filter,
transaction-cost stress, regime split and exclusion of top-contributing issuers.

Required readiness floors before exposing final-holdout outcomes are 300
deduplicated qualifying issuer-events, 50 distinct issuers, no issuer above 5%
of observations and 60 qualifying final-holdout observations. These are
data-readiness floors, not a claim of statistical power. The formal
development-only precision/power assessment must occur before OOS/holdout
evaluation.

Partitions are frozen as:

| Partition | Dates | Status |
|---|---|---|
| Development | 2016-01-01 – 2021-12-31 | Not populated |
| Validation | 2022-01-01 – 2023-12-31 | Not populated |
| OOS | 2024-01-01 – 2024-12-31 | Not populated |
| Final sealed holdout | 2025-01-01 – 2025-12-31 | **SEALED; not read** |

## Alpaca entitlement qualification

Status: **NOT RUN — REQUIRED CONFIGURATION ABSENT**.

The repository configuration has Alpaca disabled with empty API key and secret.
The process environment also has no `ALPACA_API_KEY`, `ALPACA_API_SECRET` or
feed configured. Therefore the required 5–10-symbol plus SPY qualification could
not legitimately test daily access, 2016 history, SIP availability, pagination,
adjustments, symbol changes or rate limits. No credentials were printed,
requested or created.

The existing hardened Alpaca path currently declares historical bars and bounded
pagination, but its normalizer requires raw/unadjusted bars and does not by
itself prove adjusted-bar, split/dividend, corporate-action, SIP, symbol-change
or survivorship semantics. IEX-only data is not accepted for a promotable
result; it could only support pipeline mechanics.

## SEC acquisition result

Status: **NOT RUN — AUTOMATED CLIENT IDENTITY ABSENT**.

`libs/sec` is an existing bounded official EDGAR adapter with submissions and
company-facts normalization, raw-payload provenance and acceptance-timestamp
validation. Its configuration intentionally requires `SEC_USER_AGENT` and
`SEC_CONTACT`; neither is configured. The adapter rejects fabricated identity
information, so no live SEC request was made. Credential-free SEC contract tests
passed, but fixture tests are not live source qualification.

The SEC route is technically suitable as an event-evidence component, but event
coverage, 8-K amendment handling, issuer/instrument mapping and historical
acquisition completeness remain unproven.

## Dataset identity and coverage

No HYP-EVENT-001A research dataset was created. Dataset identity is therefore
`NONE — QUALIFICATION INCOMPLETE`; no raw payload identity, panel hash or release
manifest exists.

The only existing catalog data remains four development fixtures:

| Source | Symbol | Interval | Coverage | Rows |
|---|---|---|---|---:|
| Alpaca fixture | AAPL | Daily | 2024-01-02 – 2024-02-01 | 22 |
| Alpaca fixture | AAPL | 1 minute | 2024-01-02 – 2024-01-05 | 2,729 |
| IB paper fixture | SPY | Daily | 2025-12-18 – 2026-06-10 | 120 |
| IB paper fixture | SPY | 1 minute | 2026-06-09 – 2026-06-11 | 500 |

Issuer count: 2 symbols, with no qualifying event panel. Event count:
0 acquired / 0 validated. Partition counts: 0 in development, validation and
OOS; 0 in final holdout. The 2025 holdout is sealed and was not read.

## Survivorship, corporate actions and quality

- Survivorship/delisted coverage: **UNKNOWN / NOT DEMONSTRATED**.
- Corporate actions: **UNKNOWN / NOT DEMONSTRATED**.
- Event amendments/duplicates: **NOT ASSESSED** for the historical panel.
- Timestamp integrity: provider/unit contract tests pass; no acquired panel
  exists to validate event availability against market sessions.
- Missing bars/stale/invalid bars: no panel-level metrics exist.
- Unresolved issuers/exclusions: no acquired panel exists; zero means no
  acquisition occurred, not that all records qualified.
- Benchmark coverage: no HYP-EVENT-001A SPY benchmark panel exists.
- Data quality decision: **INSUFFICIENT**.

The absence of metrics is explicitly different from zero defects.

## Licensing and cost

- Spend: **0**.
- New paid data: not introduced.
- New account or subscription: not created.
- New credentials: not requested.
- Alpaca data would remain local/private; raw data would not be redistributed
  without a separate rights review.
- SEC provenance and source-use records would be retained if acquisition is
  later authorized.
- No licensing decision can be completed for an unqualified Alpaca feed or
  unacquired panel.

## Verification

Passed:

- `go test ./libs/sec ./libs/marketdata -count=1`
- `go test ./... -count=1`
- SEC and Alpaca provider contract/unit tests
- roadmap and manifest validation from the prior readiness pack

The full Go test suite validates existing contracts and fixtures; it does not
prove live Alpaca or SEC entitlement/data readiness. No migration was changed.
No runtime implementation was added.

## Adversarial data review

No implementation defect was found because acquisition was correctly prevented
at the missing-configuration boundary. The following evidence remains absent and
therefore continues to block readiness:

- publication-time and same-day/look-ahead qualification;
- historical SIP qualification;
- adjustment/corporate-action qualification;
- symbol-change/delisted qualification;
- event deduplication and issuer-resolution results;
- dataset quality metrics and populated partitions;
- final-holdout access.

`Data-readiness blocking findings remaining: 0` means no unaddressed defect in
the bounded work performed. It does not mean that the dataset gate passed.

## Race-detector status

`RACE DETECTOR VERIFICATION OUTSTANDING` remains unchanged. No compiler or
system tooling was installed.

## Scientific limitations

No return, effect size, OOS result, statistical significance, profitability,
trading edge or forward-paper conclusion was produced. Phase 11 remains
`COMPLETE / CONDITIONAL GO`; soak infrastructure is demonstrated, but actual
forward-paper evidence remains `0 DAYS / 0 ORDERS`.

## Readiness decision

**HISTORICAL DATA INSUFFICIENT — PHASE 12 REMAINS NOT READY**

The authorized hypothesis is registered, but the data prerequisite is not
demonstrated. The next step requires an operator-supplied SEC client identity
and an already-authorized Alpaca account/feed, or an external decision on a
different lawful data route. No credential should be pasted into chat and no
paid source should be introduced without separate authorization.

WP-12.01, Qlib, RD-Agent, model training, recommendation changes and Phase 13
remain not started.
