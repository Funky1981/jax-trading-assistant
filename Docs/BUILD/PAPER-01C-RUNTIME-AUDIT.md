# PAPER-01C Runtime Audit

Status: **IMPLEMENTED / EXTERNAL RE-REVIEW REQUIRED**. This closes the
runtime-loop blocker only. It does not execute a prospective pilot or
authorize PAPER-02, FORMAL_FORWARD_PAPER, Phase 13, or live/broker execution.

## Reused runtime seams

| Loop stage | Existing repository seam | PAPER-01C integration |
| --- | --- | --- |
| Event/evidence intake | `eventdecisions.Store.LoadSelectedEvents`; `world_monitor_research_inbox`; `candidate_evidence_scores` and `candidate_evidence_items` | `PostgresStore.LoadApprovedEntries` reloads the canonical score/items projection before acting. |
| Candidate gate | `exploratorypaper.GenerateCandidate` and `EvidenceAssessment.ValidateFor` | Runtime sets the assessment only from the persisted projection; WATCH/NO_TRADE never reaches the venue. |
| Thesis/quant/risk | `TradeThesis`, `portfoliorisk.RiskDecision`, existing workflow binding | `Runtime.processEntry` validates exact event/instrument identity, accepted risk, model/policy versions, and paper safety. |
| Human approval | `workflow.Workflow`, `workflow.Confirmation`, `workflow.PaperIntent` | Protected entry queue accepts only an unexpired human-approved `PAPER_INTENT_CREATED` workflow. It cannot approve. |
| Simulated entry | `papertrading.PaperVenue`, `PaperLedger` | Runtime submits and fills only through `PaperVenue`, then calls `SaveApprovedEntry`. |
| Review inputs | Existing `candles` table and `candidate_evidence_items` | `postgresExploratoryReviewSource` loads both, validates provenance/session state, and returns explicit unavailable errors. |
| Review persistence | `ApplyEvidence`, `PersistCheckpoint`, `PersistExitRecommendation` | Duplicate evidence is idempotent; checkpoint and `EXIT_RECOMMENDED` state are durable. |
| Exit | `EvaluateExit`, workflow approval, `PaperVenue`, existing ledger | Exit recommendation waits for distinct human approval before simulated exit order/fill and `PersistOutcome`. |
| Operator review | Protected exploratory-paper API | Existing list/detail read model exposes lifecycle, reviews, checkpoints, exit state, outcome, and non-formal label. |

## Runtime path

`POST /api/v1/exploratory-paper/entry-queue` is a protected handoff from an
already-approved canonical workflow. `cmd/trader/exploratory_paper_worker.go`
starts `exploratorypaper.Runtime` only when runtime mode is `PAPER`. The worker
loads queued entries, reloads canonical evidence, performs the deterministic
candidate/risk/session/safety checks, uses the isolated venue, persists the
lifecycle, and schedules five session reviews.

The same worker calls `Runtime.RunDueReviews`. Review inputs come from the
persisted candle and evidence facilities. `PAPER_SESSION_CALENDAR_JSON` is
required; absent, incomplete, stale, or contradictory calendar state fails
closed. Missing evidence, market observations, or ledger provenance is recorded
as `MISSING_DATA`, without fabricating a completed review.

## Idempotency and restart

The entry queue is unique by candidate and has a stable identity derived from
candidate/workflow/paper-intent identity. Existing lifecycle lookup happens
before venue submission. Existing workflow, paper order/fill, ledger, and
exploratory lifecycle records remain the authoritative restart state. Review
delivery is unique by position/session; evidence is unique by position/evidence
identity; checkpoints are unique by position/session. Conflicting identity
reuse fails closed.

## Fifth-session rule

The explicit calendar returns session starts. Entry must occur during a known
open session. Reviews are scheduled for sessions 1–5, skipping weekends and
holidays. `TIME_LIMIT` becomes actionable at the opening of session five unless
a higher-precedence `RISK_KILL`, `MANUAL_OPERATOR`, `THESIS_INVALIDATED`,
`STOP`, or `TARGET` reason applies.

## Safety boundary

`Runtime` imports only `papertrading`, `portfoliorisk`, and `workflow`; it does
not import `internal/modules/execution` or an IB client. The route rejects any
runtime mode other than `PAPER`. The venue contract requires `PAPER` and
`ExecutionAuthority=NONE`; workflow and paper-intent contracts require human
approval and prohibit broker/live execution.

## Remaining limitation

The queue is an operator/runtime handoff, not an automatic pilot. External
review must still decide whether and when any prospective exploratory pilot is
authorized. No such pilot is run by this package.
