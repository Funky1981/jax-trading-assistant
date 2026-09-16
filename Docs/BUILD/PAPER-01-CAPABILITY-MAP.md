# PAPER-01 Existing Capability Map

PAPER-01 reuses the established contracts and adds only the missing
event-driven thesis, session lifecycle, outcome, and evidence-firewall layer.

| Loop stage | Existing capability | Contract / persistence | API or UI | PAPER-01 gap and change |
| --- | --- | --- | --- | --- |
| Event/news intake | `worldmonitorintelligence`, provider/event pipeline | canonical event/provenance IDs | existing event/operator diagnostics | consume accepted events; no new source or acquisition |
| Issuer/asset resolution | `assetresolution` | canonical issuer/instrument mapping and ruleset | existing evidence views | bind resolved IDs into thesis; no duplicate resolver |
| Evidence/corroboration | `evidencequality`, `researchrecommendation` | evidence packets, source references, quality/freshness | candidate evidence review | require source-backed material evidence and retain references |
| Candidate decision | `eventdecisions`, `candidates` | `WATCH` / `NO_TRADE` / `CANDIDATE`, gate/reason provenance | `CandidatesPage` | exploratory generator requires causal mechanism plus quant/risk/technical support |
| Thesis | `swingresearch` provides research thesis output | existing output is not an entry-bound immutable thesis | candidate review | new versioned `exploratorypaper.TradeThesis` |
| Quant/technical | `quant`, existing swing research | quant result IDs and deterministic outputs | candidate detail | technical confirmation supports; it cannot be sole catalyst |
| Portfolio risk | `portfoliorisk` | risk decision, policy, leverage and exposure provenance | approval review | call existing evaluator; require `NONE` authority and 1x max |
| Human approval | `approvals`, `workflow` | durable approval and workflow transition history | existing approval queue | mandatory for every new exploratory entry; immutable provenance |
| Paper order/fill | `papertrading` | paper capability, order/fill/ledger/reconciliation/attribution | existing paper plan/outcome views | use existing venue only; no parallel ledger |
| Monitoring/review | `orchestration` and scheduling seams | bounded jobs/observability | existing operational surfaces | explicit one-review-per-trading-session schedule; new evidence reassessment |
| Exit | `papertrading`, `portfoliorisk`, workflow | deterministic paper order/fill and risk controls | current paper/outcome pages | new session-calendar exit policy for stop, target, invalidation, risk kill, time, manual |
| Outcomes/learning | `paperoutcomes` legacy hypothetical checkpoints | existing 1h/1d/1w tracker | `OutcomesPage` | new 1/2/3/5-session exploratory outcome contract bound to thesis and policies |
| Formal firewall | validation artifacts and mode policy | immutable formal identities | review handovers | reject exploratory-to-formal admission and relabelling |

The implementation intentionally does not rerun VAL-03R4B, reacquire formal
data, tune a strategy, start PAPER-02, or alter Phase 13 status.
