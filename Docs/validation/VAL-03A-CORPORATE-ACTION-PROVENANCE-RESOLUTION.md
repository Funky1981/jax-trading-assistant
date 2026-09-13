# Jax VAL-03A Corporate-Action Provenance Resolution

## Status

`VAL-03A = COMPLETE — NO-GO VAL-03 RESUME / CORPORATE ACTION DATA BLOCKED`.

VAL-03 attempt #1 remains a correct hard stop. `ma_crossover_v1` remains the
selected candidate and was not performance tested. OOS run count remains zero.

## Repository and frozen contract

- Repository: `C:\Projects\Jax\jax-trading-assistant`
- Branch: `capability-reset`
- Starting HEAD: `4734a05cd5ac20ad95822da76bea67c8ddc076bf`
- Frozen manifest: `Docs/validation/manifests/VAL-02-ma_crossover_v1-PREREGISTRATION.json`
- Manifest version: `v1.3`
- Manifest SHA-256: `09738525839a49d7b5d725e52d0b8547e768ce1d82e212e63c110ae1b2d6f65e`
- Candidate: `ma_crossover_v1`
- Candidate reselected: `NO`

All five implementation identities in the frozen manifest matched the current
Git blobs. No strategy parameter, cost, benchmark, placebo, sample-floor,
partition or holdout term changed.

## Official Alpaca REST findings

Current official documentation:

- https://docs.alpaca.markets/us/reference/corporateactions-1
- https://docs.alpaca.markets/us/v1.1/reference/corporateactions-1

The REST endpoint is:

`GET https://data.alpaca.markets/v1/corporate-actions`

The current documentation identifies these supported types:

- `cash_dividend`
- `forward_split`
- `reverse_split`
- `unit_split`
- `stock_dividend`
- `spin_off`
- `cash_merger`
- `stock_merger`
- `stock_and_cash_merger`
- `redemption`
- `name_change`
- `worthless_removal`
- `rights_distribution`
- `partial_call`
- `reorganization`
- `capital_gains_distribution`

The endpoint supports:

- inclusive `start` and `end` filters;
- filtering by `process_date`;
- `complete` and `all` data-quality modes;
- limits up to 1,000 records;
- `next_page_token` pagination;
- deterministic ascending or descending sort.

The documentation explicitly distinguishes processing-date filtering from
creation/publication time and warns that Alpaca does not guarantee corporate-
action creation time. Delays may occur between announcement, provider receipt,
processing and API availability.

## Official Alpaca SSE findings

Current official documentation:

https://docs.alpaca.markets/us/reference/subscribetocorporateactioneventssse

The endpoint is:

`GET https://stream.data.alpaca.markets/v1beta1/events/corporate-actions`

The documentation states that the stream provides corporate-action mutations:

- `insert`
- `update`
- `delete`

It supports:

- `since` replay by RFC-3339 timestamp;
- `until` close-by-timestamp;
- `since_id` replay by event ULID;
- `until_id` close-by-event ID;
- `Last-Event-Id` reconnect semantics;
- event-type filtering;
- region filtering;
- event-ID deduplication.

The documentation does not establish that the historical event stream retains
2015–2024 publication chronology. It describes replay mechanics but does not
provide a retention guarantee reaching the required start date.

## Actual SSE historical coverage

A bounded read-only request was made using the existing repository Alpaca
credentials without printing or persisting credential values.

Request boundary:

- `since=2015-01-01T00:00:00Z`
- `until=2015-01-03T23:59:59Z`
- `region=us`
- no market-bar request
- no 2025 interval

Observed result:

- HTTP status: `200`
- content type: `text/event-stream`
- replayed events observed before bounded read timeout: `0`
- event IDs observed: none
- event chronology established: no
- retention boundary established: no

The stream did not close with a historical event set during the bounded test.
This is insufficient to prove that 2015 history exists or that older event
publication chronology is retained.

Therefore:

`CORPORATE ACTION SSE PROVENANCE = INSUFFICIENT`

## Bounded REST metadata test

A second read-only request was made for the fixed nine-symbol universe with:

- `start=2015-01-01`
- `end=2024-12-31`
- `data_quality=complete`
- `limit=1000`
- `sort=asc`
- `region=us`

Observed result:

- HTTP status: `200`
- first-page corporate-action records: `0`
- next-page token: absent
- 2025 records observed: none

This does not establish complete historical absence; it establishes that the
bounded request did not provide the required records or a complete historical
coverage proof. The result cannot support silently omitting dividends or splits
from VAL-03.

## Corporate-action role separation

The following separation was tested as a possible v1.4 amendment.

### Signal/alpha dependency

`CORPORATE ACTION DATA USED AS ALPHA = NO`.

The MA strategy does not use declaration dates, process dates, expected
dividends, expected splits or corporate-action types to create, filter or
strengthen signals.

### Structural price-scale hygiene

Splits could theoretically be used only as a conservative data-quality/reset
label, never to improve direction, confidence, entry timing or eligibility.
However, the required complete historical split record was not established,
and the provider does not establish creation-time chronology. A conservative
structural rule therefore cannot be executed over the required dataset without
an auditable complete record.

### Realised dividend accounting

Dividend cash could be validly applied only after signal, entry and episode
membership were independently fixed. It could not affect signal eligibility,
entry, stop, target or direction. However, the required complete 2015–2024
dividend record was not established by the bounded REST/SSE checks. Missing or
ambiguous records cannot be silently dropped.

## Signal/alpha dependency

Corporate-action data used as alpha: `NO`.

The blocker is not caused by an alpha dependency. It is caused by the inability
to prove complete and auditable structural/cash-flow corporate-action coverage
for the frozen raw-price experiment.

## Split-hygiene contract

`BLOCKED` for VAL-03 resumption.

The v1.3 intended rule remains preserved:

- no retrospective future-informed adjustment;
- split effective date is handled prospectively;
- split-crossing indicator windows abstain until 200 valid post-split sessions;
- ambiguous split data causes provenance abstention.

The rule was not relaxed or changed after any outcome.

## Dividend realised-cash contract

`BLOCKED` for VAL-03 resumption.

The intended role remains ex-post realised cash-flow accounting only, after
episode membership is fixed. No dividend information may affect signal,
confidence, entry, stop, target or population selection. A complete auditable
record was not available from the bounded checks.

## Corporate-action type inventory

| Type | VAL-03 contract status | Reason |
| --- | --- | --- |
| cash dividend | UNSUPPORTED / DATASET BLOCKER | Complete historical record not established |
| forward split | UNSUPPORTED / DATASET BLOCKER | Complete historical record not established |
| reverse split | UNSUPPORTED / DATASET BLOCKER | Complete historical record not established |
| unit split | ABSTAIN AFFECTED PERIOD | Must not be silently ignored |
| stock dividend | ABSTAIN AFFECTED PERIOD | Changes share scale and requires provenance |
| spin-off | ABSTAIN AFFECTED PERIOD | Not covered by frozen simple cash/split contract |
| cash merger | ABSTAIN AFFECTED PERIOD | Structural security event |
| stock merger | ABSTAIN AFFECTED PERIOD | Structural security event |
| stock-and-cash merger | ABSTAIN AFFECTED PERIOD | Structural security event |
| redemption | ABSTAIN AFFECTED PERIOD | Security lifecycle event |
| name change | ABSTAIN AFFECTED PERIOD | Symbol/entity continuity requires proof |
| worthless removal | ABSTAIN AFFECTED PERIOD | Security lifecycle event |
| rights distribution | ABSTAIN AFFECTED PERIOD | Economic entitlement requires proof |
| partial call | ABSTAIN AFFECTED PERIOD | Unsupported simple ETF contract |
| reorganization | ABSTAIN AFFECTED PERIOD | Structural event requires proof |
| capital-gains distribution | ABSTAIN AFFECTED PERIOD | ETF cash-flow treatment not frozen |

No action type was silently ignored in the decision.

## Unsupported-action policy

No v1.4 contract was created because complete provider coverage and point-in-
time provenance were not established. Any affected period remains blocked or
must abstain under a separately reviewed contract. No market-bar acquisition
or performance scoring is permitted.

## Hardened adapter, if implemented

Not implemented.

The current repository adapter remains market-bars-only. Adding a corporate-
action adapter without a defensible historical chronology would create the
appearance of completeness without satisfying the scientific requirement.

## 2025 guard

2025 remains sealed. No 2025 market data or corporate-action interval was
requested. No 2025 action, signal, episode, return, benchmark or outcome was
inspected.

## Manifest amendment decision

No v1.4 amendment.

The v1.3 manifest remains unchanged and retains its original SHA-256. A revised
contract cannot be honestly frozen until the required provider coverage and
provenance are available or an externally reviewed scientific protocol changes
the data requirement.

## Why no amendment was scientifically justified

- VAL-03 OOS run count was zero.
- No historical strategy performance was viewed.
- No parameters were changed.
- No costs were changed.
- No benchmarks or placebo rules were changed.
- No sample floors were changed.
- No partitions were changed.
- No candidate was reselected.
- No 2025 data was accessed.
- The only new evidence was provider/provenance feasibility evidence.

## Files changed

- `Docs/validation/VAL-03A-CORPORATE-ACTION-PROVENANCE-RESOLUTION.md`
- `Docs/validation/VAL-03-DATASET-BLOCKER.md`
- `Docs/ROADMAP.md`
- `Docs/Jax-Roadmap-v2/ROADMAP-DECISION-LOG.md`

The frozen manifest and all strategy/scientific result artifacts were unchanged.

## Verification

- Repository state: PASS
- Manifest version/hash: PASS
- Frozen source identities: PASS
- Bounded Alpaca REST metadata test: PASS at HTTP/protocol level; coverage proof FAILED
- Bounded Alpaca SSE replay test: HTTP 200; historical replay proof FAILED
- `go test ./... -count=1`: PASS
- `go vet ./...`: PASS
- `git diff --check`: PASS
- Performance experiment: NOT EXECUTED
- Returns/P&L/Sharpe/win rate/OOS: NOT CALCULATED
- Market bars acquired: NO
- 2025 accessed: NO
- Broker calls: NO
- Paid data: NO
- Hosted inference: NO
- Forward paper: NOT STARTED
- Phase 13: NOT STARTED

## Exact VAL-03A status

`VAL-03A = COMPLETE — CORPORATE ACTION DATA BLOCKED`.

`CORPORATE ACTION SSE PROVENANCE = INSUFFICIENT`.

`CORPORATE ACTION DATA CONTRACT = BLOCKED`.

`VAL-03 DATASET ACQUISITION = NOT READY`.

## Safety

- `ALLOW_LIVE_TRADING=false` preserved
- `BROKER_EXECUTION_ALLOWED=false` preserved
- `EXECUTION_ENABLED=false` preserved
- Maximum leverage remains `1x`
- No broker execution or account mutation
- No orders, fills, trades, approvals or positions
- No hosted inference
- No paid data purchase
- 2025 holdout sealed
- Phase 13 not started

## Adversarial review

SELECTED CANDIDATE = ma_crossover_v1
CANDIDATE RESELECTED = NO
OOS PERFORMANCE OBSERVED = NO
OOS RUN COUNT = 0
2025 HOLDOUT = SEALED
CORPORATE ACTION DATA USED AS ALPHA = NO
SSE HISTORICAL PROVENANCE = INSUFFICIENT
SPLIT HANDLING = BLOCKED
DIVIDEND OUTCOME ACCOUNTING = BLOCKED
UNSUPPORTED CORPORATE ACTION TYPES = all types lacked a complete proven VAL-03 coverage path; non-simple structural types remain abstention/blocker cases
CORPORATE ACTION DATA CONTRACT = BLOCKED
MANIFEST = v1.3 FROZEN / RETAINED
MARKET-BAR DATASET ACQUIRED = NO
VAL-03 PERFORMANCE EXECUTED = NO
REAL FORWARD PAPER = NOT STARTED
PHASE 13 = NOT STARTED
SAFETY BOUNDARIES = PRESERVED

Adversarial VAL-03A review = PASS
