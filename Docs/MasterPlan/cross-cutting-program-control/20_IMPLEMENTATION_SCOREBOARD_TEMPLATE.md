# Implementation Scoreboard Template (Add to Master Plan Execution)

Use this file to track the master plan execution against the `work` branch.
This is designed to prevent hidden gaps and to keep **everything inside the plan**.

## Status values (use exactly these)
- `MISSING` — not started / not present
- `PARTIAL` — exists but incomplete or not integrated
- `DONE` — implemented and integrated
- `BLOCKED` — cannot proceed (dependency/risk)
- `DEFERRED` — intentionally pushed to later phase (must include reason)
- `N/A` — not applicable (must include reason)

## Priority values
- `P0` = must complete before any paper trading
- `P1` = required for robust paper-trading operations
- `P2` = pre-live / shadow-readiness
- `P3` = later optimization / extension

## Evidence rule
Every `DONE` item must include evidence:
- file path(s)
- migration name(s)
- API route(s)
- test file(s)
- gate/proof artifact (if applicable)

## Blocking rule
Any item marked `BLOCKED` or any failed P0 gate blocks progression to paper trading.

---

## 1) Master Program Scoreboard (high level)

| ID | Workstream | Capability | Priority | Status | Phase | Owner | Depends On | Evidence | Gate | Notes |
|---|---|---|---|---|---|---|---|---|---|---|
| A1 | Safety | Runtime mode policy (dev/test/research/paper/live) | P0 | MISSING | 1 |  |  |  | Gate 0 / Gate 9 |  |
| A2 | Safety | No-fake-data enforcement in runtime paths | P0 | PARTIAL | 1 |  | A1 | `libs/utcp/backtest_local_tools.go` still exists | Gate 9 | Must block fabricated runs outside dev/test |
| A3 | Safety | Provenance fields on runs/artifacts | P0 | MISSING | 1 |  | A1 |  | Gate 9 |  |
| A4 | Safety | Startup provider validation | P0 | MISSING | 1 |  | A1 |  | Gate 9 |  |
| B1 | Event Data | Event ingestion + normalization | P0 | MISSING | 2 |  | A1-A4 |  | Gate 1 |  |
| B2 | Event Data | Symbol mapping + confidence/ambiguity handling | P0 | MISSING | 2 |  | B1 |  | Gate 1 |  |
| C1 | Market Data | Intraday candles/volume/VWAP/session correctness | P0 | PARTIAL | 3 |  | B1 |  | Gate 1 | Verify real data path and snapshots |
| D1 | Strategy | Strategy types metadata + registry | P0 | MISSING | 4 |  | C1 |  | Gate 0 |  |
| D2 | Strategy | Strategy instance CRUD + validation + UI forms | P0 | PARTIAL | 4 |  | D1 | `config/strategy-instances/*.json` examples exist | Gate 0 / Gate 3 | Needs DB/API/UI integration |
| E1 | IBM Pack | `news_shock_momentum_v1` | P0 | MISSING | 5 |  | D1,C1,B1 |  | Gate 2 |  |
| E2 | IBM Pack | `opening_range_to_close_v1` | P0 | MISSING | 5 |  | D1,C1 |  | Gate 2 |  |
| F1 | Research | Real deterministic backtest truth path | P0 | PARTIAL | 6 |  | C1,D1 | `internal/modules/backtest/engine.go` exists | Gate 2 | Must eliminate fake UTCP path from truth path |
| F2 | Research | Sweeps + walk-forward + reproducibility | P1 | MISSING | 6 |  | F1 |  | Gate 2 |  |
| G1 | Execution | Signal->intent->broker->fills chain | P0 | PARTIAL | 7 |  | D2 | `internal/modules/execution/engine.go` exists | Gate 4 | Need persistence + reconciliation proof |
| G2 | Execution | Flatten-by-close + proof | P0 | MISSING | 7 |  | G1 |  | Gate 7 |  |
| G3 | Execution | P/L reconciliation | P0 | MISSING | 7 |  | G1 |  | Gate 5 |  |
| H1 | AI Audit | `ai_decisions` + acceptance + replay | P1 | MISSING | 8 |  | B1,F1 |  | Gate 8 | Required if AI used |
| I1 | UI Ops | `/research` route + page integration | P0 | PARTIAL | 9 |  | F1,D2 | `frontend/src/app/App.tsx` changed on work | N/A | Verify route wiring exists |
| I2 | UI Ops | `/analysis` route + decision timeline | P0 | MISSING | 9 |  | F1,G1,H1 |  | N/A |  |
| I3 | UI Ops | `/testing` route + gates dashboard | P0 | MISSING | 9 |  | Gate jobs |  | N/A |  |
| J1 | Trust Gates | Gate automation (0–10) + proofs | P0 | MISSING | 10 |  | A1-I3 |  | All |  |
| J2 | Shadow | Shadow/parity validator integration + UI | P2 | PARTIAL | 10 |  | J1 | `cmd/shadow-validator` exists | Gate 10 |  |

---

## 2) File-by-file Gap Matrix (fill this against `work` branch)

> Use one row per file/module that matters. Add rows freely.  
> This is where you stop assumptions and force evidence.

| Area | File/Path | Expected Role | Exists? | Status | Used by Runtime? | Needs Tests? | Gate Impact | Owner | Notes |
|---|---|---|---|---|---|---|---|---|---|
| Safety | `libs/utcp/backtest_local_tools.go` | Dev/test helper only (must not affect truth path) | Yes | PARTIAL | Unknown | Yes | Gate 9 / Gate 2 |  | Confirm build/runtime gating |
| Research | `internal/modules/backtest/engine.go` | Deterministic real backtest engine wrapper | Yes | PARTIAL | Yes | Yes | Gate 2 |  | Verify data source provenance and persistence path |
| Execution | `internal/modules/execution/engine.go` | Execution chain core | Yes | PARTIAL | Yes | Yes | Gate 4/5/7 |  | Need intent/fill persistence + flatten proof integration |
| Trader API | `cmd/trader/frontend_api.go` | Backend API for UI ops pages | Yes | PARTIAL | Yes | Yes | Multiple |  | Map implemented endpoints to master API plan |
| Trader API | `cmd/trader/handlers_artifacts.go` | Artifact validation/promotion APIs | Yes | PARTIAL | Yes | Yes | Gate 3 |  | Add provenance enforcement checks |
| Frontend | `frontend/src/app/App.tsx` | Route wiring (`/research`,`/analysis`,`/testing`) | Yes | PARTIAL | Yes | Yes | N/A |  | Confirm all new pages wired and guarded |
| DB | `db/postgres/migrations/000008_codex_strategy_lab.up.sql` | Strategy lab schema foundation | Yes | PARTIAL | Yes | Yes | Gate 0/3 |  | Compare vs master schema requirements |
| Config | `config/providers.json` | Provider endpoints/modes policy | Yes | PARTIAL | Yes | Yes | Gate 1/9 |  | Add provider type + synthetic flags if needed |
| Config | `config/strategy-instances/*.json` | Example instance definitions | Yes | PARTIAL | Possibly | Yes | Gate 0/3 |  | Move to DB/API-driven instances while keeping import/export |
| Runtime | `cmd/research/main.go` | Research runtime orchestration | Yes | PARTIAL | Yes | Yes | Gate 2 |  | Verify no fake path reachable |
| Runtime | `cmd/shadow-validator/main.go` | Shadow/parity validation | Yes | PARTIAL | Yes | Yes | Gate 10 |  | Integrate gate reporting and UI artifacts |

---

## 3) API Endpoint Scoreboard

Track each endpoint in the master plan.

| Endpoint | Purpose | Priority | Implemented | Tested | UI wired | Audited | Provenance surfaced | Notes |
|---|---|---|---|---|---|---|---|---|
| `GET /api/v1/system/runtime` | Runtime mode/build state | P0 | No | No | No | N/A | N/A |  |
| `GET /api/v1/system/providers` | Provider map + fake/synthetic policy visibility | P0 | No | No | No | N/A | Yes |  |
| `GET /api/v1/events` | Event browsing | P0 | No | No | No | Yes | Yes |  |
| `GET /api/v1/strategy-types` | Strategy metadata | P0 | No | No | No | Yes | N/A |  |
| `GET/POST /api/v1/strategy-instances` | Instance management | P0 | Partial | No | Partial | Yes | N/A |  |
| `POST /api/v1/research/runs` | Launch backtest/research run | P0 | Partial | No | Partial | Yes | Yes |  |
| `GET /api/v1/research/runs/{id}/timeline` | Decision trace timeline | P0 | No | No | No | Yes | Yes |  |
| `POST /api/v1/flatten/run` | Flatten same-day positions | P0 | No | No | No | Yes | N/A |  |
| `GET /api/v1/gates/history` | Trust gate history/proofs | P0 | No | No | No | Yes | N/A |  |
| `GET /api/v1/ai-decisions` | AI audit listing | P1 | No | No | No | Yes | N/A |  |

---

## 4) Trust Gate Scoreboard

| Gate | Name | Required Before Paper? | Automated? | UI Visible? | Last Result | Evidence Artifact | Blocking Issues |
|---|---|---|---|---|---|---|---|
| 0 | Config/Schema Integrity | Yes | No | No | Unknown |  |  |
| 1 | Data Reconciliation | Yes | No | No | Unknown |  |  |
| 2 | Deterministic Replay | Yes | No | No | Unknown |  |  |
| 3 | Artifact Promotion Controls | Yes | Partial | Partial | Unknown |  |  |
| 4 | Execution Path Integration | Yes | No | No | Unknown |  |  |
| 5 | P/L Reconciliation | Yes | No | No | Unknown |  |  |
| 6 | Failure Injection | Yes | No | No | Unknown |  |  |
| 7 | Flatten-by-Close Proof | Yes | No | No | Unknown |  |  |
| 8 | AI Audit Completeness | If AI used | No | No | Unknown |  |  |
| 9 | Data Provenance Integrity | Yes | No | No | Unknown |  |  |
| 10 | Shadow/Parity Validation | Pre-live | Partial | No | Unknown |  |  |

---

## 5) Weekly Program Review Template

### This week (completed)
- 

### New blockers
- 

### P0 risk changes
- 

### Gate movement
- Gates moved to green:
- Gates regressed:

### Next week plan (top 10)
1. 
2. 
3. 
4. 
5. 
6. 
7. 
8. 
9. 
10. 

---

## 6) Definition of "Nothing Left Outside the Plan"
A feature/change is **not allowed** to be considered complete unless it appears in:
1. the master capability matrix
2. the phased program plan
3. this scoreboard (or a linked child scoreboard)
4. an acceptance checklist item
5. a gate/evidence path if it affects trust/trading/research


## RAG note
If RAG is implemented, it must be tracked as **research-only** work and must have an import boundary preventing `cmd/trader` from using it.
