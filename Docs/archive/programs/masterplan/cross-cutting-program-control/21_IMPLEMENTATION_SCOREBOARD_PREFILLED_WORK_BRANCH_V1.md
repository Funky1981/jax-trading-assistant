# Implementation Scoreboard (Pre-filled v1 for `work` branch)

This is a **starting baseline** populated from the repo checks we completed against `work` (including `main...work` compare showing **49 commits ahead**).
It is intentionally conservative:
- If not directly verified, status is `PARTIAL` or `MISSING`
- Do not upgrade a row to `DONE` without file/API/test evidence

## Snapshot basis
- Branch compared: `main...work` (ahead by 49 commits)
- Verified key files include:
  - `cmd/trader/frontend_api.go`
  - `cmd/trader/handlers_artifacts.go`
  - `cmd/research/main.go`
  - `cmd/shadow-validator/main.go`
  - `internal/modules/backtest/engine.go`
  - `internal/modules/execution/engine.go`
  - `frontend/src/app/App.tsx`
  - `libs/utcp/backtest_local_tools.go`
  - `config/providers.json`
  - `config/strategy-instances/*.json`
  - DB migrations `000006..000008`

## Hard reality (still true)
- `libs/utcp/backtest_local_tools.go` fake/fabricated local backtest path exists → **No-Fake-Data Phase remains P0**
- `work` has a strong backend foundation, but robust IBM-style event trading is **not yet complete/integrated**

---

## 1) Master Program Scoreboard (Pre-filled)

| ID | Workstream | Capability | Priority | Status | Phase | Evidence | Notes |
|---|---|---|---|---|---|---|---|
| A1 | Safety | Runtime mode policy (`dev/test/research/paper/live`) | P0 | MISSING | 1 | Not verified in checked files | Add startup-enforced mode policy |
| A2 | Safety | No-fake-data enforcement in runtime paths | P0 | PARTIAL | 1 | `libs/utcp/backtest_local_tools.go` still present | Must be gated/removed from truth paths |
| A3 | Safety | Data provenance on runs/artifacts | P0 | MISSING | 1 | Not verified | Add DB/API/UI provenance fields |
| A4 | Safety | Startup provider validation (block fake providers in paper/live) | P0 | MISSING | 1 | Not verified | Add boot-time policy checks |
| A5 | Safety | Provenance trust gate (Data Provenance Integrity) | P0 | MISSING | 1 | Not verified | New gate required |

| B1 | Event Data | Event ingestion adapters | P0 | PARTIAL | 2 | `cmd/trader/market_ingester.go` exists; archived ingest services moved to `archive/` | Need current event pipeline for IBM news flow |
| B2 | Event Data | Event normalization + dedupe + timestamps | P0 | MISSING | 2 | Not verified in active runtimes | Archived code exists but not current runtime evidence |
| B3 | Event Data | Symbol mapping + ambiguity handling | P0 | MISSING | 2 | Not verified | Needed for ticker correctness |
| B4 | Event Data | Event APIs + timeline | P0 | MISSING | 2 | Not verified in `frontend_api.go` from our checks | Add `/api/v1/events*` |

| C1 | Market Data | Intraday candles/volume/VWAP/session correctness | P0 | PARTIAL | 3 | `libs/calendar/*`, `cmd/trader/market_ingester.go`, provider config present | Need verified real intraday truth path + recon |
| C2 | Market Data | Dataset snapshots + provenance linkage | P0 | MISSING | 3 | `libs/dataset/registry.go` exists but empty/placeholder in compare listing | Implement storage + API + run linkage |
| C3 | Market Data | Data reconciliation gate | P0 | MISSING | 3 | Not verified | Gate 1 required |

| D1 | Strategy | Strategy type metadata registry | P0 | MISSING | 4 | Not verified | Add `/api/v1/strategy-types` |
| D2 | Strategy | Strategy instance config support | P0 | PARTIAL | 4 | `config/strategy-instances/*.json`, loader in `cmd/trader/strategy_instances_loader.go` | Needs DB/API/UI CRUD & validation |
| D3 | Strategy | UI-configurable instances (no code changes) | P0 | MISSING | 4 | Not verified | Core requirement |
| D4 | Strategy | Artifact promotion controls for strategy evidence | P0 | PARTIAL | 4 | `handlers_artifacts.go`, migrations `000006_*` | Add provenance + gate enforcement |

| E1 | IBM Pack | `news_shock_momentum_v1` | P0 | MISSING | 5 | Not verified | Primary IBM strategy |
| E2 | IBM Pack | `opening_range_to_close_v1` | P0 | MISSING | 5 | Not verified | Baseline same-day strategy |
| E3 | IBM Pack | `event_gap_continuation_v1` | P1 | MISSING | 5 | Not verified | Extreme event continuation |
| E4 | IBM Pack | `panic_reversion_v1` | P1 | MISSING | 5 | Not verified | Riskier |
| E5 | IBM Pack | `pairs_event_relative_v1` (research-only) | P2 | MISSING | 5 | Not verified | Advanced later |

| F1 | Research | Deterministic backtest engine (real) | P0 | PARTIAL | 6 | `internal/modules/backtest/engine.go` + tests exist | Strong base but fake UTCP path still exists |
| F2 | Research | Research runtime orchestration | P0 | PARTIAL | 6 | `cmd/research/main.go`, `cmd/research/backtest.go` | Verify persistence/provenance path |
| F3 | Research | Run persistence (metrics/trades/events) | P0 | PARTIAL | 6 | `000008_codex_strategy_lab.up.sql` exists | Need endpoint/UI linkage confirmation |
| F4 | Research | Parameter sweeps | P1 | MISSING | 6 | Not verified | Planned capability |
| F5 | Research | Walk-forward validation | P1 | MISSING | 6 | Not verified | Planned capability |
| F6 | Research | Reproducibility gate (same config/dataset/seed) | P0 | MISSING | 6 | Not verified | Gate 2 required |

| G1 | Execution | Execution engine core | P0 | PARTIAL | 7 | `internal/modules/execution/engine.go` + tests exist | Strong base |
| G2 | Execution | Signal->approval->intent->broker->fills persistence chain | P0 | PARTIAL | 7 | `cmd/trader/frontend_api.go` + execution module | Need DB schema/API proof end-to-end |
| G3 | Execution | Duplicate prevention + restart safety | P0 | MISSING | 7 | Not verified | Required for trust |
| G4 | Execution | Flatten-by-close + proof | P0 | MISSING | 7 | Not verified | Gate 7 required |
| G5 | Execution | P/L reconciliation | P0 | MISSING | 7 | Not verified | Gate 5 required |

| H1 | AI Audit | AI wrappers with schema outputs | P1 | PARTIAL | 8 | `libs/agent0/client.go` modified; `useAISuggestion` UI hook exists | Not enough for audit completeness |
| H2 | AI Audit | `ai_decisions` table + acceptance table | P1 | MISSING | 8 | Not verified in migrations up to 000008 | Add schema + logging |
| H3 | AI Audit | Prompt/version capture + replay | P1 | MISSING | 8 | Not verified | Needed for trust |
| H4 | AI Audit | AI decision inspector API/UI | P1 | MISSING | 8 | Not verified | Add analysis trace integration |

| I1 | UI Ops | Frontend app shell/navigation updates | P0 | PARTIAL | 9 | `AppShell.tsx` modified, `App.tsx` modified | Progress visible |
| I2 | UI Ops | `/research` route wired | P0 | PARTIAL | 9 | `App.tsx` modified on work | We did not verify route present by exact path |
| I3 | UI Ops | `/analysis` route wired | P0 | MISSING | 9 | Not verified | Must confirm |
| I4 | UI Ops | `/testing` route wired | P0 | MISSING | 9 | Not verified | Must confirm |
| I5 | UI Ops | Research page integrated to real APIs | P0 | MISSING | 9 | Not verified | No assumptions |
| I6 | UI Ops | Analysis page decision timeline | P0 | MISSING | 9 | Not verified | Core trust UX |
| I7 | UI Ops | Testing page gates dashboard/proofs | P0 | MISSING | 9 | Not verified | Core trust UX |

| J1 | Gates | Artifact validation/promotion gate workflow | P0 | PARTIAL | 10 | `handlers_artifacts.go`, artifact domain/store, migrations | Extend to provenance + evidence rules |
| J2 | Gates | Full trust gate automation (0–10) | P0 | MISSING | 10 | Not verified | Major deliverable |
| J3 | Shadow | Shadow validator runtime | P2 | PARTIAL | 10 | `cmd/shadow-validator/main.go`, `docker-compose.shadow.yml` | Integrate with gate reporting/UI |
| J4 | Incidents | Incident workflow and blocking policy | P1 | MISSING | 10 | Not verified | Needed for disciplined paper trial |

---

## 2) File-by-file Gap Matrix (Pre-filled sample)

> Expand this aggressively as you implement. This is only the starting baseline from what we have verified.

| Area | File/Path | Expected Role | Exists | Status | Evidence | Immediate Action |
|---|---|---|---|---|---|---|
| Safety | `libs/utcp/backtest_local_tools.go` | **Dev/test only** helper; must not influence truth path | Yes | PARTIAL/RISK | Fake `generateRun(...)` path previously verified | Gate behind build/runtime policy or remove from production paths |
| Research | `internal/modules/backtest/engine.go` | Real deterministic backtest engine wrapper | Yes | PARTIAL | File added on `work` with tests | Wire to real data + persistence + provenance |
| Research | `cmd/research/main.go` | Research runtime orchestration | Yes | PARTIAL | Added on `work` | Verify no fake data reachability |
| Research | `cmd/research/backtest.go` | Research backtest execution path | Yes | PARTIAL | Added on `work` | Map to run storage + gate outputs |
| Execution | `internal/modules/execution/engine.go` | Execution logic core | Yes | PARTIAL | File + tests added on `work` | Add persistence/recon/flatten integration |
| Trader API | `cmd/trader/frontend_api.go` | Primary API surface for UI ops | Yes | PARTIAL | Large file added on `work` | Create endpoint matrix vs master plan |
| Trader API | `cmd/trader/handlers_artifacts.go` | Artifact APIs | Yes | PARTIAL | Added on `work` | Add provenance and evidence gate checks |
| Trader | `cmd/trader/strategy_instances_loader.go` | Load instance configs | Yes | PARTIAL | Added on `work` | Transition to DB/API CRUD while keeping import/export |
| Trader | `cmd/trader/market_ingester.go` | Market ingestion path | Yes | PARTIAL | Added on `work` | Clarify event vs market data responsibilities |
| Frontend | `frontend/src/app/App.tsx` | Route wiring (`/research` `/analysis` `/testing`) | Yes | PARTIAL | Modified on `work` | Verify exact routes and auth guards |
| Frontend | `frontend/src/components/layout/AppShell.tsx` | Navigation shell | Yes | PARTIAL | Modified on `work` | Add links for new operations pages |
| Frontend | `frontend/src/pages/LoginPage.tsx` | Auth/login page | Yes | DONE (for scope) | Added on `work` | N/A |
| Config | `config/providers.json` | Provider registry/endpoints | Yes | PARTIAL | Modified on `work` | Add provider type/mode/synthetic policy metadata |
| Config | `config/strategy-instances/*.json` | Example paper instance configs | Yes | PARTIAL | `earnings-aapl-paper-v1`, `or-spy-paper-v1` | Keep as import/export fixtures |
| DB | `db/postgres/migrations/000006_strategy_artifacts.up.sql` | Artifact schema | Yes | PARTIAL | Added on `work` | Extend for provenance evidence links |
| DB | `db/postgres/migrations/000007_ejlayer.up.sql` | EJ layer schema | Yes | PARTIAL | Added on `work` | Map to decision trace usage |
| DB | `db/postgres/migrations/000008_codex_strategy_lab.up.sql` | Strategy lab/research foundation | Yes | PARTIAL | Added on `work` | Gap-check against full master schema |
| Runtime | `cmd/shadow-validator/main.go` | Shadow/parity runtime | Yes | PARTIAL | Added on `work` | Connect to Gate 10 + proof artifacts |
| Ops | `docker-compose.shadow.yml` | Shadow environment orchestration | Yes | PARTIAL | Added on `work` | Add runbooks + CI checks |
| Docs | `Docs/PHASE_4_EXECUTION_COMPLETE.md` | Claimed phase completion docs | Yes | PARTIAL (doc only) | Added on `work` | Validate against actual gate evidence before trusting |

---

## 3) API Endpoint Scoreboard (Pre-filled starter)

| Endpoint | Priority | Status | Notes |
|---|---|---|---|
| `GET /api/v1/artifacts` + artifact mutation endpoints | P0 | PARTIAL | Artifact handlers exist; verify exact routes/tests and add provenance checks |
| `GET /api/v1/strategy-types` | P0 | MISSING | Need coded metadata registry endpoint |
| `GET/POST /api/v1/strategy-instances` | P0 | PARTIAL | Config loader exists; DB/API/UI CRUD not verified |
| `POST /api/v1/research/runs` | P0 | PARTIAL | Research runtime exists; endpoint/path and persistence need verification |
| `GET /api/v1/research/runs/{id}/timeline` | P0 | MISSING | Needed for decision trace |
| `GET /api/v1/events` | P0 | MISSING | Event browsing not verified |
| `POST /api/v1/flatten/run` | P0 | MISSING | Flatten control/proof endpoint not verified |
| `GET /api/v1/gates/history` | P0 | MISSING | Gate dashboard backend missing/unverified |
| `GET /api/v1/ai-decisions` | P1 | MISSING | AI audit backend missing/unverified |
| `GET /api/v1/system/providers` | P0 | MISSING | Needed for no-fake-data visibility and startup policy evidence |

---

## 4) Trust Gate Scoreboard (Pre-filled starter)

| Gate | Name | Required Before Paper | Status | Notes |
|---|---|---:|---|---|
| 0 | Config/Schema Integrity | Yes | PARTIAL | Artifact validation exists; full config/schema gate automation not verified |
| 1 | Data Reconciliation | Yes | MISSING | Must validate intraday/event data quality |
| 2 | Deterministic Replay | Yes | MISSING | Backtest engine exists, but reproducibility gate not verified |
| 3 | Artifact Promotion Controls | Yes | PARTIAL | Artifact workflow exists; provenance + evidence policy still missing |
| 4 | Execution Path Integration | Yes | MISSING | Execution engine exists, gate/proofs not verified |
| 5 | P/L Reconciliation | Yes | MISSING | Required for trust |
| 6 | Failure Injection | Yes | MISSING | Required before serious paper trial |
| 7 | Flatten-by-Close Proof | Yes | MISSING | Same-day strategy blocker |
| 8 | AI Audit Completeness | If AI used | MISSING | No ai_decisions acceptance/replay evidence yet |
| 9 | Data Provenance Integrity | Yes | MISSING | New no-fake-data gate; highest priority |
| 10 | Shadow/Parity Validation | Pre-live | PARTIAL | Shadow runtime exists; gate integration/proofs missing |

---

## 5) Recommended next update cycle (P0 only)
1. Complete **No-Fake-Data Phase**:
   - runtime mode policy
   - provenance schema/API/UI
   - startup provider validation
   - fake UTCP path blocked in paper/research truth paths
   - Gate 9 implemented
2. Verify `App.tsx` actual routes and add route evidence to scoreboard
3. Build endpoint matrix from `cmd/trader/frontend_api.go` and mark implemented vs missing
4. Implement event data foundation (`events`, normalization, symbol mapping)
5. Implement strategy types metadata + instance DB/API CRUD
6. Connect research runtime to persisted, provenance-verified runs only
7. Add flatten-by-close + proof + Gate 7 before any IBM same-day paper testing

---

## 6) Rule for using this file
Do not treat this as “progress proof.”
Treat it as a **truth ledger**:
- downgrade uncertain items
- attach evidence links
- only mark `DONE` with files + tests + gate proof
