# Roadmap Decision Log

## RD-2026-09-09-12 - HYP-EVENT-001A real scientific evaluation completed

- Date: 2026-09-09
- Stage: `PHASE 12 — REAL HYP-EVENT-001A SCIENTIFIC EVALUATION`
- Status: `COMPLETE / CONDITIONAL GO — PROMOTION CLOSED`
- Decision authority: external conditional Phase-12 authorization; final phase
  review remains external
- Hypothesis: `HYP-EVENT-001A — SEC evidence-backed issuer-event reaction`

The authorized GPT-5.6 Luna hosted-inference run completed using the frozen
`jax.hyp-event-001a.direction/v1` contract and prompt identity
`d0b09acf412eb73ff97fc2e9bd5f4fc609dc57d5bd1dbd17d57f986b158d4143`.
It produced 939 unique 2016–2024 event direction records (POSITIVE 483,
NEGATIVE 162, NEUTRAL 259, INSUFFICIENT_EVIDENCE 35), with 3 bounded output
validation retries and accounted provider usage cost of `$2.677034`, below the
authorized `$4.50` ceiling. No 2025 event semantics or outcomes were used.

The immutable scientific artifact is
`data/datasets/hyp-event-001a/scientific-results-v3/report.json` (private and
ignored by Git). It contains 645 valid market/event observations after the
frozen market-window rules: 391 development, 162 validation and 92 formal 2024
OOS. The candidate was frozen before OOS; costs are bound to
`cost_phase11_v1`; the full registered falsification plan is retained. The
conditioned candidate exceeded direction-only on the formal 2024 descriptive
cost-adjusted mean, but this is retained only as a `PROMISING RESEARCH
CANDIDATE — NOT PROMOTED`; it is not a profitability or trading-edge claim.

Promotion remains `PROMOTION_CLOSED` because the 2025 final holdout is sealed,
current-ticker survivorship/delisted coverage is unresolved, and actual
forward-paper evidence remains `0 DAYS / 0 ORDERS`. Recommendation logic,
paper history, portfolio state and execution authority are unchanged. Phase 10
and Phase 11 remain `COMPLETE / GO`; Phase 13 remains `NOT STARTED`.

## RD-2026-09-08-10 - Phase 10/11 GO and Phase 12 scientific condition

- Date: 2026-09-08
- Stage: `PHASE 12 — REAL HYP-EVENT-001A SCIENTIFIC EVALUATION`
- Status: `COMPLETE / CONDITIONAL GO — REAL SCIENTIFIC EVALUATION REQUIRED`
- Decision: external technical-lead `GO PHASE 10`, `GO PHASE 11`,
  `CONDITIONAL GO PHASE 12`
- Hypothesis: `HYP-EVENT-001A — SEC evidence-backed issuer-event reaction`

The supplied external decision is recorded. Phase 10 and Phase 11 are now
`COMPLETE / GO`; actual forward-paper evidence remains `0 DAYS / 0 ORDERS`.
Phase 12 remains research-only and conditional on completing the real
HYP-EVENT-001A scientific evaluation. The frozen dataset identity is verified,
but its SEC panel contains metadata and item codes rather than filing bodies,
and no suitable immutable historical direction labels exist. The existing
unversioned keyword helper is not an admissible historical classifier.

The development/validation/OOS evaluation is therefore stopped before any
outcome analysis at the hosted inference/data-evidence cost gate. No bulk
hosted inference, new filing-body acquisition, 2024 outcome scoring or 2025
holdout outcome access occurred. No recommendation logic, paper history or
execution authority changed. `TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT
SAMPLE` and `Phase 13 — NOT STARTED` remain in force. External resolution is
required before any inference spend or frozen-dataset evidence extension.

## RD-2026-09-08-09 - Phase 12 internal completion and race verification

- Date: 2026-09-08
- Stage: `PHASE 12 — ADVANCED QUANT RESEARCH`
- Status: `COMPLETE / INTERNALLY VERIFIED — RESEARCH ONLY`
- Hypothesis: `HYP-EVENT-001A — SEC evidence-backed issuer-event reaction`
- Decision authority: external Phase-12 review remains required

WP-12.01 through WP-12.08 are implemented in the existing Jax Go architecture.
The capability exit condition is demonstrated by the Phase-12 harness: feature
knowability, sealed-holdout enforcement, validation-only selection, one-way OOS,
trial/falsification retention, drift/failure states and a fail-closed promotion
gate. No real HYP-EVENT-001A direction labels or performance outcomes were
generated, no advanced model was trained or promoted, and recommendation logic
was not changed. The 2025 holdout remains sealed.

The required existing-container command
`CGO_ENABLED=1 go test -race ./internal/modules/workflow
./internal/modules/papertrading -count=1` passed. Native Windows cgo remains
unavailable; the result is limited to the tested packages and is not a claim of
native-host race verification. Actual forward-paper evidence remains `0 DAYS /
0 ORDERS`; `TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE` remains
unchanged. Phase 13 is not started.

## RD-2026-09-08-08 - Phase 12 authorization and internal completion

- Date: 2026-09-08
- Stage: `PHASE 12 — ADVANCED QUANT RESEARCH`
- Status: `COMPLETE / INTERNALLY VERIFIED — RESEARCH ONLY`
- Hypothesis: `HYP-EVENT-001A — SEC evidence-backed issuer-event reaction`
- Decision authority: external authorization recorded; phase-level review pending

Phase 12 was externally authorized after acceptance of Phase-07 GO, the
HYP-EVENT-001A data-readiness gate and the explicit falsifiable hypothesis.
WP-12.01 through WP-12.08 are implemented in the existing Jax Go architecture.
No Qlib/RD-Agent runtime, ML dependency, hosted inference, paid service or new
credential was introduced.

The phase exit condition is demonstrated at capability level: advanced model
selection is validation-only, formal OOS is one-way and non-holdout, features
carry event-time knowability, falsification/trial evidence is retained, drift
and failure states are explicit, and the promotion gate cannot mutate
recommendation logic. The phase harness uses synthetic event-time-safe fixture
labels; the real HYP-EVENT-001A panel has no materialized direction assessment,
so no real HYP performance or edge claim is made.

No advanced model was promoted. Promotion remains closed for the sealed 2025
holdout, unresolved current-ticker survivorship and zero actual forward-paper
evidence. `TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE`, `0 DAYS / 0
ORDERS`, `SOAK INFRASTRUCTURE DEMONSTRATED` and outstanding race-detector
verification remain preserved. Phase 13 is not started.

## RD-2026-09-08-07 - HYP-EVENT-001A historical-data readiness — gate demonstrated

- Date: 2026-09-08
- Stage: `PHASE 12 HISTORICAL DATA READINESS — IN PROGRESS`
- Hypothesis: `HYP-EVENT-001A — SEC evidence-backed issuer-event reaction`
- Status: `SUFFICIENT CLEAN HISTORICAL DATA — READY FOR EXTERNAL PHASE-12 AUTHORISATION`
- Decision authority: bounded readiness implementation; external Phase-12 review remains required

The previously authorised hypothesis was registered and the repository’s
existing root `.env` loading convention was used without changing or exposing
secret values. `ALPACA_API_KEY`, `ALPACA_API_SECRET`, `SEC_USER_AGENT` and
`SEC_CONTACT` were present. Level-1 Alpaca SIP and SEC EDGAR qualification,
including a temporal cross-check, passed before bounded acquisition.

The frozen local/private dataset is
`hyp-event-001a-sec-8k-alpaca-sip-2016-2025-v1` with content-manifest SHA-256
`db2b454793e50f28c34c9c2a7d91f798741936528752de7bd27cb52072a4864d`. It
contains 1,059 qualifying deduplicated exact 8-K events across 60 issuers,
154,674 raw market rows, 120 qualifying 2025 holdout observations and a
maximum issuer share of approximately 1.79%. The holdout is sealed and no
outcomes were calculated.

The result meets the readiness floors for initial research only. Current-ticker
survivorship, incomplete demonstrated delisted/rename reconstruction,
acceptance timestamp as the available public-availability proxy and pending
field-level corporate-action reconciliation remain explicit limitations. No
paid data, new account or new credential was introduced. The migration/runtime
system and recommendation logic were not changed. `TRADING EDGE NOT
DEMONSTRATED / INSUFFICIENT SAMPLE`, 0 days / 0 orders of actual forward paper
evidence and outstanding race-detector verification remain preserved. WP-12.01
and Phase 13 remain unstarted.

## RD-2026-09-08-06 - HYP-EVENT-001A data-readiness authorization — insufficient

- Date: 2026-09-08
- Stage: `PHASE 12 HISTORICAL DATA READINESS — IN PROGRESS`
- Hypothesis: `HYP-EVENT-001A — SEC evidence-backed issuer-event reaction`
- Status: `AUTHORISED FOR DATA READINESS AND BASELINE RESEARCH DESIGN`
- Phase-12 implementation: `NOT AUTHORISED`
- Decision authority: external GPT-5.6 Sol technical-lead review

The authorized research question is whether timestamped material SEC 8-K events,
directionally classified from information available at the time, show more
consistent subsequent SPY-relative returns when evidence quality is stronger.
The frozen initial design uses 2016–2025, next regular-session open entry, a
5-trading-day primary horizon, 1/3-day diagnostics, SPY benchmark, null and
direction-only baselines, frozen Phase-11 costs and a sealed 2025 holdout.

Level-1 qualification did not pass. Alpaca is disabled with no configured API
key, secret or feed, so no entitlement/SIP/history/adjustment qualification was
possible. The SEC adapter requires configured `SEC_USER_AGENT` and `SEC_CONTACT`
and neither is present; no live SEC request was made. Existing SEC/marketdata
contract tests passed. No large acquisition occurred.

The historical-data result is `INSUFFICIENT`: no HYP-EVENT-001A dataset exists,
no event panel was acquired, and survivorship, delisted coverage,
corporate-action treatment, event timing and panel-level quality metrics remain
unproven. Actual forward-paper evidence remains 0 days / 0 orders, race-detector
verification remains outstanding, and `TRADING EDGE NOT DEMONSTRATED /
INSUFFICIENT SAMPLE` is preserved. WP-12.01, Phase-12 runtime work and Phase 13
remain unstarted.

## RD-2026-09-08-05 - Phase 12 readiness remediation — not ready

- Date: 2026-09-08
- Stage: `PHASE 12 READINESS — HISTORICAL DATA & RESEARCH HYPOTHESIS`
- Phase: 12 — Advanced Quant Research
- Status: `PHASE 12 NOT READY / NOT AUTHORISED`
- Decision authority: bounded readiness assessment; no Phase-12 implementation

The accepted Phase-11 state remains `COMPLETE / CONDITIONAL GO`; race-detector
verification is still outstanding because the current environment lacks a usable
`gcc`/cgo toolchain. Actual forward-paper evidence remains 0 days / 0 orders and
the scientific status remains `TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT
SAMPLE`.

The readiness pack independently assessed the two remaining Phase-12
prerequisites. Historical data is `INSUFFICIENT`: the current catalog contains
only four narrow OHLCV fixtures (AAPL daily 22 rows, AAPL minute 2,729 rows, SPY
daily 120 rows and SPY minute 500 rows), with no demonstrated clean broad panel,
point-in-time universe, delisted coverage, corporate-action policy or complete
train/validation/OOS evidence. No `AUTHORISED EXPLICIT RESEARCH HYPOTHESIS` was
found; templates, documentation examples and golden fixtures are not
authorization.

The bounded pack defines a first-experiment data standard, inventories existing
providers, records source/cost/licensing gaps, and provides four candidate
hypotheses. `HYP-EVENT-001` is a technical recommendation only and is not
authorized. No data was purchased or downloaded, no credential was requested,
and no runtime/provider/ML/recommendation code was changed. WP-12.01 and Phase
13 remain unstarted. The next decision requires external selection and
authorization of a falsifiable hypothesis plus independent acceptance of a
sufficient clean dataset.

## RD-2026-09-08-04 - Phase 12 prerequisite review — not ready

- Date: 2026-09-08
- Phase: 12 — Advanced Quant Research
- Status: `NOT READY / NOT AUTHORISED`
- Decision authority: autonomous prerequisite verification; no Phase-12 start

Phase 11 is recorded as `COMPLETE / CONDITIONAL GO` under the external
technical-lead decision. The inherited race-detector condition remains open:
`RACE DETECTOR NOT VERIFIED — ENVIRONMENTAL LIMITATION`. Actual forward-paper
evidence remains 0 days / 0 orders, and `TRADING EDGE NOT DEMONSTRATED /
INSUFFICIENT SAMPLE` is unchanged.

Phase-07 GO is satisfied. Prerequisite A is `INSUFFICIENT`: the repository's
available dataset catalog contains only four narrow OHLCV fixtures — AAPL
daily 22 rows, AAPL minute 2,729 rows, SPY daily 120 rows and SPY minute 500
rows. The repository does not demonstrate the breadth, history, point-in-time
fundamental/macro coverage, corporate-action treatment, delisted universe,
survivorship controls, licensing record or train/validation/OOS coverage
needed for advanced research. The SPY historical fixtures are sourced from an
`ib-bridge-paper` development path.

Prerequisite B is `NO AUTHORISED EXPLICIT HYPOTHESIS`. The search found the
generic hypothesis template, documentation examples such as `hyp_swing_001`,
and golden test fixtures such as `hyp_commodity_dislocation`; none is a
Phase-12 authorization or an accepted decision promoting a falsifiable
advanced-quant hypothesis. No hypothesis was invented or promoted. WP-12.01
and all later Phase-12 packages remain unstarted. No paid data or dependency
is authorized or required by this review. Phase 13 is NOT STARTED.

## RD-2026-09-08-03 - Phase 11 exit demonstrated / external review pending

- Date: 2026-09-08
- Phase: 11 — High-Fidelity Paper Trading
- Status: Internally complete; external phase-gate review pending
- Decision authority: autonomous phase verification; no self-awarded GO

WP-11.01 through WP-11.08 are implemented in bounded commits. The exact
Phase-11 capability gate is demonstrated by
`internal/modules/papertrading/phase11_exit_test.go`: an approved Phase-10
paper intent becomes a provenance-bound paper order, deterministic cost and
latency rules apply, partial fills update the isolated event-derived ledger,
reconciliation detects corruption, attribution separates execution effects,
restart/idempotent replay does not duplicate fills, breakers block processing,
and no live path or real portfolio mutation exists.

A final adversarial pass identified and corrected a same-length tampered
ledger-event-stream reconciliation gap in `0d1ab24`/`ecfd6f5`. It also found
map-iteration nondeterminism in the exit harness and corrected it in `450f85c`.
The focused tests and exit harness were rerun after each correction.

The soak result is **SOAK INFRASTRUCTURE DEMONSTRATED** using accelerated
synthetic fixtures. Actual elapsed forward-paper evidence and real paper
recommendation/order sample are both zero. `TRADING EDGE NOT DEMONSTRATED /
INSUFFICIENT SAMPLE` remains unchanged. Phase 10 remains COMPLETE / CONDITIONAL
GO with race verification required before Phase 11 GO; Phase 12 is NOT STARTED.

## RD-2026-09-08-02 - WP-11.01 implementation

- Date: 2026-09-08
- Phase: 11 — High-Fidelity Paper Trading
- Status: Implemented / internally verified
- Decision authority: autonomous implementation under external Phase-10 conditional GO

WP-11.01 establishes the provider-neutral capability contract for the isolated
paper domain. Environment, supported order types, precision, market-hours,
quote-age, partial-fill, account/position/status and reconciliation
capabilities are explicit and versioned. Unknown mode and unsupported
capabilities fail closed. No live adapter, broker credential, paid dependency
or execution authority was introduced. Phase-10 race verification remains
open and is required before a Phase-11 GO recommendation.

## RD-2026-09-08-01 - Phase 10 conditional GO / Phase 11 authorization

- Date: 2026-09-08
- Phase: 10 -> 11 transition
- Status: External technical-lead `CONDITIONAL GO PHASE 10`; Phase 11 authorised/in progress
- Decision authority: external GPT-5.6 Sol technical-lead review

Phase 10 is accepted as **COMPLETE / CONDITIONAL GO**. Its exact workflow
exit condition and migration remediation were accepted. The remaining
condition is `RACE DETECTOR VERIFICATION REQUIRED BEFORE GO PHASE 11`; the
available Windows Go environment still lacks cgo/gcc, so race verification is
not claimed as passed. Phase 11 is authorised beginning at WP-11.01. The
scientific status remains **TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT
SAMPLE**. Live execution remains disabled.

## RD-2026-09-07-29 - Phase 09 exit demonstrated / external review pending

- Date: 2026-09-07
- Phase: 09 — Portfolio Intelligence & Deterministic Risk
- Work packages: WP-09.01 through WP-09.07
- Status: Exit condition demonstrated; external technical-lead review pending
- Decision authority: autonomous phase verification; no self-awarded phase GO

Phase 09 delivers a provider-neutral canonical portfolio snapshot with explicit
unknown/freshness state, deterministic signed exposure/concentration and safe
correlation semantics, versioned risk policy, deterministic ACCEPT/AMEND/REJECT,
bounded descriptive position proposals, frozen stress scenarios, and stable
append-only audit artifacts. The phase harness proves a synthetic frozen
portfolio can produce all three outcomes with reproducible reason codes and no
portfolio or execution-side effect. It also proves stale/unknown state does not
silently return ACCEPT and invalid policy/scenario paths fail closed.

Evidence: `internal/modules/portfoliorisk/phase09_exit_test.go` and
`09-portfolio-intelligence-deterministic-risk/PHASE-09-INTERNAL-VERIFICATION.md`.
Full tests and vet pass. Race detection remains a
`NON-BLOCKING ENVIRONMENTAL LIMITATION` because `gcc` is unavailable; it has
not passed. No paid service or credential was added. `TRADING EDGE NOT
DEMONSTRATED / INSUFFICIENT SAMPLE` is preserved. Fresh adversarial review over
`69224644a8ac2d6d1cdb61894ea6c6a8c90604db..CURRENT_HEAD` found and fixed the
analytics provenance-binding issue and related boundary issues. Blocking
findings remaining: `0`. Adversarial phase review: `PASS`. Phase 10 is not
started.

## RD-2026-09-07-28 - Phase 08 GO / Phase 09 authorized

- Date: 2026-09-07
- Phase: 08 -> 09 transition
- Status: External GPT-5.6 Sol accepted Phase 08 as COMPLETE / GO; Phase 09 AUTHORISED / IN PROGRESS
- Decision authority: external technical-lead review for Phase 08; autonomous implementation within Phase 09

Phase 08's demonstrated bounded-researcher exit condition and dedicated
adversarial review were accepted. The race-detection gap caused by unavailable
`gcc` is retained as a `NON-BLOCKING ENVIRONMENTAL LIMITATION`; race detection
has not passed and must be rerun when the toolchain supports it. Phase 09 starts
at WP-09.01. `TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE` is
preserved; Phase 07/08 acceptance is not a profitability claim. Phase 10 is
not authorized or started.

## RD-2026-09-07-27 - Resume budget integrity correction

- Date: 2026-09-07
- Phase: 08 — Controlled AI Tools & Durable Research Agents
- Scope: WP-08.03 / WP-08.04 cross-package correction
- Status: Corrected / internally verified; Phase-08 exit proof rerun
- Decision authority: autonomous adversarial correction under authorised Phase-08 scope

The phase review found that a resumed task could otherwise reuse a fresh
in-memory budget controller. `NewBudgetControllerWithState` now restores
checkpointed counters and task start time, and `Context` enforces the remaining
wall-clock deadline. Checkpoint validation also requires UTC task start time,
valid evidence identities, permitted tool tiers and known/ambiguous usage
status. Regression tests and the full Phase-08 exit harness pass.

## RD-2026-09-07-26 - Phase 08 exit demonstrated / external review pending

- Date: 2026-09-07
- Phase: 08 — Controlled AI Tools & Durable Research Agents
- Work packages: WP-08.01 through WP-08.08
- Status: Exit condition demonstrated; external technical-lead review pending
- Decision authority: autonomous phase verification; no self-awarded phase GO

The reproducible Phase-08 harness executes the complete bounded researcher
chain: explicit objective and plan, registered read-only tools with schema
validation, evidence/provenance and budget accounting, durable checkpoint,
simulated interruption, bounded resume, evidence-linked gap/replan, critic
reflection and report improvement, controlled failure recovery, and final
checkpoint. It also proves denial of forbidden tool, exhausted budget,
corrupted checkpoint and incompatible checkpoint paths. The evaluation compares
the agent with a simpler baseline and records local-fixture routing with zero
spend. `ALLOW_LIVE_TRADING=false`, `BROKER_EXECUTION_ALLOWED=false`, execution
worker disabled, maximum leverage 1x and recommendation execution authority
`NONE` remain enforced. Phase 09 is not authorized or started.

Evidence: `internal/modules/harness/evaluation_test.go` and
`internal/modules/harness/evaluation.go`.

Adversarial phase review over `0cfafde..CURRENT_HEAD` is complete: arbitrary
tool execution, permission escalation, schema bypass, prompt-injection scope
change, loop/retry amplification, budget bypass, checkpoint tampering and
stale resume, transcript replay, semantic-cache contamination, model
escalation, critic scope expansion, cancellation/de-duplication, hidden cost,
recommendation-gate bypass and execution authority were reviewed. Blocking
findings remaining: `0`. Adversarial phase review: `PASS`.

## RD-2026-09-07-25 - WP-08.07 internal verification

- Date: 2026-09-07
- Phase: 08 — Controlled AI Tools & Durable Research Agents
- Work package: WP-08.07 — Research memory with provenance
- Status: Implemented / internally verified; WP-08.08 current
- Decision authority: autonomous package verification under authorised Phase-08 scope

WP-08.07 adds complete exact validity keys and labelled provenance-preserving
reuse. Material changes to objective/evidence/vintage/tool/prompt/output
contract/provider/model/research/policy/quant/runtime dimensions produce a
miss; stale, superseded and invalidated entries cannot be reused. Source
evidence and model conclusions remain distinct. Additive Postgres persistence
uses the existing boundary, while semantic result caching and vector storage
remain disabled. Focused provenance, invalidation, freshness and lifecycle
tests pass.

Evidence: `08-controlled-ai-tools-durable-research-agents/WP-08.07-research-memory-with-provenance.md`.

## RD-2026-09-07-24 - WP-08.06 internal verification

- Date: 2026-09-07
- Phase: 08 — Controlled AI Tools & Durable Research Agents
- Work package: WP-08.06 — Critic/reflection stage
- Status: Implemented / internally verified; WP-08.07 current
- Decision authority: autonomous package verification under authorised Phase-08 scope

WP-08.06 adds bounded, versioned critic policy and reflection contracts. The
stage identifies evidence-quality defects, preserves contradictions, adds
unknowns, downgrades unsupported conclusions and requests only bounded
evidence-linked research. Critique input is report-hash bound, cycles are capped
at three, and applying it cannot alter objective, plan, permissions, budgets or
execution authority. Focused improvement, stale-input, contradiction,
cycle-limit and injection tests pass.

Evidence: `08-controlled-ai-tools-durable-research-agents/WP-08.06-critic-reflection-stage.md`.

## RD-2026-09-07-23 - WP-08.05 internal verification

- Date: 2026-09-07
- Phase: 08 — Controlled AI Tools & Durable Research Agents
- Work package: WP-08.05 — Adaptive gap-finding/replanning
- Status: Implemented / internally verified; WP-08.06 current
- Decision authority: autonomous package verification under authorised Phase-08 scope

WP-08.05 adds evidence-linked bounded gap and replan contracts. A replan can
only add registered, version-matched, permitted read-only tool steps, while
preserving the original objective, trigger evidence, plan history and
checkpoint replan count. Explicit sufficient, insufficient, blocked and
budget-exhausted outcomes prevent wandering; candidate work is rejected when
the replan cap is exhausted. Focused positive, negative and injection-shaped
tests pass.

Evidence: `08-controlled-ai-tools-durable-research-agents/WP-08.05-adaptive-gap-finding-replanning.md`.

## RD-2026-09-07-22 - WP-08.04 internal verification

- Date: 2026-09-07
- Phase: 08 — Controlled AI Tools & Durable Research Agents
- Work package: WP-08.04 — Durable research task/checkpoint state
- Status: Implemented / internally verified; WP-08.05 current
- Decision authority: autonomous package verification under authorised Phase-08 scope

WP-08.04 adds the versioned durable checkpoint contract and additive Postgres
JSONB persistence. Checkpoints are immutable content-addressed task/version
records containing bounded plan progress, evidence and contradiction identity,
tool/model provenance, report state, budgets and failure status. Resume builds
minimum sufficient bounded context without transcript replay. Integrity checks
reject corruption, unsupported state, stale/mutated plan membership, trusted
tool records and over-budget state. Focused checkpoint and migration tests pass.

Evidence: `08-controlled-ai-tools-durable-research-agents/WP-08.04-durable-research-task-checkpoint-state.md`.

## RD-2026-09-07-21 - WP-08.03 internal verification

- Date: 2026-09-07
- Phase: 08 — Controlled AI Tools & Durable Research Agents
- Work package: WP-08.03 — Timeout/cancellation/budget controls
- Status: Implemented / internally verified; WP-08.04 current
- Decision authority: autonomous package verification under authorised Phase-08 scope

WP-08.03 adds deterministic wall-clock, step, tool, model, retry, token, cost
and model-tier controls plus cancellation-aware contexts. Provider usage is
normalized without fabricating unavailable categories; raw usage hashes,
cache/reasoning availability and known/ambiguous cost are retained. Strict
budget runs reject incomplete usage and all limits fail closed. Focused budget,
pricing and cancellation tests pass without paid inference.

Evidence: `08-controlled-ai-tools-durable-research-agents/WP-08.03-timeout-cancellation-budget-controls.md`.

## RD-2026-09-07-20 - WP-08.02 internal verification

- Date: 2026-09-07
- Phase: 08 — Controlled AI Tools & Durable Research Agents
- Work package: WP-08.02 — Read-only tool permission tiers
- Status: Implemented / internally verified; WP-08.03 current
- Decision authority: autonomous package verification under authorised Phase-08 scope

WP-08.02 adds a versioned research permission policy and controlled invocation
boundary. Only declared read-only evidence/derived/memory tiers are allowed;
external data is denied by default; execution and arbitrary-system capabilities
are explicitly forbidden. Outputs require provenance and remain untrusted data,
including injection-shaped content. Focused permission, safety and injection
tests pass.

Evidence: `08-controlled-ai-tools-durable-research-agents/WP-08.02-read-only-tool-permission-tiers.md`.

## RD-2026-09-07-19 - WP-08.01 internal verification

- Date: 2026-09-07
- Phase: 08 — Controlled AI Tools & Durable Research Agents
- Work package: WP-08.01 — Tool registry and JSON-schema validation
- Status: Implemented / internally verified; WP-08.02 current
- Decision authority: autonomous package verification under authorised Phase-08 scope

WP-08.01 hardens the existing harness boundary with a versioned controlled-tool
registry and bounded JSON argument validation. Definitions are explicitly
read-only, provenance-required and handler-bound; unknown tools, duplicate or
unsafe registrations, unsupported versions, malformed arguments, unknown
fields, invalid enums/IDs/timestamps, oversized values and trailing JSON fail
closed. No new runtime or dependency was introduced.

Evidence: `08-controlled-ai-tools-durable-research-agents/WP-08.01-tool-registry-and-json-schema-validation.md`.

## RD-2026-09-07-18 - External Phase 07 GO / Phase 08 authorization

- Date: 2026-09-07
- Phase: 07 accepted COMPLETE / GO; Phase 08 authorised / in progress
- Current work package: WP-08.01 — Tool registry and JSON-schema validation
- Decision authority: external GPT-5.6 Sol technical-lead review

External technical-lead decision: `GO PHASE 07`. Phase 07 evaluation, replay
and OOS capability passed its roadmap gate. This does not demonstrate a trading
edge; the accepted evidence remains `EXPLICIT_OOS_SINGLE_CASE_INSUFFICIENT_SAMPLE`
and `TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE`.

Phase 08 is now authorised through WP-08.08. Its bounded researcher must use
permitted read-only tools, durable bounded checkpoints, deterministic budgets,
provenance-safe memory and preserved recommendation/risk/execution gates. Phase
09 is not authorised or started.

## RD-2026-09-07-17 - Phase 07 exit demonstrated / external review pending

- Date: 2026-09-07
- Phase: 07 — Evaluation, Replay & Backtesting
- Work packages: WP-07.01 through WP-07.08
- Status: Exit condition demonstrated; external technical-lead review pending
- Decision authority: autonomous phase verification; no self-awarded phase GO

The complete Phase-07 chain now reproduces a frozen historical case's context
and recommendation, records post-decision outcomes without look-ahead, binds a
candidate logic version to explicit out-of-sample membership, applies frozen
cost assumptions and emits an operator-facing report. The proof deliberately
reports `EXPLICIT_OOS_SINGLE_CASE_INSUFFICIENT_SAMPLE` and
`UNKNOWN_INSUFFICIENT_SAMPLE_NO_EDGE_CLAIM`; it does not claim a trading edge.
Focused evaluation tests, the full Go suite, adversarial tests and repository
hygiene checks pass. No Phase-08 work has begun.

Evidence: `internal/modules/evaluation/phase07_exit_test.go` and the WP-07.01
through WP-07.08 evidence sections.

## RD-2026-09-07-16 - WP-07.08 internal verification

- Date: 2026-09-07
- Phase: 07 — Evaluation, Replay & Backtesting
- Work package: WP-07.08 — Operational replay tooling
- Status: Implemented / internally verified; Phase-07 exit proof pending
- Decision authority: autonomous package verification under authorised Phase-07 scope

WP-07.08 adds a deterministic operator-facing report binding a frozen case,
artifact replay, tracked outcome, adapter assessment, cost policy and explicit
out-of-sample protocol membership. It rejects tampered identities, incomplete
evidence and non-OOS cases. The report records an insufficient-sample/no-edge
claim conclusion and remains execution-disabled. Focused tests pass.

Evidence: `07-evaluation-replay-backtesting/WP-07.08-operational-replay-tooling.md`.

## RD-2026-09-07-15 - WP-07.07 internal verification

- Date: 2026-09-07
- Phase: 07 — Evaluation, Replay & Backtesting
- Work package: WP-07.07 — Walk-forward/out-of-sample protocol
- Status: Implemented / internally verified; WP-07.08 current
- Decision authority: autonomous package verification under authorised Phase-07 scope

WP-07.07 adds a deterministic walk-forward protocol bound to frozen benchmark
identities and explicit candidate/cost versions. It requires development,
validation and out-of-sample windows, decision-time containment, chronological
separation with purge/embargo, configuration freeze before OOS, and prohibits
OOS/final-holdout tuning. Focused leakage-negative tests pass.

Evidence: `07-evaluation-replay-backtesting/WP-07.07-walk-forward-out-of-sample-protocol.md`.

## RD-2026-09-07-14 - WP-07.06 internal verification

- Date: 2026-09-07
- Phase: 07 — Evaluation, Replay & Backtesting
- Work package: WP-07.06 — Transaction cost/slippage assumptions
- Status: Implemented / internally verified; WP-07.07 current
- Decision authority: autonomous package verification under authorised Phase-07 scope

WP-07.06 adds content-addressed transaction-cost and slippage assumptions and
deterministic hypothetical BUY/SELL cost calculations. Commission, spread,
slippage, impact and borrow components are explicit and arithmetic is checked.
Silent zero-friction baselines are rejected; explicit diagnostic baselines are
allowed. The result cannot create orders, fills or execution authority.

Evidence: `07-evaluation-replay-backtesting/WP-07.06-transaction-cost-slippage-assumptions.md`.

## RD-2026-09-07-13 - WP-07.05 internal verification

- Date: 2026-09-07
- Phase: 07 — Evaluation, Replay & Backtesting
- Work package: WP-07.05 — Traditional strategy backtesting adapter evaluation
- Status: Implemented / internally verified; WP-07.06 current
- Decision authority: autonomous package verification under authorised Phase-07 scope

WP-07.05 records a versioned adapter assessment without adding a new runtime or
dependency. The selected custom event-replay boundary is required to be
event-driven, point-in-time safe, reproducible, custom-data capable and
transaction-cost capable. Alternatives and operational/license burden remain
explicit; unsafe selection, ambiguous selection and missing rationale are
rejected. Focused tests pass and execution authority remains NONE.

Evidence: `07-evaluation-replay-backtesting/WP-07.05-traditional-strategy-backtesting-adapter-evaluation.md`.

## RD-2026-09-07-12 - WP-07.04 internal verification

- Date: 2026-09-07
- Phase: 07 — Evaluation, Replay & Backtesting
- Work package: WP-07.04 — Recommendation outcome tracking
- Status: Implemented / internally verified; WP-07.05 current
- Decision authority: autonomous package verification under authorised Phase-07 scope

WP-07.04 adds a separate content-addressed recommendation outcome artifact.
It records post-decision observation windows, explicit incomplete horizons,
benchmark-relative results and excursion bounds without creating trade or
execution state. Strict observation ordering, recording-time bounds and
invalidation-time look-ahead checks pass focused tests; original recommendation
identity is immutable and mutation is rejected.

Evidence: `07-evaluation-replay-backtesting/WP-07.04-recommendation-outcome-tracking.md`.

## RD-2026-09-07-11 - WP-07.03 internal verification

- Date: 2026-09-07
- Phase: 07 — Evaluation, Replay & Backtesting
- Work package: WP-07.03 — Model/prompt/algorithm version comparison
- Status: Implemented / internally verified; WP-07.04 current
- Decision authority: autonomous package verification under authorised Phase-07 scope

WP-07.03 adds controlled same-benchmark comparison artifacts for model, prompt,
system, algorithm and policy versions. It records every variant tried and
reported, requires complete case coverage, preserves explicit selection
criteria and rejects holdout-touch state. Focused tests pass; no inference or
model escalation was performed.

Evidence: `07-evaluation-replay-backtesting/WP-07.03-model-prompt-algorithm-version-comparison.md`.

## RD-2026-09-07-10 - WP-07.02 internal verification

- Date: 2026-09-07
- Phase: 07 — Evaluation, Replay & Backtesting
- Work package: WP-07.02 — Frozen benchmark registry
- Status: Implemented / internally verified; WP-07.03 current
- Decision authority: autonomous package verification under authorised Phase-07 scope

WP-07.02 adds an immutable versioned benchmark registry with dataset/content
hashes, split classification, fixed cases/timestamps, allowed evidence vintages,
complete configuration freeze and provenance. Changed content cannot silently
reuse a frozen identity; final holdout tuning is rejected. Focused tests pass.

Evidence: `07-evaluation-replay-backtesting/WP-07.02-frozen-benchmark-registry.md`.

## RD-2026-09-07-09 - WP-07.01 internal verification

- Date: 2026-09-07
- Phase: 07 — Evaluation, Replay & Backtesting
- Work package: WP-07.01 — Historical event/research replay engine
- Status: Implemented / internally verified; WP-07.02 current
- Decision authority: autonomous package verification under authorised Phase-07 scope

WP-07.01 adds a versioned frozen historical-case boundary with explicit
decision-time knowability, exact Phase-06 artifact reconstruction and fail-closed
future-evidence checks. Focused success and leakage-negative tests pass. No
fresh inference, current-state lookup, outcome mutation or trading state is
introduced.

Evidence: `07-evaluation-replay-backtesting/WP-07.01-historical-event-research-replay-engine.md`.

## RD-2026-09-07-08 - External Phase 06 GO / Phase 07 authorization

- Date: 2026-09-07
- Phase: 06 — Research & Recommendation Engine; 07 — Evaluation, Replay & Backtesting
- Status: Phase 06 accepted COMPLETE / GO; Phase 07 authorised / in progress
- Decision authority: explicit external GPT-5.6 Sol technical-lead decision supplied by the user

External GPT-5.6 Sol accepted the complete Phase-06 handover, exit proof,
full verification, security evidence and adversarial review with `GO PHASE 06`.
Phase 06 is recorded COMPLETE / GO. Phase 07 is authorized through WP-07.08,
beginning with WP-07.01. Phase 07 is validation-first and must not create
approval, order, trade, fill or live execution state.

Evidence: the supplied external Phase-06 review decision and
`06-research-recommendation-engine/PHASE-06-INTERNAL-VERIFICATION.md`.

## RD-2026-09-07-06 - Phase 06 exit demonstration and adversarial review

- Date: 2026-09-07
- Phase: 06 — Research & Recommendation Engine
- Status: Exit demonstrated; external technical-lead phase review pending
- Decision authority: internal autonomous verification; no external GO awarded

WP-06.01 through WP-06.07 are implemented and locally committed. The bounded
exit proof demonstrates reproducible evidence-linked research-only WATCH output
with thesis, bull/counter evidence, contradiction, unknown, invalidation,
freshness and Phase-05 quant context. The dedicated adversarial phase review
passed after correcting timestamp normalization, incomplete content identities,
retry enforcement and read-model cross-reference validation. Phase 06 remains
outside Phase 07 until external `GO PHASE 06`.

Evidence: `06-research-recommendation-engine/PHASE-06-INTERNAL-VERIFICATION.md`.

## RD-2026-09-07-05 - Phase 05 GO / Phase 06 authorization

- Date: 2026-09-07
- Phase: 05 — Deterministic Quant Core; 06 — Research & Recommendation Engine
- Status: Phase 05 accepted COMPLETE / GO; Phase 06 authorized / in progress
- Decision authority: explicit external GPT-5.6 Sol technical-lead decision supplied by the user

External GPT-5.6 Sol accepted the Phase-05 handover, deterministic exit proof,
full verification, and adversarial phase review with `GO PHASE 05`. Phase 05 is
recorded COMPLETE / GO. The same decision authorizes the sequential Phase-06
work-package scope beginning at `WP-06.01 — Define evidence packet contract`.
Phase 06 remains research-only and must not create approvals, orders, trades,
fills, or live execution authority.

Evidence: `05-deterministic-quant-core/PHASE-05-INTERNAL-VERIFICATION.md` and
the supplied external Phase-05 review decision.

## RD-2026-09-07-01 - Zero-cost market evidence closure

- Date: 2026-09-07
- Phase: 03 - Core Financial Evidence
- Status: exit condition demonstrated; awaiting technical-lead GO
- Decision authority: technical-lead roadmap change supplied by the user; final Phase-03 decision remains open

Financial Datasets remains an accepted provider option, but its configured
development credential returned HTTP 401 and Phase-03 acceptance will not be
made contingent on paying recurring market-data fees. The bounded closure
evaluated the existing Alpaca path, added the smallest raw-first canonical
historical-bars path, and demonstrated real AAPL SIP market evidence together
with real SEC Apple and Treasury evidence in `jax.phase03_exit_packet/v1`.
Alpaca Basic is a development evidence source only; production and serious
backtesting qualification remain open. Phase 04 has not started.

Evidence: `../evidence/PHASE-03-FREE-MARKET-DATA-CLOSURE.md` and
`../evidence/PHASE-03-EXIT-GATE.md`.

## RD-2026-09-04-03 - WP-03.06 FINAL GO and Phase-03 exit-gate closure demonstration

- Date: 2026-09-04
- Phase: 03 - Core Financial Evidence
- Status: WP-03.06 accepted COMPLETE / GO; Phase-03 exit gate under review
- Decision authority: independent technical-lead FINAL GO supplied for WP-03.06; Phase-03 exit decision remains open

WP-03.06 has received independent technical-lead **FINAL GO**. The bounded
exit-gate verification retained real Treasury and Cboe context evidence, but
the accepted AAPL market path failed with authentication and the accepted SEC
path lacked the required automated-client identity. The fail-closed gate
therefore reports `PHASE 03 EXIT CONDITION NOT YET DEMONSTRATED`. Phase 03
remains in progress and Phase 04 has not started.

Evidence: `../evidence/PHASE-03-EXIT-GATE.md`.

## RD-2026-09-04-02 - WP-03.05 FINAL GO and WP-03.06 implementation handover

- Date: 2026-09-04
- Phase: 03 - Core Financial Evidence
- Status: WP-03.05 accepted COMPLETE / GO; WP-03.06 implemented and awaiting independent review
- Decision authority: independent technical-lead final decision supplied for WP-03.05; WP-03.06 remains subject to technical-lead decision

WP-03.05 Treasury / EIA / CBOE / CFTC Source Evaluation and First Approved
Integrations has received **FINAL GO**. WP-03.06 Evidence-quality
Cross-source Checks is implemented under its bounded package scope and is
handed over for independent review. Phase 03 remains in progress and Phase 04
has not started.

Evidence: `../evidence/WP-03.05-TREASURY-EIA-CBOE-CFTC-SOURCE-EVALUATION.md` and
`../evidence/WP-03.06-EVIDENCE-QUALITY-CROSS-SOURCE-CHECKS.md`.

## RD-2026-09-04-01 - WP-03.04 FINAL GO and WP-03.05 review handover

- Date: 2026-09-04
- Phase: 03 - Core Financial Evidence
- Status: WP-03.04 accepted COMPLETE / GO; WP-03.05 implemented and awaiting independent review
- Decision authority: independent technical-lead final decision supplied for WP-03.04; WP-03.05 remains subject to technical-lead decision

WP-03.04 Economic Release / Calendar Ingestion has received **FINAL GO**.
WP-03.05 Treasury / EIA / CBOE / CFTC Source Evaluation and First Approved
Integrations is the current implementation package. WP-03.06 remains NOT
STARTED. See `../evidence/WP-03.04-ECONOMIC-RELEASE-CALENDAR-EVIDENCE.md`,
`../evidence/WP-03.05-TREASURY-EIA-CBOE-CFTC-SOURCE-EVALUATION.md`, and
`Docs/ROADMAP.md`.

## RD-2026-08-21-01 - Phase 00 GO

- Date: 2026-08-21
- Phase: 00 - Current Issuer & Asset Resolution
- Status: Accepted
- Decision authority: explicit technical-lead decision supplied for close-out

### Decision

Phase 00 is **GO**. Jax has sufficiently demonstrated that the current architecture can reliably identify the affected issuer/instrument or safely remain unresolved:

`Event -> typed causal attribution -> deterministic policy -> DIRECT / PROXY / UNRESOLVED -> deterministic resolver`

The decision is supported by the unseen Luna Generalization gate pass, Luna r3 repeatability at 46/48 (95.83%), all frozen retention gates passing, zero incorrect deterministic ticker/rule resolutions, and zero safety/persistence violations. Terra t1 independently achieved 47/48 semantic correctness and 5/6 PROXY recall with all six retention gates passing, and is classified `MATERIALLY BETTER`.

Residual limitations are accepted: Luna is weaker on difficult macro/proxy attribution; Terra is one-shot only; and Terra case 042 selected `US_RATES_CATEGORY` instead of `FEDERAL_RESERVE_OFFICIAL`. They do not invalidate GO.

Evidence: `../evidence/PHASE-00-ISSUER-RESOLUTION-CLOSEOUT.md`.

## RD-2026-08-21-02 - Hosted-model policy after Phase 00

- Date: 2026-08-21
- Status: Accepted

Luna (`gpt-5.6-luna`) remains the default Jax development/runtime hosted model. Terra (`gpt-5.6-terra`) is a validated higher-capability option that may be reconsidered for high-value/ambiguous escalation or when economics justify it.

No runtime default switch, escalation architecture, further Terra experiment, Terra repeatability, Sol challenger, additional model comparison, or Phase 00 prompt/schema tuning is authorized. Model evaluation is closed.

## RD-2026-08-21-03 - Roadmap reconciliation and next package

- Date: 2026-08-21
- Status: Accepted

WP-00.01, WP-00.02, and the expanded WP-00.03 sequence are complete. WP-00.04 was the only formally unclosed Phase 00 package; implementation evidence reduced it to a gate/closure and compact evidence-preservation action, which is completed by this close-out. No Phase 00 item remains.

No **ROADMAP CHANGE** is required. The implementation evolved beyond the roadmap's terse WP-00.03 wording, but it proved rather than invalidated the stated Phase 00 outcome and preserves the sequencing into canonical contracts/provenance/audit.

The exact first incomplete authorized-by-roadmap package is `WP-01.01 - Inventory existing Jax domain contracts before adding new ones`. It is not authorized for implementation by this decision and has not started. See `NEXT-WORK-PACKAGE.md`.

## RD-2026-08-24-01 - WP-01.04 GO

- Date: 2026-08-24
- Phase: 01 - Canonical Contracts, Provenance & Audit
- Work package: WP-01.04 - Define replay/audit event model and compatibility strategy
- Status: Accepted
- Decision authority: explicit independent technical-lead decision supplied for close-out

### Decision

WP-01.04 is **GO**. The accepted implementation establishes immutable audit history, current-projection separation, deterministic replay manifests and verification, explicit compatibility classifications, V1/V2 fail-closed translation semantics, and the normative cross-runtime canonical-byte specification without changing the accepted WP-01.02/WP-01.03 architecture.

Evidence: `../evidence/WP-01.04-REPLAY-AUDIT-COMPATIBILITY.md`.

## RD-2026-08-24-02 - Phase 01 GO

- Date: 2026-08-24
- Phase: 01 - Canonical Contracts, Provenance & Audit
- Status: Accepted
- Decision authority: explicit independent technical-lead phase-gate decision supplied for close-out

### Decision

Phase 01 is **GO PHASE 01**. WP-01.01 established the source-backed contract inventory, WP-01.02 the canonical domain vocabulary, WP-01.03 immutable provenance/version/content identities, and WP-01.04 immutable audit/replay/compatibility semantics. All four packages were independently reviewed; no unresolved NO-GO or blocking CONDITIONAL GO remains.

The exit condition is demonstrated by the deterministic in-memory reconstruction chain in the WP-01.04 evidence: a representative Recommendation traces to immutable inputs, source/provider, versions, and timestamps without transient logs. The proof requires no database, provider, inference, or trading mutation.

The following accepted debts are non-blocking: broad production adoption of the canonical V2/audit model; append-only persistence enforcement; a non-Go canonical-byte conformance implementation; the inability to classify stochastic model re-inference as exact replay; unrelated repository-wide gofmt/lint debt; and the requirement that future adapters preserve immutable raw/provider evidence.

Phase 02 is eligible but not started. Its next package is `WP-02.01 - Provider registry/capability contract`, which awaits separate technical-lead package authorization. See `NEXT-WORK-PACKAGE.md`.

Evidence:

- `../evidence/WP-01.01-JAX-DOMAIN-CONTRACT-INVENTORY.md`
- `../evidence/WP-01.02-CANONICAL-DOMAIN-CONTRACTS.md`
- `../evidence/WP-01.03-CANONICAL-PROVENANCE-IDENTITIES.md`
- `../evidence/WP-01.04-REPLAY-AUDIT-COMPATIBILITY.md`

## RD-2026-08-24-03 - WP-02.01 GO

- Date: 2026-08-24
- Phase: 02 - Data Platform & Provider Architecture
- Work package: WP-02.01 - Provider registry/capability contract
- Status: Accepted
- Decision authority: explicit technical-lead decision supplied with WP-02.03 authorization

### Decision

WP-02.01 is **GO**. The accepted implementation establishes stable provider identity, capability-driven canonical output declarations, explicit provider-raw representation/schema boundaries, deterministic registry behavior, static support semantics, and bounded future runtime-state attachment without provider calls or runtime migration.

Evidence: `../evidence/WP-02.01-PROVIDER-REGISTRY-CAPABILITY-CONTRACT.md`.

## RD-2026-08-24-04 - WP-02.02 GO

- Date: 2026-08-24
- Phase: 02 - Data Platform & Provider Architecture
- Work package: WP-02.02 - Raw payload persistence/reference policy
- Status: Accepted
- Decision authority: explicit technical-lead decision supplied with WP-02.03 authorization

### Decision

WP-02.02 is **GO**, including the corrective identity-namespace commit `9b022fff3cd8d0f2cb08f2b104e3910a3ea4f573`. Accepted raw-payload acquisition identities use `rpa_`; the Phase 01 replay-manifest namespace remains `rpl_`. The implementation establishes exact-byte hashing, immutable acquisition/content separation, provider/capability/schema binding, verified storage-port reads, retention/redistribution metadata, and deterministic in-memory proof without production persistence or later Phase 02 behavior.

Evidence: `../evidence/WP-02.02-RAW-PAYLOAD-PERSISTENCE-REFERENCE-POLICY.md`.

## RD-2026-08-24-05 - WP-02.03 GO

- Date: 2026-08-24
- Phase: 02 - Data Platform & Provider Architecture
- Work package: WP-02.03 - Normalization and validation pipeline
- Status: Accepted
- Decision authority: explicit technical-lead decision supplied with WP-02.04 authorization

### Decision

WP-02.03 is **GO**. The accepted implementation establishes deterministic provider-owned normalization, exact capability/raw-schema/canonical-target routing, typed stage failures, canonical and raw-provenance validation, loss/omission metadata, strict normalizer identity/version binding, storage-port-to-normalization proof, and repeatability without runtime provider migration or later Phase 02 behavior.

Evidence: `../evidence/WP-02.03-NORMALIZATION-VALIDATION-PIPELINE.md`.

## RD-2026-08-24-06 - WP-02.04 GO

- Date: 2026-08-24
- Phase: 02 - Data Platform & Provider Architecture
- Work package: WP-02.04 - Freshness/TTL/last-known-good semantics
- Status: Accepted
- Decision authority: explicit technical-lead decision supplied with WP-02.05 authorization

### Decision

WP-02.04 is **GO**. The accepted implementation establishes provider-neutral versioned freshness/LKG policies, explicit canonical timestamp authority and evaluation time, exact TTL/expiry states, deterministic same-key LKG qualification/selection, visible fallback identity/age/reason, future-skew and lifecycle fail-closed behavior, and an Observation V2 proof without migrating runtime providers or conflating provider health with datum freshness.

Evidence: `../evidence/WP-02.04-FRESHNESS-TTL-LAST-KNOWN-GOOD-SEMANTICS.md`.

## RD-2026-08-26-01 - WP-02.06 GO

- Date: 2026-08-26
- Phase: 02 - Data Platform & Provider Architecture
- Work package: WP-02.06 - Data source qualification registry
- Status: Accepted
- Decision authority: explicit independent technical-lead decision supplied for close-out

### Decision

WP-02.06 is **GO**. The accepted implementation establishes evidence-backed, role-specific source qualification with separate source/provider-path identity, authority class, intended-use permissions, licensing and retention rights, historical reliability, cost, coverage, typed conditions, non-overridable hard disqualifiers, exact policy/assessor identity, explicit review/expiry, and immutable historical decisions. It does not collapse these dimensions into a trust score or source-selection engine.

Evidence: `../evidence/WP-02.06-DATA-SOURCE-QUALIFICATION-REGISTRY.md`.

## RD-2026-08-26-02 - Phase 02 GO

- Date: 2026-08-26
- Phase: 02 - Data Platform & Provider Architecture
- Status: Accepted
- Decision authority: explicit independent technical-lead phase-gate decision supplied for close-out

### Decision

Phase 02 is **GO PHASE 02**. WP-02.01 established the stable provider registry/capability boundary; WP-02.02 exact-byte raw acquisition identity and forensic persistence/reference semantics; WP-02.03 deterministic raw-to-canonical normalization with immutable provenance; WP-02.04 explicit freshness, TTL, and deterministic LKG/fallback semantics; WP-02.05 bounded retry, rate-limit, backoff, and provider-health semantics while preserving the exact raw-data path; and WP-02.06 evidence-backed role-specific source qualification without a universal trust score or source-selection engine. All six packages were independently reviewed. No unresolved NO-GO or blocking CONDITIONAL GO remains.

The Phase 02 exit condition is demonstrated by the executable synthetic chain:

```text
provider capability
    -> operational acquisition
    -> exact-byte raw persistence
    -> deterministic normalization
    -> canonical validated data + provenance
    -> freshness
    -> provider health
    -> source qualification
```

A second synthetic adapter using a different provider identity and raw schema produces the same provider-neutral downstream canonical research projection. A provider can therefore change behind the stable adapter boundary without changing research-facing canonical logic.

The following accepted debts are non-blocking: provider/raw/canonical contracts are not yet broadly adopted by production consumers; the initial in-memory raw proof store was subsequently closed by corrective WP-02.07 with a durable PostgreSQL `RawPayloadStore`; production freshness TTLs and operational retry/rate-limit/health thresholds require evidence-backed configuration; no real provider/source qualification catalogue exists; real-provider licensing, reliability, and cost facts remain unassessed; no non-Go canonical-byte conformance implementation exists; and unrelated repository-wide gofmt/lint debt remains.

Phase 03 is now **in progress**. WP-03.01 Market price/OHLCV, WP-03.02
SEC/EDGAR/XBRL, and WP-03.03 FRED/ALFRED macro observations and vintages are
accepted **COMPLETE / GO** in their retained evidence handovers. The next
package is `WP-03.04 — Economic Release / Calendar Ingestion`; it is not
started and still requires separate technical-lead package authorization. See
`Docs/ROADMAP.md` and `NEXT-WORK-PACKAGE.md`.

Evidence:

- `../evidence/WP-02.01-PROVIDER-REGISTRY-CAPABILITY-CONTRACT.md`
- `../evidence/WP-02.02-RAW-PAYLOAD-PERSISTENCE-REFERENCE-POLICY.md`
- `../evidence/WP-02.03-NORMALIZATION-VALIDATION-PIPELINE.md`
- `../evidence/WP-02.04-FRESHNESS-TTL-LAST-KNOWN-GOOD-SEMANTICS.md`
- `../evidence/WP-02.05-RATE-LIMIT-RETRY-BACKOFF-HEALTH-INSTRUMENTATION.md`
- `../evidence/WP-02.06-DATA-SOURCE-QUALIFICATION-REGISTRY.md`

## RD-2026-09-07-01 - Phase 03 GO

- Date: 2026-09-07
- Phase: 03 - Core Financial Evidence
- Status: Accepted
- Decision authority: independent technical-lead phase-gate decision

### Decision

Phase 03 is **GO PHASE 03**. The exit condition was demonstrated with a deterministic source-linked AAPL/Apple evidence packet containing real accepted market, company and macro/context evidence with raw acquisition provenance and no model-memory dependency.

The final development market-evidence path used Alpaca as an explicit zero-cost development source. Financial Datasets remains an accepted provider option, but Phase-03 acceptance was not made contingent on paid development access.

Remaining serious-backtesting market-data qualification, production licensing, corporate-action policy and historical knowability are future validation concerns and do not invalidate Phase-03 GO. Phase 04 is authorised at WP-04.01.

## RD-2026-09-07-02 - Autonomous phase development mode

- Date: 2026-09-07
- Status: Accepted
- Applies from: Phase 04 onward

### Decision

External technical-lead review moves from every work package to the phase boundary. Codex still implements packages sequentially, verifies/self-reviews them, and preserves bounded commits/evidence, but may continue automatically inside the authorised phase.

Hard stops include roadmap changes, paid dependency/account decisions, missing/rejected credentials, consequential architecture decisions, destructive data/Git actions, and live trading/execution authority changes.

Model policy: GPT-5.6 Luna for default implementation; GPT-5.6 Sol for consequential decisions and phase review; GPT-5.6 Terra as optional bounded escalation when Luna materially struggles.

## RD-2026-09-07-04 - Phase 04 GO / Phase 05 authorization

- Date: 2026-09-07
- Status: Accepted
- Decision authority: external GPT-5.6 Sol technical-lead review

### Decision

Phase 04 — World Monitor Intelligence — is accepted `COMPLETE / GO`. The exact
Phase-04 exit condition and supplied phase handover were accepted. Phase 05 —
Deterministic Quant Core — is authorised/in progress beginning at
`WP-05.01 — Quant service/library boundary and versioned request/response contract`.

## RD-2026-09-07-03 - Mandatory adversarial phase review

- Date: 2026-09-07
- Status: Applied
- Applies from: Phase 05 onward

### Decision

Before any Phase-05-or-later handover for external technical-lead review, Codex
must perform a dedicated adversarial phase review separate from package-level
self-review. It must inspect the actual phase diff, challenge correctness,
provenance, temporal semantics, persistence, contracts, tests, architecture,
security/cost and trading safety, test the tests, and independently verify
important expected values. All blocking findings must be corrected and
reverified or reported as an explicit `NO-GO` blocker. A GO recommendation must
state `Blocking findings remaining: 0` and `Adversarial phase review: PASS`.

## RD-2026-09-07-06 - Phase 09 GO / migration collision remediation

- Date: 2026-09-07
- Phase: 09 - Portfolio Intelligence & Deterministic Risk
- Status: Accepted and remediated
- Decision authority: external GPT-5.6 Sol technical-lead review

### Decision

Phase 09 remains **COMPLETE / GO**. A duplicate-version defect was found before
Phase 10: the single active golang-migrate stream contained historical
`000055_raw_payload_storage` and `000056_world_monitor_pull_pages` alongside the
new Phase-09 migrations using 000055 and 000056. The approved forward-only
remediation preserved the historical files byte-for-byte and moved the new
Phase-09 migrations to the next monotonically increasing unused versions:
`000059_portfolio_snapshots` and `000060_portfolio_risk_decisions`.

The persistent-application preflight found no evidence that either Phase-09
migration had been applied. The repository-linked migration logs ended at
version 53 before the Phase-09 commits; configured endpoints were unavailable
or rejected credentials, and no database was modified. A permanent
directory-wide migration registry invariant now enforces unique versions,
deterministic filenames/order, and matching up/down pairs. The complete
migration and Phase-09 verification passed. `MIGRATION BLOCKER RESOLVED`.

Phase 10 is **AUTHORISED / IN PROGRESS** beginning at WP-10.01. Scientific
status remains **TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE**.

## RD-2026-09-07-07 - Phase 10 exit demonstrated / external review pending

- Date: 2026-09-07
- Phase: 10 - Workflow, HITL & Operational Safety
- Status: Internally complete; external phase-gate review pending
- Decision authority: Codex internal verification only; no GO self-awarded

### Decision record

WP-10.01 through WP-10.07 are implemented in bounded commits and the exact
Phase-10 exit condition is demonstrated by
`internal/modules/workflow/phase10_exit_test.go` plus the focused negative,
recovery, audit, breaker, health and concurrency tests. The bounded workflow
records explicit deterministic transitions from an accepted/amended Phase-09
risk artifact through explicit human paper-only confirmation to an immutable
inert `PAPER_INTENT`. It cannot create a paper/live broker order, broker call,
trade, fill, approval outside the explicit HITL contract, or portfolio
mutation.

The Phase-09 migration collision remediation remains part of the accepted
baseline: historical 000055/000056 migrations are unchanged, new Phase-09
migrations are 000059/000060, and the complete-stream registry invariant
passes. Phase 09 remains COMPLETE / GO. The scientific status remains
**TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT SAMPLE**. Phase 11 is NOT
STARTED.
## RD-2026-09-08-11 - HYP-EVENT-001A SEC evidence extension and hosted-inference cost gate

- Date: 2026-09-08
- Phase: 12 - Advanced Quant Research
- Status: Evidence extension complete; external cost decision pending
- Decision authority: external GPT-5.6 Sol roadmap-change approval; no hosted inference authorized by this record

### Decision record

The approved roadmap change extended the immutable HYP-EVENT-001A research
panel with SEC accession-time evidence packets. The accepted parent dataset
`hyp-event-001a-sec-8k-alpaca-sip-2016-2025-v1` remains unchanged and retains
manifest SHA-256
`db2b454793e50f28c34c9c2a7d91f798741936528752de7bd27cb52072a4864d`. The new
private derived dataset is
`hyp-event-001a-sec-8k-alpaca-sip-2016-2025-evidence-v2` with manifest SHA-256
`967dfcb18eff8b6a3f4ec39fedd3898537446a284dc2a868631926dc1f41408c`.

The acquisition rule retains each accession's complete index inventory,
primary Form 8-K and all same-accession textual documents selected by stable
metadata, while excluding non-semantic support files and duplicate aggregate
submission text. Coverage is 1,059 packets, 1,059 primary matches, 10,731
inventory rows, 2,418 selected raw documents and zero retrieval failures. 2024
is semantically sealed until classifier freeze; 2025 is hash/inventory-only
and sealed for semantic and outcome use.

The frozen direction contract is `jax.hyp-event-001a.direction/v1` with prompt
identity
`d0b09acf412eb73ff97fc2e9bd5f4fc609dc57d5bd1dbd17d57f986b158d4143`. No
direction labels, OOS outcomes or hosted inference were run. Current spend is
$0. The measured 2016-2024 planning workload is 939 events and 12,859,803
estimated evidence tokens; one event crosses the provider's >272K pricing
threshold, none exceeds context, base cost is approximately $2.98 and the
one-retry maximum envelope is approximately $5.97. A separate external cost
decision is required before any paid call.

Scientific status remains **TRADING EDGE NOT DEMONSTRATED / INSUFFICIENT
SAMPLE**. Phase 13 remains NOT STARTED.
