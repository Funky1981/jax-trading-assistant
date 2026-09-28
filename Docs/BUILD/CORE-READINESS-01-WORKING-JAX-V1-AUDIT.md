# CORE-READINESS-01 — Working Jax v1 / Scientific Readiness Audit

Status: **AUDIT COMPLETE / BLOCKED BEFORE HYPOTHESIS DESIGN**

Reviewed baseline: `337e914e08140c066e367ba43f41ba603620afc2`.

This is an implementation and evidence audit, not a strategy decision. No
strategy was selected, no pilot was created or activated, and no normal Jax
database or runtime was mutated or started.

## Executive conclusion

The repository contains substantial, tested components for an event-driven
exploratory PAPER lifecycle. OPS-02B proved the corrected local PAPER runtime
could start safely, consume genuine World Monitor events, persist event
provenance, and produce real-origin `NO_TRADE` decisions while creating zero
economic activity. OPS-01 proved a synthetic/disposable-PostgreSQL full-loop
exercise across production components. Neither proves that a real candidate,
after authenticated human approval and a production risk decision, naturally
reaches the account-scoped PAPER entry worker using genuine, temporally safe
market prices.

Three blockers prevent responsibly beginning controlled economic-hypothesis
testing:

1. The canonical candidate approval endpoint records its actor from
   `X-User-ID`; it does not bind that actor to validated JWT claims. A caller
   can spoof the durable approval actor.
2. Candidate approval, candidate-level risk review, portfolio workflow risk,
   and the exploratory approved-entry queue are separate seams. No production
   call connects the candidate approval action to a server-built
   workflow/risk/entry request. The queue accepts a caller-supplied snapshot;
   content hashes establish consistency, not authenticated provenance.
3. The full-loop proof uses synthetic provider and market fixtures. OPS-02B
   explicitly disabled the market ingester and produced no orders, fills,
   ledger activity, or lifecycle. The production review source reads a
   persisted candle close as a modeled observation, without enforcing a
   timeframe, source allowlist, upper time bound, or freshness limit in that
   query. Genuine, appropriately timestamped market-price use in the corrected
   runtime is unproven, and the current query is insufficiently bounded for
   scientific use.

Technical verdict: **BLOCKED_BEFORE_HYPOTHESIS_DESIGN**. The minimum next work
is a bounded seam-and-proof correction, not an architecture rewrite. This
verdict does not authorize selecting or implementing a strategy, creating a
pilot, or activating runtime collection.

## Evidence vocabulary

Classifications used literally:

`PROVEN_REAL_RUNTIME`, `PROVEN_SYNTHETIC_RUNTIME`, `TESTED_INTEGRATION`,
`TESTED_UNIT_ONLY`, `IMPLEMENTED_UNPROVEN`, `MISSING`,
`NOT_REQUIRED_YET`.

`PROVEN_REAL_RUNTIME` is limited to the bounded claims OPS-02B actually
observed. OPS-01 is synthetic/disposable integration evidence, not real
economic activity. Historical implementation records, documentation claims,
and tests are not promoted to real-runtime proof.

## Working Jax v1 capability matrix

| Stage | Canonical current code path / input-output / persistence | Tests and strongest evidence | Failure handling / unresolved dependency | Readiness |
|---|---|---|---|---|
| Data — event/evidence | `startWorldMonitorPullWorker` → `worldMonitorPullWorker.cycle` → `eventdecisions.Replayer` → `world_monitor_pull_pages`, inbox and `genuine_event_decisions`. Input is a genuine provider page with source/native identity, raw page bytes, cursor and event timestamps; output is normalized inbox, provenance and versioned deterministic decision. | Pull-worker and eventdecision tests; `TestOPS01OperationalReadinessProof`; OPS-02B traced a genuine CNBC event through retained pull page, inbox, normalized event and real-origin `NO_TRADE`: `PROVEN_REAL_RUNTIME` for that bounded trace. | Configuration, safety, fetch, parse and persistence failures are surfaced in worker health/logs; cursor replay is idempotent. Real proof covers intake and abstention, not promotion of a genuine event into an economic candidate. | `PROVEN_REAL_RUNTIME` (bounded intake claim) |
| Data — market/price | `startMarketIngester` / `ingestQuote` / `ingestCandles`; optional `collectGenuineCandles`; runtime review input: `postgresExploratoryReviewSource.LoadReviewObservation`. Quotes/candles can persist with source and candle semantics. Review selects `timestamp, close, source` from `candles`, models bid/ask from close and labels `MODELED_CANDLE_CLOSE`; entry tick is supplied in queued `EntryRequest`. | Provider/unit and collection tests; OPS-01 uses `ops01-provider-substitute`, `ops01-market-substitute`, and marks actual quote unavailable: `TESTED_INTEGRATION` for substitute use only. OPS-02B disabled market ingester and had no economic activity. | Runtime review SQL has no timeframe, source allowlist, `timestamp <= now`, or freshness bound. Candle close is not bid/ask. IB bridge is the normal ingester dependency; genuine collection helpers exist, but corrected-runtime genuine price use was not demonstrated. | `IMPLEMENTED_UNPROVEN`; genuine corrected-runtime price path is a `CORE_HARD_BLOCKER` |
| Analysis | `worldMonitorPullWorker` deterministic eventdecision plus `worldMonitorOpportunityPromoter.promoteRow` / `promoteSymbol`, candidate evidence and chart confirmation. Event/source evidence and affected-ETF resolution may become a candidate/evidence projection; weak/unknown cases remain blocked or no-trade. | Deterministic event tests, promoter tests and OPS-01 integration with synthetic provider: `TESTED_INTEGRATION`; OPS-02B real-origin event decision proves only its `NO_TRADE`: `PROVEN_REAL_RUNTIME`. | Promotion requires eligible strategy instance and usable persisted candles. No evidence shows genuine events producing candidates under reviewed real price input. JaxMind is not required by this deterministic contract. | `TESTED_INTEGRATION` |
| Decision | `eventdecisions.Replayer` persists current decisions; promoter links eligible inbox decisions to `strategy_signals` and `candidate_trades`; `GenerateCandidate` supports `NO_TRADE`, `WATCH`, `CANDIDATE`. | OPS-02B real-world `NO_TRADE`; unit/golden decision tests; OPS-01 synthetic promotion: `PROVEN_REAL_RUNTIME` for real-origin abstention, `TESTED_INTEGRATION` for promotion. | Replay and identity are tested. The older `internal/decisioning/*` pipeline is not called by the `cmd/trader` World Monitor production path and must not be combined with it implicitly. | `TESTED_INTEGRATION` for candidate route; real abstention `PROVEN_REAL_RUNTIME` |
| Risk | Promoter candidate risk via `ReviewCandidateRisk` / `PersistRiskReview`; separate `portfoliorisk` workflow decision consumed by `exploratorypaper.Runtime.processEntry`. | Candidate risk tests and promoter tests: `TESTED_INTEGRATION`. OPS-01 `ops01SyntheticRisk` constructs a `Synthetic: true` portfolio and fixture policy: `PROVEN_SYNTHETIC_RUNTIME` within the disposable full-loop test. | No production adapter turns persisted candidate risk plus current account snapshot into the portfolio risk decision required by workflow/entry. Promoter records proof equity assumption and absent slippage as zero. | Production risk binding `MISSING`; components `TESTED_INTEGRATION` / `PROVEN_SYNTHETIC_RUNTIME` |
| Human approval | `/api/v1/approvals/{candidateId}/approve|reject` → `approvals.Service.Decide`; separate `workflow.Store` confirmation and exploratory exit routes. Candidate decisions and paper-ticket readiness persist; workflow stores risk binding, confirmation, transition audit and paper intent. | API/service tests and OPS-01 synthetic workflow tests: `TESTED_INTEGRATION`. Exit-decision handler uses JWT context and has spoof/missing-claim tests. | Candidate handler calls `actorFromRequest`, which reads caller-controlled `X-User-ID` (or `anonymous`) rather than JWT context. Route is JWT-protected, but caller can claim another actor. Candidate approval does not create the workflow/entry queue record. | UI/API exists; trustworthy candidate approval provenance is a `CORE_HARD_BLOCKER` |
| Paper trade | `POST /api/v1/exploratory-paper/entry-queue` → `QueueApprovedEntry` → `LoadApprovedEntries` → `Runtime.RunEntryCycle` → in-process `PaperVenue` → durable orders/fills/ledger/lifecycle. Runtime rehydrates canonical event decision/evidence and checks account, mode, risk, workflow, market tick and calendar. | Runtime, Postgres restart/account-scope tests and OPS-01: `TESTED_INTEGRATION`; OPS-01 exercises simulated venue without IB. | No production caller links candidate approval to `entry-queue`/`QueueApprovedEntry`. Queue accepts caller-supplied full `EntryRequest`; validation checks consistency and public content identity, not authenticated human or canonical candidate approval provenance. This is a manual/untrusted handoff. | Runtime `TESTED_INTEGRATION`; candidate-to-PAPER transition `DISCONNECTED` / `CORE_HARD_BLOCKER` |
| Monitor | `startExploratoryPaperReviewWorker` → `Runtime.RunDueReviews` → `LoadReviewObservation` → `EvaluateExit`, checkpoint/review persistence. Schedule is one review per configured trading session. | Runtime-loop, lifecycle and Postgres tests; OPS-01 restart/full-loop substitute observation: `TESTED_INTEGRATION`. | Worker polls each minute. Missing observations are marked unavailable and do not create approval. Production candle lookup lacks explicit source/timeframe/freshness/upper-bound constraints. Review-unavailable persistence error is ignored at call site. | `TESTED_INTEGRATION` with substitute data; genuine-price monitoring `IMPLEMENTED_UNPROVEN` |
| Exit | Review/exit evaluation → durable `EXIT_RECOMMENDED` → protected JWT-bound exit decision endpoint → persisted approval/rejection → simulated order/fill. | PAPER-02R unit/Postgres tests and OPS-01 authenticated API plus restart integration: `TESTED_INTEGRATION` (OPS-01 inputs remain synthetic). | Missing approval remains pending; rejection persists and creates no fill. Decision replay and persistence are tested. Production automation still depends on unproven market input. | `TESTED_INTEGRATION` |
| Outcome | `BuildOutcomeFromFillsWithCoverage` → `papertrading.Reconcile` → `PersistOutcome` transaction. Outcome binds fills, policy/cost versions, gross/net P&L, exit reason/timestamps and excursion coverage. | Unit/Postgres tests; OPS-01 verifies unique entry/exit orders, fills and ledger events: `TESTED_INTEGRATION`. | Spread/slippage are embedded in executed prices and attributed; commission is deducted once. MFE/MAE require validated coverage; OPS-01 path is incomplete fixture data, not genuine observed prices. | `TESTED_INTEGRATION`; real-data economic outcome `IMPLEMENTED_UNPROVEN` |

## Event-data and analysis path distinctions

The canonical event route for the first Working Jax v1 experiment is:

```text
World Monitor pull worker
  → retained raw page + cursor
  → normalized research inbox
  → deterministic genuine_event_decisions
  → opportunity scanner / World Monitor promoter
  → strategy signal + candidate/evidence/risk-review projection
  → candidate approval surface
```

The pull worker retains raw provider page provenance and uses source/native
identity and cursor replay. OPS-02B proves this real intake path and a real
`NO_TRADE` decision only. OPS-01 promoted one event through the production
promoter, but its provider was an HTTP fixture; that is integration proof, not
a real event-to-candidate runtime proof.

`internal/decisioning/classify`, `core`, `brains`, `risk`, `paper`, `pipeline`,
and related packages form a separate deterministic decision/research
framework with unit/golden coverage and test persistence. No production
`cmd/trader` World Monitor call routes through that pipeline. It is supporting
capability, not a second canonical Jax v1 production decision path. Do not
splice its decisions into the World Monitor path without an explicit contract.

AI/JaxMind is `NOT_REQUIRED_YET`: deterministic source-backed event decisions
are sufficient for a first bounded hypothesis once the identified market,
risk and approval/execution seams are corrected. Its absence is not a blocker.

## End-to-end seam map

| Transition | Classification | Evidence / limitation |
|---|---|---|
| Genuine provider → retained page/cursor/inbox | `CONNECTED_AND_PROVED` | OPS-02B real local runtime; durable source/native IDs and trace; no economic write |
| Inbox/event → deterministic `NO_TRADE` | `CONNECTED_AND_PROVED` | OPS-02B real-origin decision trace |
| Inbox/event → candidate/evidence promotion | `CONNECTED_BUT_NOT_PROVED` | Production promoter ran in OPS-01, but with synthetic provider fixture |
| Candidate → candidate-level risk review | `CONNECTED_BUT_NOT_PROVED` | Promoter integration; proof assumptions and no real account-bound workflow risk |
| Candidate risk review → portfolio workflow risk | `DISCONNECTED` | OPS-01 constructs `ops01SyntheticRisk` in test code; no production adapter found |
| Candidate → human candidate approval/rejection | `CONNECTED_BUT_NOT_PROVED` | UI/API and durable approval exist; actor is header-controlled, not JWT-bound |
| Authenticated approval → exploratory PAPER queue | `MANUAL_HANDOFF` | Separate protected queue endpoint accepts caller-supplied request; no production caller found |
| Queue → simulated order/fill/ledger/lifecycle | `CONNECTED_BUT_NOT_PROVED` | Runtime/Postgres integration; OPS-01 synthetic account/event/market/workflow artifacts |
| Open lifecycle → review/checkpoint/exit recommendation | `CONNECTED_BUT_NOT_PROVED` | Production runtime tests and OPS-01 fixture review |
| Exit recommendation → JWT-bound human exit decision | `CONNECTED_BUT_NOT_PROVED` | Corrected API and durable Postgres flow tested; real economic activity absent |
| Approved exit → close/outcome/reconciliation | `CONNECTED_BUT_NOT_PROVED` | OPS-01 Postgres accounting proof with synthetic observation |

No real-runtime trace spans the full economic loop. Integration tests prove
component connectivity under fixtures, not genuine price semantics or a real
authenticated candidate-to-entry journey.

## Parallel and legacy path classification

| Surface | Classification | Finding |
|---|---|---|
| `world_monitor_pull_worker` | `CANONICAL_CURRENT` | Real event intake and persisted deterministic event decisions |
| `opportunity_scanner` / `world_monitor_opportunity_promoter` | `CANONICAL_CURRENT` for candidate promotion | Reads inbox decisions, strategy instances and persisted candles; writes candidate/evidence/risk-review records |
| `candidate_trades` / `strategy_signals` | `SUPPORTING_COMPONENT` | Promotion output and approval linkage; not exploratory execution |
| `candidate_approvals` / approval UI | `CANONICAL_CURRENT` for candidate review | Durable approval status and paper-ticket-ready state; not bound to workflow risk/entry queue; actor source is spoofable |
| `execution_instructions` / execution worker | `LEGACY_BUT_SAFE` when disabled; separate conditional broker-paper route | Requires paper mode, `IB_PAPER_TRADING=true`, `ALLOW_LIVE_TRADING!=true`, and optional enable gate. OPS-02B disabled it. It is not the exploratory in-process PaperVenue. Do not enable or bridge it as a shortcut. |
| `exploratorypaper` + `papertrading.PaperVenue` | `CANONICAL_CURRENT` for corrected simulated exploratory lifecycle | Account-scoped simulated venue and persistent lifecycle, review/exit/outcome; lacks production candidate-approval producer |
| `internal/decisioning/*` | `SUPPORTING_COMPONENT` | Deterministic tested framework, not wired to genuine World Monitor production path |
| Paper-ticket review / older manual signal routes | `LEGACY_BUT_SAFE` / `SUPPORTING_COMPONENT` | Review/read surfaces and separate workflows; no evidence they complete the corrected exploratory queue seam |

The instruction/broker-paper worker is a parallel execution surface, but OPS-02B
had it disabled and created no economic activity. No same-candidate duplicate
execution was observed. Keep it disabled and keep the account-scoped internal
PaperVenue as the sole intended Working Jax v1 economic sink until explicit
review resolves the route.

## Cross-cutting trustworthiness matrix

| Concern | Finding and strongest evidence | Classification |
|---|---|---|
| Account scoping | Explicit `PAPER_ACCOUNT_ID`; stores/runtime fail closed without it; restore derives ownership from ledger events; OPS-01 tests cross-account rejection and read isolation | `TESTED_INTEGRATION` (synthetic/disposable DB) |
| Idempotency | Stable entry/exit/order/fill/outcome identities; same-payload entry queue and exit decision replay covered | `TESTED_INTEGRATION` |
| Restart recovery | Venue restore and lifecycle/review reload tested; OPS-01 restarted entry and exit flow | `TESTED_INTEGRATION` |
| Concurrency | Required Linux race jobs cover replay/lifecycle and OPS-01; implementation evidence, not real economic evidence | `TESTED_INTEGRATION` |
| Temporal integrity | Event timestamps and decision versions retained. Review market query lacks explicit current-time, source, timeframe and freshness limits; no complete real-data as-of proof | `IMPLEMENTED_UNPROVEN`; market temporal constraint is a `CORE_HARD_BLOCKER` |
| Error visibility | OPS-01 readiness exposes worker health; worker failures log/persist World Monitor health. Failed `RecordReviewUnavailable` persistence is ignored in `Runtime.processReview` | `TESTED_INTEGRATION`; narrow visibility gap `EVIDENCE_GAP_REQUIRING_PROOF` |
| Accounting integrity | Corrected outcome deducts commissions once; spread/slippage are embedded/attributed; ledger reconciliation and transaction closure tested | `TESTED_INTEGRATION` |
| Test isolation | `internal/testsupport` rejects normal/non-disposable DB identities; CI sets `PAPER02R_REQUIRED_INTEGRATION=true`, provisions disposable PostgreSQL and runs required integrations | `TESTED_INTEGRATION` |
| Safety | Exploratory binding requires PAPER, authority NONE, broker false and leverage ≤1; OPS-02B execution/market ingester disabled. Candidate approval actor binding remains unsafe | `TESTED_INTEGRATION`, with actor issue a `CORE_HARD_BLOCKER` |

## Monitor, exit and outcome detail

`BuildReviewSchedule` schedules one review per configured trading session;
`startExploratoryPaperReviewWorker` polls each minute. The runtime persists a
checkpoint and exit recommendation before awaiting approval. Production
`PostgresStore.ApproveExit` reads the durable operator decision and returns
`ErrHumanApprovalPending` while only a recommendation exists. The protected
exit handler checks JWT-context identity, requires confirmation actor equality,
and the Postgres store validates position/review/recommendation/entry workflow
and intent binding.

Exit approval/rejection is durable and idempotently verified; simulated exit
persists closure and outcome transactionally. Corrected accounting is gross
executed-price P&L minus entry/exit commission; spread and slippage remain
attribution because they are embedded in executed prices. Excursion metrics
are `COMPLETE` only with sufficient validated coverage; otherwise they remain
incomplete/unknown. These are implementation and disposable-Postgres
integration guarantees; OPS-01 did not provide genuine market marks or a real
economic outcome.

## Blockers and deferred items

| Finding | Classification | Minimum correction/proof |
|---|---|---|
| Candidate approval actor can be spoofed with `X-User-ID` despite JWT-protected route | `CORE_HARD_BLOCKER` | Bind approval/rejection identity to validated JWT claims; handler tests must prove spoof resistance and durable identity |
| Candidate-level risk/approval does not produce portfolio-risk-bound workflow and exploratory entry; queue trusts caller-supplied full snapshot | `CORE_HARD_BLOCKER` | Add one server-owned, account-scoped path loading canonical candidate/evidence/risk/approval state, creating/loading versioned workflow and intent, and queueing exactly once |
| Genuine/provenanced market-price input has not been shown in corrected economic runtime; review query lacks source/timeframe/current-time/freshness constraints | `CORE_HARD_BLOCKER` | Define provider/source/timeframe/freshness contract; bind observed/received times; reject future/stale/unknown input; prove persistence-to-review use without broker execution |
| Production risk defaults/account assumptions not validated against a real paper account and selected hypothesis | `HYPOTHESIS_SPECIFIC_REQUIREMENT` | At preregistration, freeze sizing/loss/stop/target/instrument/account policy; no strategy selected here |
| Full real-runtime economic loop and genuine-price outcome/restart evidence absent | `EVIDENCE_GAP_REQUIRING_PROOF` | After bounded seam fixes, separately authorized isolated rehearsal before prospective activity |
| AI/JaxMind, advanced memory, orchestration, live broker, HARNESS-04, zero selected strategy count | `DEFERRED_NON_BLOCKER` | Not required for deterministic first-hypothesis contract; no activation authority |
| UI polish | `COSMETIC_NON_BLOCKER` | Does not affect the identified minimum core |

Zero enabled strategies is intentional before hypothesis selection, not a
defect. This audit does not claim a current strategy count because the normal
database is stopped.

## Normal database and runtime state

Docker context was `desktop-linux`. Canonical normal Jax container
`jax-trading-assistant-postgres-1` was present but **exited (255)** (host port
5433 when running). No Jax application runtime was started and no database
query was attempted. The unrelated World Monitor container was restarting and
was not touched.

Use only the following as `HISTORICAL_REVIEWED_CHECKPOINT`, not current truth:

- OPS-02C1: six ABORTED exploratory pilots, zero ACTIVE.
- OPS-02B3: 19 strategy instances, zero enabled and 19 disabled; twelve
  confirmed test fixtures disabled and preserved.
- OPS-02B: PAPER-02 ABORTED, one incident, zero genuine prospective
  opportunities; runtime proof had zero account-scoped economic writes.
- OPS-02B: `jax-paper-runtime-v1` had zero account rows/orders/fills/ledger
  events at the reviewed checkpoint; the bounded run made no economic action.

These checkpoints are historical and time-bounded, not present database proof.

## Scientific readiness verdict

**BLOCKED_BEFORE_HYPOTHESIS_DESIGN**

Reusable components exist and the evidence does not support a major rewrite.
The loop is not yet trustworthy end to end because candidate actor identity
and approval handoff are not authoritative, and genuine market-price
provenance/temporal semantics have not been proven in the corrected runtime.

## Exact next package recommendation

**CORE-READINESS-02 — CANONICAL APPROVED-PAPER SEAM AND MARKET-DATA PROOF**

Bound the package to:

1. JWT-claim-bound candidate approval/rejection identity;
2. server-owned candidate → current account risk decision → workflow/human
   approval → paper intent → account-scoped exploratory queue;
3. strict provenance, timeframe, freshness and no-future-observation checks
   for entry/review/exit prices;
4. disposable-PostgreSQL integration, concurrency/restart proof and isolated
   end-to-end rehearsal with genuine provider-origin market input, exactly one
   entry/exit and reconciled outcome, without broker execution.

This recommendation does not select a strategy or authorize normal-database
mutation, pilot, prospective collection, broker action or live trading.
Separate technical-lead authorization is required before implementation or
economic activity.

## Evidence inventory

Reviewed records: PAPER-01 / PAPER-01B / PAPER-01C, PAPER-02R, OPS-01,
OPS-02A, OPS-02B, OPS-02C1 and OPS-02D. The frozen PAPER-02 protocol and
incident remain unchanged. OPS-02B is weighted only as real corrected-runtime
plus genuine event-intake evidence with zero economic activity. OPS-01 is
weighted only as synthetic/disposable full-loop operational evidence. No
record establishes profitability, predictive edge or formal-forward
readiness.
