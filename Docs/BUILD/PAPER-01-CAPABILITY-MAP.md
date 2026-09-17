# PAPER-01 Existing Capability Map

PAPER-01B reuses the established contracts and closes the operational gaps
identified by external review. It adds a durable exploratory lifecycle without
creating a second workflow or paper ledger.

| Loop stage | Existing capability | Contract / persistence | API or UI | PAPER-01 gap and change |
| --- | --- | --- | --- | --- |
| Event/news intake | `worldmonitorintelligence`, provider/event pipeline | canonical event/provenance IDs | existing event/operator diagnostics | consume accepted events; no new source or acquisition |
| Issuer/asset resolution | `assetresolution` | canonical issuer/instrument mapping and ruleset | existing evidence views | bind resolved IDs into thesis; no duplicate resolver |
| Evidence/corroboration | `eventdecisions`, `evidencequality`, `candidate_evidence_scores` | reviewed score projection, policy version, freshness/contradiction/relevance facts | candidate evidence review | candidate eligibility derives from reviewed facts; caller booleans/floats cannot promote a candidate |
| Candidate decision | `eventdecisions`, `candidates` | `WATCH` / `NO_TRADE` / `CANDIDATE`, gate/reason provenance | `CandidatesPage` | exploratory generator requires causal mechanism plus quant/risk/technical support |
| Thesis | `swingresearch` provides research thesis output | existing output is not an entry-bound immutable thesis | candidate review | new versioned `exploratorypaper.TradeThesis` |
| Quant/technical | `quant`, existing swing research | quant result IDs and deterministic outputs | candidate detail | technical confirmation supports; it cannot be sole catalyst |
| Portfolio risk | `portfoliorisk` | risk decision, policy, leverage and exposure provenance | approval review | call existing evaluator; require `NONE` authority and 1x max |
| Human approval | `approvals`, `workflow` | durable approval and workflow transition history | existing approval queue | mandatory for every new exploratory entry; immutable provenance |
| Paper order/fill | `papertrading` | paper capability, order/fill/ledger/reconciliation/attribution | existing paper plan/outcome views | use existing venue and ledger only; durable lifecycle references existing IDs |
| Monitoring/review | scheduler seams in `cmd/trader` | durable review rows, idempotency identities, missing-data status | exploratory-paper read model | restart-safe due-review worker; duplicate delivery has no duplicate scientific effect |
| Exit | `papertrading`, `portfoliorisk`, workflow | deterministic paper order/fill and risk controls | current paper/outcome pages | new session-calendar exit policy for stop, target, invalidation, risk kill, time, manual |
| Outcomes/learning | `papertrading` fills/ledger plus exploratory contract | deterministic gross/net/cost/MFE/MAE/checkpoint validation | exploratory-paper read model | outcome arithmetic is verified from identified simulated fills and observations |
| Durable lifecycle | existing Postgres workflow/paper tables | `exploratory_paper_lifecycles` and append-only reassessment/review/checkpoint/outcome tables | protected operator API | frozen thesis/binding/policy hashes and fail-closed restart reconciliation |
| Formal firewall | validation artifacts and mode policy | immutable formal identities | review handovers | reject exploratory-to-formal admission and relabelling |

The implementation intentionally does not rerun VAL-03R4B, reacquire formal
data, tune a strategy, start PAPER-02, or alter Phase 13 status.
