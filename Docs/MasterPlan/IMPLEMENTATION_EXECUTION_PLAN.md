# MasterPlan Implementation Execution Plan

Last updated: February 25, 2026  
Branch baseline: `work`  
Scope source: `Docs/MasterPlan/*`

## 1. Program Objective
Deliver the complete MasterPlan as an auditable, no-fake-data, research-first event trading platform on the ADR-0012 runtime shape (`cmd/trader` + `cmd/research`), with gate-based promotion and paper-trading proof.

## 2. Non-Negotiables
1. No fake or synthetic data in paper, research truth paths, analysis, trust gates, or promotion decisions.
2. AI is advisory only, never execution authority.
3. No gate pass means no promotion and no trading.
4. Every completed capability must include code, tests, and evidence artifacts.

## 3. Phase Scoreboard (Prefilled)

| Phase | Name | Priority | Status | Owner | Depends On | Exit Gate(s) |
|---|---|---|---|---|---|---|
| 0 | Baseline hardening and governance | P0 | PARTIAL | `TBD-PM` | None | Program tracking evidence in place |
| 1 | No-fake-data enforcement | P0 | PARTIAL | `TBD-Platform` | 0 | Gate 9 |
| 2 | Event data foundation | P0 | MISSING | `TBD-Data` | 1 | Gate 1 |
| 3 | Market/session correctness | P0 | PARTIAL | `TBD-Data` | 2 | Gate 1 |
| 4 | Strategy framework and instances | P0 | PARTIAL | `TBD-Strategy` | 3 | Gate 0, Gate 3 |
| 5 | IBM event strategy pack | P0 | MISSING | `TBD-Strategy` | 4 | Gate 2 |
| 6 | Research reproducibility | P0 | PARTIAL | `TBD-Research` | 5 | Gate 2 |
| 6b | RAG research-only boundary work | P1 | MISSING | `TBD-Research` | 6 | RAG boundary checks |
| 7 | Execution risk/flatten/reconciliation | P0 | PARTIAL | `TBD-Execution` | 6 | Gate 4, Gate 5, Gate 7 |
| 8 | AI audit and decision trace | P1 | PARTIAL | `TBD-AI` | 6,7 | Gate 8 (if AI enabled) |
| 9 | UI operations integration | P0 | PARTIAL | `TBD-Frontend` | 6,7,8 | UI/API integration checks |
| 10 | Trust gates and shadow validation | P0 | MISSING | `TBD-Platform` | 1-9 | Gates 0-10 automation |
| 11 | Controlled live readiness (later) | P2 | MISSING | `TBD-Ops` | 10 | Extended paper evidence + signoff |

## 4. Workstream Owners (Set Before Sprint Start)

| Workstream | Scope | Owner | Backup |
|---|---|---|---|
| A | Safety and no-fake-data | `TBD` | `TBD` |
| B | Event and market data correctness | `TBD` | `TBD` |
| C | Strategy framework and IBM pack | `TBD` | `TBD` |
| D | Research and reproducibility | `TBD` | `TBD` |
| E | Execution, risk, flatten, recon | `TBD` | `TBD` |
| F | AI audit and replay | `TBD` | `TBD` |
| G | UI operations pages | `TBD` | `TBD` |
| H | Trust gates and shadow | `TBD` | `TBD` |
| I | Observability and incidents | `TBD` | `TBD` |

## 5. Gate Map (Must Be Green Before Paper Promotion)

| Gate | Name | Required | Current | Evidence Path |
|---|---|---|---|---|
| 0 | Config and schema integrity | Yes | PARTIAL | `reports/gate0/<date>/` |
| 1 | Data reconciliation | Yes | MISSING | `reports/gate1/<date>/` |
| 2 | Deterministic replay | Yes | MISSING | `reports/gate2/<date>/` |
| 3 | Artifact promotion controls | Yes | PARTIAL | `reports/gate3/<date>/` |
| 4 | Execution path integration | Yes | MISSING | `reports/gate4/<date>/` |
| 5 | PnL reconciliation | Yes | MISSING | `reports/gate5/<date>/` |
| 6 | Failure injection | Yes | MISSING | `reports/gate6/<date>/` |
| 7 | Flatten-by-close proof | Yes | MISSING | `reports/gate7/<date>/` |
| 8 | AI audit completeness | If AI enabled | MISSING | `reports/gate8/<date>/` |
| 9 | Data provenance integrity | Yes | MISSING | `reports/gate9/<date>/` |
| 10 | Shadow/parity validation | Pre-live | PARTIAL | `reports/gate10/<date>/` |

## 6. First 30 Execution Tasks (Ordered)

1. Freeze scoreboard baseline and assign owners.
2. Implement runtime mode policy in trader and research startup.
3. Add startup provider policy validation.
4. Gate synthetic/fake backtest helpers outside dev/test.
5. Add provenance fields to runs and artifact evidence.
6. Backfill existing rows and expose provenance in APIs.
7. Add provenance gate job and blocking behavior.
8. Implement event raw/normalized/symbol mapping schema.
9. Build event normalization and ambiguity handling.
10. Add event list/detail/timeline APIs.
11. Add intraday dataset snapshot and hash linkage.
12. Implement data reconciliation gate.
13. Implement strategy type metadata registry and endpoints.
14. Implement strategy instance DB CRUD and validation.
15. Wire strategy instance management UI.
16. Implement `news_shock_momentum_v1`.
17. Implement `opening_range_to_close_v1`.
18. Add deterministic strategy snapshot tests.
19. Complete real deterministic research run persistence.
20. Add parameter sweeps orchestration.
21. Add walk-forward orchestration.
22. Persist full execution lifecycle (intent/order/fill transitions).
23. Add idempotency and duplicate suppression.
24. Add restart-safe execution processing tests.
25. Implement flatten-by-close and proof artifact output.
26. Implement PnL reconciliation and correction append model.
27. Implement `ai_decisions` and acceptance/rejection capture.
28. Add decision timeline API with AI and provenance linkage.
29. Complete `/research`, `/analysis`, `/testing` integration.
30. Automate gates 0-10 with API and UI status/proof links.

## 7. Weekly Checkpoint Plan (Prefilled)

| Week | Focus | Planned Completion | Review Check |
|---|---|---|---|
| Week 1 | Phase 0 + Phase 1 core (`mode`, `provider checks`, fake-path gating) | Tasks 1-4 | Gate 9 dry run |
| Week 2 | Provenance schema/API/UI + promotion blocking | Tasks 5-7 | Gate 9 enforced |
| Week 3 | Event schema, normalization, symbol mapping, APIs | Tasks 8-10 | Event pipeline demo |
| Week 4 | Intraday datasets and reconciliation | Tasks 11-12 | Gate 1 first pass |
| Week 5 | Strategy metadata + instances API/UI | Tasks 13-15 | Config-to-instance roundtrip |
| Week 6 | IBM safe strategies + deterministic tests | Tasks 16-18 | Replay consistency review |
| Week 7 | Research reproducibility, sweeps, walk-forward | Tasks 19-21 | Gate 2 first pass |
| Week 8 | Execution lifecycle hardening and idempotency | Tasks 22-24 | Gate 4 dry run |
| Week 9 | Flatten and reconciliations | Tasks 25-26 | Gates 5 and 7 first pass |
| Week 10 | AI audit trace and timeline linkage | Tasks 27-28 | Gate 8 dry run |
| Week 11 | UI operations completion | Task 29 | UI acceptance review |
| Week 12 | Gate automation and shadow integration | Task 30 | Gates 0-10 status review |

## 8. Evidence Requirements Per Completed Item
Any row moved to `DONE` must include:
1. File path(s) changed.
2. Migration file(s) if schema changes.
3. API route(s) added or updated.
4. Test command and result.
5. Gate artifact path if gate-relevant.

## 9. Immediate Next Actions
1. Assign owners in Section 4.
2. Confirm Week 1 and Week 2 capacity and scope lock.
3. Start Task 1 through Task 4 and update this file daily.

## 10. Status Update (2026-02-25)
**Completed (evidence captured in git history on branch `work`):**
1. Tasks 1-4 (runtime mode policy, provider policy, synthetic backtest gating).
2. Tasks 5-7 (provenance fields, APIs, dataset provenance gate).
3. Tasks 8-12 (event schema, normalization, APIs, dataset snapshots + Gate 1 recon).
4. Tasks 13-18 (strategy types registry, IBM pack strategies, deterministic tests).
5. Tasks 19-21 (research backtest persistence, parameter sweeps, walk-forward wiring).
6. Strategytypes backtest integration with event sentiment/materiality persistence.
7. NewsAPI + FinancialDatasets fallbacks; Polygon can be disabled via `POLYGON_ENABLED=false`.

**Evidence pointers (commits):**
- `4c6cfb4`, `9579b4f`, `996c259`, `19ee219`, `f887549`, `d9626aa` (phases 0-3, provenance/event/data/UI).
- `c25721a`, `eb7259a` (IBM strategies + deterministic tests).
- `b68c049`, `ec86b95` (strategytypes backtests + event sentiment/materiality).
- `cc5a9c5`, `39d6f08`, `929677f`, `d45df1b`, `7ac6969` (provider wiring/priorities).

**Remaining priority work (next up):**
1. Task 22: execution lifecycle persistence (order intents, fills).
2. Task 23: idempotency + duplicate suppression.
3. Task 24: restart-safe execution processing tests.
4. Task 25-26: flatten-by-close proof + PnL reconciliation model.
5. Task 27-28: AI decision logging + timeline endpoints.
6. Task 29-30: UI completion + gate automation.
