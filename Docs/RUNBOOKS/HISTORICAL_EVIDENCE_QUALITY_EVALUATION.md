# Historical evidence quality and market-relevance evaluation

## Scope

This phase asks whether existing genuine event-level `WATCH` decisions separate from genuine event-level `NO_TRADE` decisions on subsequent market movement. It is retrospective and read-only. It does not collect news or market data, change decision thresholds, call AI, infer trade direction, alter historical decisions, or create a candidate, approval, paper ticket, execution instruction, order intent, broker order, trade, or fill.

The evaluation implementation is `cmd/evidence-quality-evaluation` over `internal/modules/evidencequality`. It opens one PostgreSQL `REPEATABLE READ, READ ONLY` transaction, reads the existing projections and provenance-qualified candles, captures prohibited-state counts before and after, rolls the transaction back, and writes only isolated local report artefacts.

## Data map at evaluation time

The operator database contained 209 current event-decision projections across 209 events: 18 `NO_TRADE`, 191 `WATCH`, and zero `CANDIDATE`. It contained 201 evidence subjects: 132 currently projected `NO_TRADE`, 69 `WATCH`, and zero `CANDIDATE`. There were 197 single-event subjects, two two-event subjects, and two four-event subjects.

The event-decision source distribution was 205 `world-monitor` and four `world-monitor-local-proof`. Event categories were 170 `unknown`, 18 `macro_rates`, ten `energy_oil`, seven `geopolitical`, three `inflation`, and one `cyber_outage`. Raw source metadata identified 90 CNBC Top News, 71 BBC World, 23 Federal Reserve, and 25 with an unknown source name. Publication timestamps, receipt timestamps, and decision timestamps were present on all 209 rows; collection timestamps were present on 190. All 209 had publication no later than receipt and receipt no later than decision. The database-wide unfiltered latency medians were 10h38m18.130s publication to collection, 27.078s collection to receipt, and 48h09m31.138s receipt to decision.

Only 19/209 event decisions had resolved assets or sector themes, while 190 were explicitly unknown. The 19 resolved rows included four controlled local QQQ proofs and 14 smoke/scanner/workflow records; the remaining direct mapping was a genuine GLD mapping. The durable subject projection similarly had 11 resolved subjects and 190 unknown subjects.

The `candles` table contained 9,025 rows and 13 symbols overall, including legacy rows whose source, timeframe, and timestamp semantics are `unknown`. The evaluator excludes those unverifiable rows. The provenance-qualified subset contains 6,629 candles across 11 symbols: 6,617 daily candles and only 12 hourly QQQ candles. Providers are Alpaca and IB Bridge. Retained timestamps are `TIMESTAMPTZ` values with `interval_start` semantics; IB Bridge explicitly records regular-trading-hours data, while the Alpaca rows have a null RTH marker. Reliable daily coverage is 2025-03-10 through the 2026-07-30 session. The hourly observations are sparse, covering only parts of 2026-07-17 through 2026-07-27. The only retained ETF benchmarks are QQQ and SPY. There are no provenance-qualified persisted candles for GLD, XLE, TLT, or SOXX.

Three existing outcome records belong to an unrelated paper ticket (`1h`, `1d`, and `1w`). They are reported but never joined to the genuine-event population.

## Stable population

Ruleset `historical-evidence-quality-v1` considered all 209 current projections and included 101 events. It excluded:

- 14 manual smoke, scanner, chart-block, or promotion test records;
- four controlled QQQ local-proof records;
- 90 events received after the latest completed persisted market session.

No replay-only decision version entered the population because only `is_current=true` projections are read. Synthetic flags, controlled identifiers and sources, test hosts, canonical content hashes/URLs, timestamp ordering, and market coverage are all deterministic exclusion gates.

The included publication range is 2026-06-05T03:04:14Z through 2026-07-29T16:18:04Z. The receipt range is 2026-06-05T16:54:33.071509Z through 2026-07-29T16:40:50.945360Z. The current decisions were all produced at 2026-07-31T12:49:54.265511Z.

The included population is:

| Decision | Included | Conservatively mapped | Measurable receipt-anchored outcomes |
|---|---:|---:|---:|
| WATCH | 101 | 8 | 0 |
| NO_TRADE | 0 | 0 | 0 |
| CANDIDATE | 0 | 0 | 0 |

The eight mappings are one direct GLD mapping, six energy-category XLE proxies, and one inflation-category TLT proxy. Each mapping records type, symbol, confidence, reason, direct/proxy state, benchmark choice, and ruleset version. None can be measured from the existing qualified candle set. The other 93 events remain unknown; 89 of those have category `unknown`.

The original 18 event-level `NO_TRADE` decisions are all controlled/test/proof records and are correctly excluded. Current subject-level `NO_TRADE` projections are not substituted as historical labels: they were assigned later by the 24-hour staleness re-evaluation, after the relevant outcome windows. Substituting them would leak future timing into the label.

## Outcomes

No receipt-anchored `1h`, `1d`, or `1w` outcome is available for either decision class. Consequently median and mean absolute return, absolute abnormal return, realised range, maximum favourable/adverse excursion, threshold exceedance rates, bootstrap intervals, Mann-Whitney U, permutation results, and effect size are all correctly reported as unavailable with count zero. No strongest or weakest category can be ranked, no large `NO_TRADE` miss can be claimed, and no weak mapped `WATCH` can be identified without fabrication.

The evaluator supports all required measures and only emits inferential comparisons once both groups meet the configured minimum of five observations. Direction is never inferred. A daily event before the US session begins at that session's open; an event after the open begins at the next persisted session open. An hourly outcome begins at the first interval start at or after the event anchor. Missing or widely gapped windows fail closed.

## Latency

Within the 101-event stable population, median publication-to-collection latency is 8h05m17s, collection-to-receipt latency is 4s, and receipt-to-decision latency is 48h10m17s. Price movement before versus after receipt cannot be calculated because none of the mapped symbols has hourly coverage. Publication, collection, receipt, and decision anchors are emitted separately, with receipt as the primary operational anchor. Publication-time observations are never presented as action-time evidence.

The collection delay is material, but the first-order blocker is earlier: the population has no valid genuine event-level `NO_TRADE` comparator, 92.1% of included events are unmapped, and the eight mapped symbols do not overlap the qualified market-data universe.

## Evidence accumulation

All 101 included observations belong to single-event subjects. Twenty have primary/independent source evidence, while 81 have unknown independence; none has a mapped outcome. The multi-event subjects in the wider database are controlled repeated-report cases excluded from this genuine outcome population. Therefore no claim can be made that multiple events, independent source groups, or repeated source groups correlate with larger movement. Evidence accumulation remains behaviourally proven but market-value unproven.

## Bias controls

- The evaluation uses genuine persisted rows only and applies no market-data fetch or fabrication.
- Controlled QQQ proofs, test URLs and identifiers, duplicates, replay-only versions, invalid timestamps, and post-coverage events are excluded before mapping or outcome calculation.
- The current subject projection is not used to relabel an older event.
- Unknown assets remain unknown; no universal SPY or QQQ proxy is applied.
- Direct mappings take precedence over a small versioned category-proxy allowlist.
- Benchmarks are explicit and versioned. QQQ is compared with SPY, SOXX with QQQ, and XLE with SPY; mappings without an appropriate persisted benchmark retain raw-return eligibility only.
- Outcome candles begin at or after the selected anchor. No pre-event candle supplies an entry price.
- The primary anchor is Jax receipt; publication-time results are separate latency diagnostics.
- No directional accuracy is reported because the included decisions contain no trustworthy pre-outcome direction.

## Reproduction

With the existing Jax PostgreSQL service running, execute from the repository root:

```powershell
.\scripts\evaluate-evidence-quality.ps1
```

The command requires explicit ruleset `historical-evidence-quality-v1` internally and writes deterministic artefacts for unchanged inputs to:

- `.runtime/evidence-quality/report.md`
- `.runtime/evidence-quality/summary.json`
- `.runtime/evidence-quality/population.csv`
- `.runtime/evidence-quality/outcomes.csv`

The evaluated input fingerprint is `a15d48d84b2a36103036fff484cebfd5997e846747ef8c719cc8527431e231b1`. Two consecutive runs produced byte-identical hashes for all four artefacts.

## Verification

The focused evaluator run executed 14 top-level tests: 14 passed, zero failed, and zero skipped. It covers population filtering, direct and proxy mapping, unknown assets, benchmark selection, raw and abnormal returns, all timestamp anchors, US session boundaries, missing candles, duplicates, publication/receipt ordering, no look-ahead, deterministic statistics/reruns, and prohibited-state mutation detection. Targeted `go vet` also passed.

An uncached `go test -json -count=1 ./...` executed 942 top-level tests: 940 passed, two failed, and zero skipped. The failures are the two pre-existing approval-system defects:

- `cmd/trader.TestAISuggestionPromoteCreatesApprovalCandidate`: the candidate is `blocked` rather than the fixture's expected `awaiting_approval` state.
- `internal/modules/approvals.TestApprovalDetailPersistedStatesAndDuplicateSafety`: the `approval_and_ticket` fixture fails PostgreSQL parameter type deduction.

Neither failure contaminates this evaluation. The evaluation command dependency graph contains neither `cmd/trader` nor `internal/modules/approvals`; its only reuse from event decisions is the environment safety-state reader. The evaluation package and command pass independently.

The repository's standard verification wrapper stops at a pre-existing repository-wide `gofmt` backlog across unrelated tracked files. All new Go files are formatted; the scoped formatting check, tests, and vet pass. No frontend code or World Monitor code changed, so frontend and World Monitor suites are out of scope.

## Safety result

Runtime safety was paper mode, live trading false, execution false, execution worker false, broker execution false, and maximum leverage `1x`.

Prohibited-state before and after counts were identical: two approvals, two candidate approvals, one paper ticket, one execution instruction, zero order intents, zero broker order identifiers, zero trades, and zero fills. Delta was zero for every record type.

## Conclusion

Primary conclusion: **timing/data quality prevents conclusion**.

The current dataset cannot answer whether `WATCH` separates from `NO_TRADE`. It contains no valid genuine event-level `NO_TRADE` comparison group after mandatory exclusions and no measurable mapped outcome. The dominant actionable limitation is asset resolution and its mismatch with persisted market coverage; collection and decision latency are additional material constraints.

Product recommendation: **SOLVE ASSET RESOLUTION FIRST**.

Final verdict: **PASS WITH LIMITATIONS**.
