# Roadmap Decision Log

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
