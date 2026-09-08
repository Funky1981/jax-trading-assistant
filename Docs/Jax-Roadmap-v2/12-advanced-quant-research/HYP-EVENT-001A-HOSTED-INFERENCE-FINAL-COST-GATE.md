# HYP-EVENT-001A Hosted Inference Final Cost Gate

**Status:** Prepared for external technical-lead decision. No hosted
classification has been run. Current hosted-inference spend ceiling remains
`$0`.

## Repository and roadmap state

- Repository: `C:\Projects\Jax\jax-trading-assistant`
- Branch: `capability-reset`
- Evidence-extension implementation start: `69ac9ba5c89565cf03cbbfccd186130c11cc08b2`
- Phase 10: `COMPLETE / GO`
- Phase 11: `COMPLETE / GO`
- Phase 12: `COMPLETE / CONDITIONAL GO — REAL HYP-EVENT-001A SCIENTIFIC EVALUATION REQUIRED`
- Phase 13: `NOT STARTED`
- Scientific status: `TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE`
- Actual forward-paper evidence: `0 DAYS / 0 ORDERS`

## Parent and derived datasets

The parent dataset was not modified:

- Parent ID: `hyp-event-001a-sec-8k-alpaca-sip-2016-2025-v1`
- Parent content-manifest SHA-256: `db2b454793e50f28c34c9c2a7d91f798741936528752de7bd27cb52072a4864d`
- Derived ID: `hyp-event-001a-sec-8k-alpaca-sip-2016-2025-evidence-v2`
- Derived manifest SHA-256: `967dfcb18eff8b6a3f4ec39fedd3898537446a284dc2a868631926dc1f41408c`
- Local private path: `data/datasets/hyp-event-001a/evidence-v2/`

The derived manifest binds the parent ID and exact parent manifest hash. Raw
SEC evidence is local and ignored by Git; it is not committed or redistributed.

## SEC evidence-packet construction

Acquisition rule: `sec_accession_evidence_packet_v2_primary_and_same_accession_text_documents_excluding_duplicate_submission_text`.

For every qualifying non-amended Form 8-K, the packet retains the SEC index,
full document inventory, accession identity, CIK, acceptance/public-availability
cutoff, primary document, same-accession textual HTML/TXT documents and raw
content hashes. The selected text set is determined from index metadata before
any outcome analysis: the primary document plus all same-accession textual
documents, excluding index pages, binary/presentation support files and the
duplicate aggregate submission `.txt` rendering. Later accessions and
amendments are not merged into an earlier packet.

Normalization is deterministic (`sec_html_text_normalizer/v1`), preserves raw
bytes, removes presentation-only markup, and records normalized hash,
character count and a deterministic pre-tokenizer estimate. It does not
summarize or paraphrase filing content.

## Coverage and integrity

| Check | Result |
|---|---:|
| Qualifying event packets | 1,059 |
| SEC index inventories | 1,059 / 1,059 |
| Inventoried document rows | 10,731 |
| Selected raw primary/text documents | 2,418 |
| Primary-document matches | 1,059 / 1,059 |
| Accession mismatches | 0 |
| Retrieval failures | 0 |
| Missing selected documents | 0 |
| Raw bytes retained | 701,774,055 |
| 2016–2024 normalized characters | 51,436,074 |

The selected-document count includes 2,141 documents normalized for 2016–2024
and 277 documents retained for 2025 hash/inventory-only handling. The packet
parser and URL builder enforce SEC archive-host and accession binding. The
index inventory remains available for excluded non-semantic files.

### Semantic seals

- 2024: 120 packets; normalized only for automated size/integrity accounting;
  `SEMANTICALLY_SEALED_UNTIL_CLASSIFIER_FREEZE` and
  `SEALED_FOR_DESIGN_AND_OUTCOME_USE`. No classifier design or outcome was
  performed from 2024 content.
- 2025: 120 packets; 277 raw selected documents; no normalized text, normalized
  hashes or token estimates; `SEALED_HASH_AND_INVENTORY_ONLY` and
  `SEALED_FOR_SEMANTIC_AND_OUTCOME_USE`. No 2025 content was used for design,
  classification or outcomes.

## Frozen direction-classification contract

- Hypothesis: `HYP-EVENT-001A`
- Contract: `jax.hyp-event-001a.direction/v1`
- Prompt version: `d0b09acf412eb73ff97fc2e9bd5f4fc609dc57d5bd1dbd17d57f986b158d4143`
- Allowed directions: `POSITIVE`, `NEGATIVE`, `NEUTRAL`,
  `INSUFFICIENT_EVIDENCE`
- Allowed reasons: `GROUNDED_DIRECTION`, `INSUFFICIENT_EVIDENCE`,
  `CONFLICTING_EVIDENCE`, `INVALID_GROUNDING`
- Output ceiling: 256 estimated tokens
- Maximum retries: 1
- Planned calls: one result per event

The bounded question asks only for the issuer-directional economic implication
from supplied accession-time SEC evidence over the registered short horizon.
The prompt explicitly treats filing text as untrusted data, forbids embedded
instructions from changing the task, requires evidence anchors for polarity,
and forbids subsequent returns, later filings, later news, future prices,
revised information and model memory. No probability is requested and no
recommendation or execution authority is present.

Ungrounded polarity, invalid schema, wrong event/packet identity, non-UTC
cutoff, missing anchors or missing raw-response identity is rejected. Abstention
is valid. The contract is separate from Phase-06 recommendation and all
paper/live execution domains.

## Provider/model recommendation

**Candidate recommendation: OpenAI GPT-5.6 Luna, structured output, one bounded
call per event.** This is a recommendation for the external cost gate, not an
authorization to call the provider. The official model page describes Luna as
optimized for cost-sensitive workloads, supports structured outputs, and lists
a 1,050,000-token context window. Current pricing checked 2026-09-08 is
`$0.20 / 1M` input, `$0.02 / 1M` cached input and `$1.20 / 1M` output;
requests over 272K input tokens are priced at 2x input and 1.5x output for the
full request. Source: [OpenAI GPT-5.6 Luna pricing](https://developers.openai.com/api/docs/models/gpt-5.6-luna).

Terra/Sol-class inference is not justified without qualification evidence.
The existing local diagnostic inference path does not provide an accepted
historical classifier contract or a current provider-cost basis. Qlib/RD-Agent,
new Python runtimes and external research infrastructure are out of scope.

The no-truncation policy is frozen for this gate: send the complete selected
packet when it fits the model context; do not silently truncate. The largest
packet is below the context limit. One packet exceeds the 272K pricing
threshold, so its surcharge is included. No chunking or map/reduce classifier
is needed for the measured panel; introducing one would be a new contract.

## Measured token profile

Estimates use Jax's deterministic `(rune_count + 3) / 4` planning estimator;
they are not provider billing measurements. Fixed system/template/schema
overhead is 342 estimated tokens per call. The profile excludes all 2025
content.

| Partition | Events | Evidence tokens | Median | P75 | P90 | P95 | P99 | Max |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Development 2016–2021 | 583 | 7,140,078 | 7,677 | 13,859 | 25,378 | 42,457 | 86,788 | 295,938 |
| Validation 2022–2023 | 236 | 4,036,048 | 7,567 | 17,178 | 36,730 | 66,933 | 144,331 | 238,144 |
| OOS 2024 | 120 | 1,683,677 | 8,844 | 16,073 | 31,868 | 44,112 | 78,454 | 107,711 |
| All 2016–2024 | 939 | 12,859,803 | 7,826 | 14,987 | 29,702 | 49,970 | 107,611 | 295,938 |

With fixed overhead, one event is above 272K input tokens (`296,280` planned
input tokens) and zero events exceed the 1,050,000-token context limit.

## Calls, retries and cost scenarios

Retry policy is frozen at zero or one retry per event. The expected scenario
uses a transparent 5% planning retry rate; this is not an observed inference
rate because no inference has run. Cached input is not assumed, cache writes
are not assumed, and there is no tool-call charge in this classification plan.

| Scenario | Calls | Input tokens | Output tokens | Estimated total |
|---|---:|---:|---:|---:|
| Base, no retry | 939 | 13,180,941 | 240,384 | `$2.98` |
| Expected, 5% retry planning case | approximately 986 | approximately 13,839,988 | approximately 252,403 | `$3.13` |
| Maximum, one retry for every event | 1,878 | 26,361,882 | 480,768 | `$5.97` |

The expected total is `$3.133262` before any account/provider variance. The
maximum envelope is `$5.968117` before account/provider variance. The expected
cost components are approximately `$2.83` uncached
input and `$0.30` output, including the one large-request surcharge. The
maximum components are approximately `$5.39` input and `$0.58` output. Cached
input contribution is `$0` in these scenarios because reuse is not assumed.

**Proposed future hard ceiling: `$6.00 USD`, one-time, no overage.** The active
ceiling remains `$0`; this document does not authorize or execute a paid call.

## Qualification sample and quality gate

Before any bulk run, use a frozen 30–50 event DEVELOPMENT-only qualification
sample, recommended 40 events, stratified without returns by issuer, SEC item
type, packet size and apparent ambiguity. A small blinded human review subset
may be used; reviewers must not see subsequent returns. The sample is not a
performance sample and cannot open 2024 or 2025.

Planned pass criteria:

1. 100% exact event, accession, packet-hash and UTC-cutoff binding.
2. 100% rejection of malformed/extra-field output and invalid directions.
3. 100% polarity outputs contain packet anchors; unsupported cases abstain.
4. 100% compliance with the future-information prohibition, including
   filing-embedded prompt-injection fixtures.
5. At least 95% first-attempt schema-valid responses and 100% bounded completion
   after the single permitted retry.
6. No ungrounded polarity in the blinded review; disagreements are retained as
   abstention/quality findings rather than relabelled from returns.
7. Repeatability and model/provider behavior are recorded as qualification
   evidence; failure blocks bulk inference and does not trigger model
   escalation automatically.

## Security, egress and data rights

SEC configuration was loaded through the existing local Jax environment
convention. Required settings were present; secret values were not printed,
stored, committed or included in artifacts. Bulk inference is not run, so no
filing text has left the local environment. Any future approval must explicitly
review provider egress, retention and applicable data-processing terms.

SEC raw and normalized evidence remains private/local. Alpaca raw market data
remains private/local and is not redistributed. The derived artifact stores
provenance and hashes, not a downloadable public market-data package. No paid
data, new credential, broker call, live execution, paper execution or
recommendation mutation occurred.

## Verification and adversarial review

Completed focused verification:

- `go test ./internal/modules/hypevidence ./cmd/hyp-event-evidence -count=1`
- `go vet ./internal/modules/hypevidence ./cmd/hyp-event-evidence`
- parent manifest identity check;
- derived manifest and packet-count check;
- accession/primary-document binding check;
- zero retrieval-failure check;
- 2024/2025 semantic-seal metadata check;
- deterministic token-profile check;
- direction-contract validation tests;
- prompt-injection and malformed structured-result contract tests.

The fresh review attacked later-accession/amendment contamination, selective
exhibit choice, acquisition/publication timestamp confusion, cross-accession
URLs, normalization loss, 2024/2025 semantic access, hidden token truncation,
context overrun, pricing surcharge omission, retry multiplication, unbounded
calls and secret exposure. Blocking findings remaining: `0`.

Adversarial review: `PASS`.

Full repository verification remains required before any future scientific
experiment, including `go test ./... -count=1`, `go vet ./...`, migration and
replay contracts, roadmap/manifest validation and `git diff --check`. The
Phase-10/11 race condition was previously verified in the existing Docker
toolchain; native Windows cgo remains unavailable, which is not changed by
this evidence-extension stage.

## Scientific and safety limitations

- No direction labels have been generated.
- No development, validation or 2024 OOS outcome has been calculated.
- The 2025 holdout remains sealed and has not been scored.
- Current-ticker/survivorship and inactive/delisted coverage limitations remain;
  promotion is closed.
- Actual forward-paper evidence remains `0 DAYS / 0 ORDERS`.
- This cost gate does not establish an effect, profitability, edge or live
  readiness.
- `ALLOW_LIVE_TRADING=false`, `BROKER_EXECUTION_ALLOWED=false`, live worker
  disabled, maximum leverage `1x`, broker execution authority `NONE`.

## Recommended external decision

**APPROVE BOUNDED HOSTED INFERENCE** — only as a later, explicit one-time
authorization capped at `$6.00 USD`, after the 30–50 DEVELOPMENT-event
qualification stage is separately accepted. Until that decision is made, the
effective ceiling remains `$0` and no hosted inference may run.
