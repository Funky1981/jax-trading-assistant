# PAPER-01B External Review Evidence (Superseded by PAPER-01C)

Status: **IMPLEMENTED / ACCEPTED DIRECTIONALLY**. PAPER-01 is now
**CONDITIONAL GO** pending PAPER-01C re-review. No PAPER-02 pilot, formal run, historical
validation rerun, Phase 13 work, or live/broker action is part of this package.

| External finding | Implementation | Tests | Persistence/runtime evidence | Documentation | Remaining limitation |
| --- | --- | --- | --- | --- | --- |
| Exploratory lifecycle was not durable/restart-safe | `postgres.go`; migration `000069`; frozen lifecycle, review, reassessment, checkpoint, outcome records | exploratory contract/reassessment/restart-oriented tests; full repository validation | `PostgresStore.Get` revalidates workflow, paper order/fill, frozen hashes, reviews, checkpoints, and outcome; missing/divergent identity fails closed | `PAPER-01B-IMPLEMENTATION-AUDIT.md`, `PAPER-01-CAPABILITY-MAP.md` | A future pilot still needs operational deployment/runbook and real approved intake; this package does not run one |
| Candidate gate trusted caller quality/corroboration fields | `EvidenceAssessment.ValidateFor`; `GenerateCandidate` requires canonical reviewed score facts, fingerprint, policy, relevance, freshness, contradiction and independent-source gates | candidate adversarial table tests | source is the existing `candidate_evidence_scores` projection | `PAPER-01.md`, `EXPLORATORY_VS_FORMAL.md` | Canonical upstream projection remains the source of truth |
| Approved thesis was mutable/weakly bound | `FreezeThesis`, content/evidence hashes, immutable binding hash, `VerifyFrozenIdentity`, `VerifyApprovalBinding` | tamper and approval-binding tests | frozen payload/hash trigger and paper/workflow identity checks | `APPROVAL_AND_EXIT_POLICY.md` | Hash verification detects tampering; it does not make mutable upstream source records immutable |
| Evidence replay could duplicate effects | stable evidence fingerprint and append-only reassessment key `(position_id,evidence_id)` | duplicate strengthening/invalidating/conflict tests | unique key and append-only trigger | lifecycle and formal-firewall docs | Exactly-once scheduling is not required; effects are idempotent |
| Exit/session semantics were inconsistent | canonical precedence in `EvaluateExit`; explicit calendar timezone/open-close/session coverage | collision, closed-market, weekend/holiday/unknown-calendar tests | review schedule and next-review fields are durable | `APPROVAL_AND_EXIT_POLICY.md`, `TRADE_LIFECYCLE.md` | Calendar data must be supplied with complete coverage; unknown coverage fails closed |
| Outcome/MFE/MAE arithmetic was caller-declared | `BuildOutcomeFromFills`, `Outcome.Validate`, identified price path, cost model/version binding | LONG/SHORT, cost, arithmetic, excursion and checkpoint tests | outcomes are append-only and linked to existing simulated fills | `PAPER-01.md`, `TRADE_LIFECYCLE.md` | Missing price observations remain incomplete rather than being inferred |
| Operator review surface was insufficient | protected exploratory-paper list/detail API and scheduler worker | compile/package tests and repository checks | read model comes from durable lifecycle plus existing workflow/paper rows | `PAPER-01-CAPABILITY-MAP.md` | Minimal API only; no UI redesign or pilot controls were added |
| Exploratory/formal and live safety boundaries | immutable mode/formal flag, relabel rejection, PAPER/NONE/false/<=1x validation | firewall and paper venue safety tests | database checks/triggers prevent formal promotion and identity mutation | roadmap/status/current package docs | Formal and live authorization remain external decisions and are not implied |

## Exact safety position

`PAPER-01B = IMPLEMENTED / ACCEPTED DIRECTIONALLY`; `PAPER-01C = IMPLEMENTED /
EXTERNAL RE-REVIEW REQUIRED`; `PAPER-01 = CONDITIONAL GO / PENDING PAPER-01C
EXTERNAL RE-REVIEW`; `PAPER-02 = NOT AUTHORIZED`;
`FORMAL_FORWARD_PAPER = NOT STARTED`; demonstrated edge = NO; live trading and
Phase 13 = NOT AUTHORIZED; optional commercialisation = DEFERRED / NOT A
CURRENT OBJECTIVE.
