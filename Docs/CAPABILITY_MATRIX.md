# Jax Capability Matrix

## Status values

| Status | Meaning |
|---|---|
| NOT_PLANNED | Explicitly excluded from current roadmap |
| PLANNED | Desired capability but not designed |
| DESIGNED | Spec exists but code does not |
| IMPLEMENTED | Code exists |
| TESTED | Automated tests exist |
| PROVEN | Validated through paper/research evidence |

## Capability matrix

| Capability | Status | Owner Area | Evidence Required | Code Path | Test Path | Notes |
|---|---:|---|---|---|---|---|
| Product charter | DESIGNED | Docs | Approved product truth | `Docs/JAX_PRODUCT_CHARTER.md` | N/A | Source of truth |
| Capability matrix | DESIGNED | Docs | Matrix exists and is maintained | `Docs/CAPABILITY_MATRIX.md` | N/A | Must be updated every phase |
| Event intake | PLANNED | Decisioning | Event input schema + parser tests | `internal/decisioning/core` | `tests/golden/events` | Pasted/news/manual event first |
| Event classification | PLANNED | Event Intelligence | Golden classification tests | `internal/decisioning/classify` | `*_test.go` | Macro/company/commodity/etc |
| Primary driver extraction | PLANNED | Event Intelligence | Golden extraction tests | `internal/decisioning/classify` | `*_test.go` | Extract main causes |
| Conflicting signal detection | PLANNED | Event Intelligence | Conflict golden cases | `internal/decisioning/classify` | `tests/golden/events` | Critical no-trade feature |
| Affected asset mapping | PLANNED | Event Intelligence | Driver→asset tests | `internal/decisioning/classify` | `*_test.go` | Oil→BP/SHEL etc |
| Decision enum | PLANNED | Decision Core | Enum tests | `internal/decisioning/core/decision.go` | `decision_test.go` | Must include NO_TRADE |
| NO_TRADE decision | PLANNED | Decision Core | Golden rejection tests | `internal/decisioning/core` | `tests/golden/events` | Default outcome |
| WATCH decision | PLANNED | Decision Core | Golden watch tests | `internal/decisioning/core` | `tests/golden/events` | Missing confirmation |
| SETUP_FORMING decision | PLANNED | Decision Core | Golden setup tests | `internal/decisioning/core` | `tests/golden/events` | Not tradable yet |
| TRADE_CANDIDATE decision | PLANNED | Decision Core | Candidate tests | `internal/decisioning/core` | `tests/golden/events` | Must include risk/invalidation |
| Evidence bundle | PLANNED | Decision Core | Schema validation | `internal/decisioning/core/evidence.go` | `evidence_test.go` | Carries event/reasoning/scores |
| Swing Brain v1 | PLANNED | Trading Brain | 25 golden cases | `internal/decisioning/brains/swing` | `brain_test.go` | First active brain |
| Day Trading Brain | NOT_PLANNED | Future | Not in current roadmap | N/A | N/A | Excluded until swing proven |
| Long-Term Brain | PLANNED | Future | Future charter | `internal/decisioning/brains/longterm` | TBD | Later, not before swing |
| Risk veto | PLANNED | Risk | Veto tests | `internal/decisioning/risk` | `risk_test.go` | Can downgrade/reject |
| Human approval | PARTIAL | Trader/API | Approval workflow test | Existing approval routes | Existing tests | Must carry decision evidence |
| Paper trade ticket | PLANNED | Paper Trading | Ticket schema + tests | `internal/decisioning/core/paper_ticket.go` | `paper_ticket_test.go` | Candidate→paper approval |
| Backtest evidence bundle | PLANNED | Research | Evidence validation | `internal/decisioning/core/backtest_evidence.go` | `backtest_evidence_test.go` | Uses existing research runtime |
| Dataset integrity check | PARTIAL | Research | Dataset hash validation | `cmd/research/backtest.go` | Existing/new tests | Existing foundation present |
| Slippage/cost modelling | PLANNED | Research | Backtest assumptions required | Research/backtest layer | Evidence tests | Required before PAPER_READY |
| Out-of-sample validation | PLANNED | Research | OOS evidence required | Research/backtest layer | Evidence tests | Required before promotion |
| Decision memory logging | PLANNED | Memory/Review | Decision records persisted | `internal/decisioning/review` | `review_test.go` | Every decision logged |
| No-trade outcome review | PLANNED | Memory/Review | 1d/1w/1m review records | `internal/decisioning/review` | `review_test.go` | Must review rejected trades |
| Paper-trade outcome review | PLANNED | Memory/Review | Outcome review | `internal/decisioning/review` | `review_test.go` | Learn from paper trades |
| Live trading | NOT_PLANNED | Execution | Explicitly excluded | N/A | N/A | Do not implement |

## Matrix rule

Every pull request must update this file if it changes, adds, or claims a capability.

No implementation is considered complete unless the capability state is updated and supported by evidence.
