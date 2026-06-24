# Jax Capability Matrix

## Status values

| Status | Meaning |
|---|---|
| NOT_PLANNED | Explicitly excluded from current roadmap |
| PLANNED | Desired capability but not designed |
| DESIGNED | Spec exists but code does not |
| PARTIAL | Existing support exists but does not satisfy the full Jax capability |
| IMPLEMENTED | Code exists |
| TESTED | Automated tests exist |
| PROVEN | Validated through paper/research evidence |

## Capability matrix

| Capability | Status | Owner Area | Evidence Required | Code Path | Test Path | Notes |
|---|---:|---|---|---|---|---|
| Product charter | DESIGNED | Docs | Approved product truth | `Docs/JAX_PRODUCT_CHARTER.md` | N/A | Source of truth |
| Capability matrix | DESIGNED | Docs | Matrix exists and is maintained | `Docs/CAPABILITY_MATRIX.md` | N/A | Must be updated every phase |
| Phase 0 capability reset | DESIGNED | Docs | Phase 0 contract accepted | `Docs/PHASE_CONTRACTS/00_CAPABILITY_RESET.md` | N/A | Governance/reset phase only; no trading logic |
| Decision Core Phase 1 | TESTED | Decision Core | Unit tests + FTSE golden fixture | `internal/decisioning/core` | `internal/decisioning/core/decision_test.go`, `tests/golden/decision_runner_test.go` | Deterministic structured decision core implemented; Phase 2 Event Intelligence feeds enriched events into this core |
| Event intake | TESTED | Decisioning | Structured event schema + golden input decode | `internal/decisioning/core/event.go` | `tests/golden/events` | Structured event input only; no scraping or article parsing |
| Event classification | TESTED | Event Intelligence | Golden classification tests | `internal/decisioning/classify` | `internal/decisioning/classify/event_classifier_test.go`, `tests/golden/decision_runner_test.go` | Deterministic macro/company/commodity/central-bank classification; unclear events become `UNKNOWN` |
| Primary driver extraction | TESTED | Event Intelligence | Golden extraction tests | `internal/decisioning/classify` | `internal/decisioning/classify/driver_extractor_test.go`, `tests/golden/decision_runner_test.go` | Normalises known drivers including oil, labour data, rates, central bank, guidance, earnings, FX, and geopolitical risk |
| Conflicting signal detection | TESTED | Event Intelligence | Conflict golden cases | `internal/decisioning/classify` | `internal/decisioning/classify/conflict_detector_test.go`, `tests/golden/decision_runner_test.go` | Detects FTSE/oil/labour, earnings/guidance, index-composition, macro/FX/rates, pending central-bank, rumour-only, and no-edge conflicts |
| Affected asset mapping | TESTED | Event Intelligence | Driver-to-asset tests | `internal/decisioning/classify` | `internal/decisioning/classify/asset_mapper_test.go`, `tests/golden/decision_runner_test.go` | Maps oil to BP/SHEL/energy exposure and UK labour/rates to GBP/gilts/FTSE/UK sectors |
| Decision enum | TESTED | Decision Core | Enum tests | `internal/decisioning/core/decision.go` | `internal/decisioning/core/decision_test.go` | Includes all Phase 1 allowed decisions |
| NO_TRADE decision | TESTED | Decision Core | Golden rejection tests | `internal/decisioning/core` | `tests/golden/events` | Default decision and expected common outcome; not an error |
| WATCH decision | TESTED | Decision Core | Rule tests | `internal/decisioning/core` | `internal/decisioning/core/decision_test.go` | Missing confirmation |
| SETUP_FORMING decision | PLANNED | Decision Core | Golden setup tests | `internal/decisioning/core` | `tests/golden/events` | Not tradable yet |
| TRADE_CANDIDATE decision | TESTED | Decision Core | Candidate tests | `internal/decisioning/core` | `internal/decisioning/core/decision_test.go` | Structured candidate only; no paper approval or execution |
| Evidence bundle | TESTED | Decision Core | Schema validation | `internal/decisioning/core/evidence.go` | `internal/decisioning/core/decision_test.go` | Carries event/reasoning/scores/final decision |
| Swing Brain v1 | PLANNED | Trading Brain | 25 golden cases | `internal/decisioning/brains/swing` | `brain_test.go` | First active strategy brain after Decision Core |
| Day Trading Brain | NOT_PLANNED | Future | Not in current roadmap | N/A | N/A | Explicitly excluded; do not add day-trading infrastructure |
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
| Live trading | NOT_PLANNED | Execution | Explicitly excluded | N/A | N/A | Do not implement live orders, auto execution, or broker order placement |

## Matrix rule

Every pull request must update this file if it changes, adds, or claims a capability.

No implementation is considered complete unless the capability state is updated and supported by evidence.
