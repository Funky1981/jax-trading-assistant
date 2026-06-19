# Test Plan

## Objective

Validate the current production path end-to-end for:
- runtime health
- API contract availability
- backend regression stability
- frontend correctness
- replay/golden comparison utility behavior
- memory system (Postgres/pgvector retain, recall, reflect)

## Entry Criteria

- `docker compose up -d` completed successfully
- `jax-trader` and `jax-research` are healthy
- Postgres migrations are applied (including `000020_memory_items` and `000021_memory_items_constraints`)
- Postgres image is `pgvector/pgvector:pg16` (required for `memory_items`)
- frontend dependencies are installed (`frontend/node_modules`)

## Docker Image Requirements

The `postgres` service must use the pgvector image:

```yaml
# docker-compose.yml
postgres:
  image: pgvector/pgvector:pg16
```

Apply migrations:

```powershell
docker compose up db-migrate
```

## Automated Execution

### Quick Gate (recommended for every change)

```powershell
.\scripts\test-platform.ps1 -Mode quick
```

### Full Gate (pre-release)

```powershell
.\scripts\test-platform.ps1 -Mode full
```

### Full Gate + Visual E2E Report

```powershell
.\scripts\test-platform.ps1 -Mode full -OpenVisualReport
```

### Memory System Integration Tests

Requires Postgres running with pgvector. No OpenAI key required (uses `noopEmbedder`).

```powershell
$env:TEST_DATABASE_URL = "postgresql://jax:jax@localhost:5433/jax?sslmode=disable"
go test -tags integration -count=1 -v ./libs/pgmemory/...
```

Expected: 16 tests pass (12 integration + 4 unit).

### Memory Smoke Script (HTTP end-to-end)

Requires Docker and `jax-research` running.

Current implementation state: full Retain/Recall/Reflect supports `EMBEDDING_PROVIDER=local` with zero paid API usage, while `EMBEDDING_PROVIDER=openai` uses an OpenAI-compatible endpoint and requires `OPENAI_API_KEY`.

```powershell
# Without OpenAI key (schema + banks + list-items checks only):
.\scripts\smoke-memory.ps1 -SkipEmbed

# With OpenAI key (full retain/recall/reflect/search cycle):
$env:OPENAI_API_KEY = "<key>"
.\scripts\smoke-memory.ps1
```

## Coverage Matrix

1. Health checks:
   - `8081` trader API
   - `8091` research (`/health` and `/ready`)
   - `8092` ib-bridge
   - `8093` agent0-service
2. API smoke:
   - `/api/v1/signals`
   - `/api/v1/artifacts`
   - `/api/v1/testing/status`
   - `/api/v1/runs`
   - `/api/v1/ai-decisions`
3. Memory API:
   - `GET /v1/memory/banks` — returns `["research","trades","signals","reflections"]`
   - `POST /tools` (memory.retain) — stores item + embedding, returns `id`
   - `POST /tools` (memory.recall) — structured query by symbol/tags/date
   - `POST /tools` (memory.recall with `q`) — vector cosine nearest-neighbour
   - `POST /tools` (memory.reflect) — ephemeral synthesis item
   - `GET /v1/memory/banks/{bank}/items` — paginated list
   - `GET /v1/memory/banks/{bank}/items/{id}` — fetch by ID
   - `GET /v1/memory/search?q=...&bank=...` — search shortcut
4. Backend quality:
   - `scripts/go-verify.ps1` quick/full modes
   - `scripts/golden-check.ps1 -Mode verify`
5. Frontend quality:
   - `npm run lint`
   - `npm run typecheck`
   - `npm run test`
   - `npx playwright test --reporter=html` (full mode)

## Evidence Output

Each automated run writes:
- `Docs/runs/test_run_<timestamp>.md`
- `Docs/runs/test_run_<timestamp>.json`

Playwright full runs generate:
- `frontend/playwright-report/index.html`

## Manual Spot Checks (Release Candidate)

### Core Platform

1. Login/auth status path works (`/auth/status`, `/auth/login` if enabled).
2. Strategy/artifact list pages load with no frontend console errors.
3. Run detail and AI decisions pages load and show timeline data.
4. Artifact validation endpoint (`/api/v1/artifacts/{id}/validate`) returns trust-gate evidence.
5. Audit trail queries in `Docs/AUDIT_TRAIL.md` return expected rows.

### Memory Browser (MemoryBrowserPanel)

1. Open the dashboard and navigate to the Memory panel.
2. Confirm the bank selector shows: **research**, **trades**, **signals**, **reflections**.
3. Select **research** — panel loads without errors; empty state or item list shown.
4. Type a search query (e.g. "AAPL MACD") — results update in both embedding providers.
   - `EMBEDDING_PROVIDER=local` must work without `OPENAI_API_KEY`.
   - `EMBEDDING_PROVIDER=openai` requires `OPENAI_API_KEY`.
5. Open browser DevTools console — confirm no React errors or failed network requests.
6. Verify `GET /v1/memory/banks/research/items` returns `{"items": [...]}` envelope.

### Memory Retain/Recall Round-Trip

```powershell
# 1. Retain a test item
$body = @{
  tool  = "memory.retain"
  input = @{
    bank = "research"
    item = @{
      ts      = (Get-Date -Format "o")
      type    = "signal"
      symbol  = "AAPL"
      summary = "Manual test: strong MACD crossover signal"
      tags    = @("manual-test", "aapl")
      data    = @{ confidence = 0.9 }
      source  = @{ system = "manual" }
    }
  }
} | ConvertTo-Json -Depth 10
$r = Invoke-RestMethod "http://localhost:8091/tools" -Method Post -Body $body -ContentType "application/json"
$id = $r.output.id
Write-Host "Retained id=$id"

# 2. Recall by symbol (structured)
$q = @{ tool="memory.recall"; input=@{ bank="research"; query=@{ symbol="AAPL"; limit=5 } } } | ConvertTo-Json -Depth 10
(Invoke-RestMethod "http://localhost:8091/tools" -Method Post -Body $q -ContentType "application/json").output.items

# 3. Recall by text (vector)
$q = @{ tool="memory.recall"; input=@{ bank="research"; query=@{ q="MACD crossover AAPL"; limit=5 } } } | ConvertTo-Json -Depth 10
(Invoke-RestMethod "http://localhost:8091/tools" -Method Post -Body $q -ContentType "application/json").output.items

# 4. Fetch by ID
Invoke-RestMethod "http://localhost:8091/v1/memory/banks/research/items/$id"
```

Expected: each step returns data without 4xx/5xx errors. Vector recall result count > 0.

## Jax Memory Migration Release Readiness

### 1. Updated blocker list

1. Dual embedding providers must stay release-gated until both local and openai paths are proven in the deployed stack.
   - Current implementation: [`cmd/research/memory_proxy.go`](/c:/Projects/jax-trading-assistant/cmd/research/memory_proxy.go) now selects `EMBEDDING_PROVIDER=local|openai`; [`libs/pgmemory/embedding.go`](/c:/Projects/jax-trading-assistant/libs/pgmemory/embedding.go) validates provider-specific config and keeps Postgres + pgvector as the storage path.
   - Remaining release impact: both providers still need full stack verification and promotion evidence before the blocker can be closed.
2. `agent0-service` memory failures must stay fail-fast in verification.
   - Current implementation: [`services/agent0-service/agent.py`](/c:/Projects/jax-trading-assistant/services/agent0-service/agent.py) now raises on memory recall failures instead of silently converting them to `[]`, but the deployed stack still needs verification that those failures surface cleanly.
3. Silent default bank behavior has been removed, but explicit bank wiring remains release-critical.
   - Current implementation: [`services/agent0-service/config.py`](/c:/Projects/jax-trading-assistant/services/agent0-service/config.py) no longer defaults `memory_bank`; release verification still needs to prove every caller and environment sets the intended bank explicitly.
4. Runtime/deploy contract must be re-verified end to end.
   - Current implementation updates [`docker-compose.yml`](/c:/Projects/jax-trading-assistant/docker-compose.yml), [`.env.example`](/c:/Projects/jax-trading-assistant/.env.example), [`start.ps1`](/c:/Projects/jax-trading-assistant/start.ps1), [`Docs/OPERATIONS/DEBUGGING.md`](/c:/Projects/jax-trading-assistant/Docs/OPERATIONS/DEBUGGING.md), and [`scripts/smoke-memory.ps1`](/c:/Projects/jax-trading-assistant/scripts/smoke-memory.ps1) for local-by-default embeddings, but release still depends on stack verification with those settings.
5. DB hardening migration is not proven applied in every target environment.
   - Release still requires [`000020_memory_items.up.sql`](/c:/Projects/jax-trading-assistant/db/postgres/migrations/000020_memory_items.up.sql) and [`000021_memory_items_constraints.up.sql`](/c:/Projects/jax-trading-assistant/db/postgres/migrations/000021_memory_items_constraints.up.sql) to be present and reflected in the live schema before promotion.
6. Readiness still needs full release-grade verification against actual memory startup requirements.
   - Current implementation: [`cmd/research/main.go`](/c:/Projects/jax-trading-assistant/cmd/research/main.go) now exposes the selected `embedding_provider` on `/health`, but release still needs end-to-end proof that health/readiness reflects the deployed memory path correctly.
7. Trader runtime health remains a release blocker.
   - Carry forward unchanged: release stays blocked until `jax-trader` is healthy under the deployed stack and remains healthy through smoke and soak checks.

### 2. Embedding-mode target design

- Add explicit `EMBEDDING_PROVIDER=local|openai`.
- `local` provider is the required default for development/testing and must not require `OPENAI_API_KEY`.
- `dev` and `test` runtime modes must reject `EMBEDDING_PROVIDER=openai` at startup.
- `openai` provider is allowed for production and optional remote testing and must require `OPENAI_API_KEY` plus the OpenAI-compatible endpoint settings it uses.
- Implement a local embedder in [`libs/pgmemory`](/c:/Projects/jax-trading-assistant/libs/pgmemory) that runs in-process, uses zero paid API calls, and emits deterministic `1536`-dimension vectors so it stays compatible with `memory_items.embedding vector(1536)`.
- Keep the existing OpenAI-compatible HTTP embedder for `openai` provider, with `OPENAI_BASE_URL` defaulting only inside that provider.
- Resolve the embedding provider before the HTTP server starts, validate only the config required by that provider, and fail fast before binding the port if selection or required settings are invalid.
- Log the selected embedding provider, model/provider identity, and vector dimension at startup.
- Expose mode-aligned memory readiness so `/health` and any readiness gate report the same truth as startup validation.
- Remove or explicitly reject silent bank defaults on release-critical paths so bank selection errors fail loudly.

### 3. Ordered implementation plan

1. Refactor [`libs/pgmemory/embedding.go`](/c:/Projects/jax-trading-assistant/libs/pgmemory/embedding.go) into a mode-aware embedding config.
   - Add a shared config type with `Provider`, `BaseURL`, `APIKey`, and `Model` fields.
   - Add a local embedder implementation that keeps the current `1536`-dimension storage contract.
   - Keep the OpenAI-compatible embedder as the `openai` provider.
2. Update [`cmd/research/memory_proxy.go`](/c:/Projects/jax-trading-assistant/cmd/research/memory_proxy.go) and startup wiring in [`cmd/research/main.go`](/c:/Projects/jax-trading-assistant/cmd/research/main.go).
   - Resolve `EMBEDDING_PROVIDER` first.
   - `local` provider: no API key required; fail only if local embedder config is invalid.
   - `openai` provider: require `OPENAI_API_KEY`; validate `OPENAI_BASE_URL` if set; keep `EMBEDDING_MODEL` optional with a default.
   - Add readiness/health reporting for selected mode and memory-store readiness.
3. Close the silent-failure and silent-default paths around memory consumers.
   - Update [`services/agent0-service/agent.py`](/c:/Projects/jax-trading-assistant/services/agent0-service/agent.py) so release smoke does not pass when memory is unavailable but silently replaced with an empty result.
   - Require explicit bank wiring in release-critical callers instead of relying on implicit defaults from [`services/agent0-service/config.py`](/c:/Projects/jax-trading-assistant/services/agent0-service/config.py) or hard-coded orchestration defaults.
4. Update docker/dev stack configuration.
   - Set `EMBEDDING_PROVIDER=local` in local compose and bootstrap flows.
   - Keep `OPENAI_API_KEY`, `OPENAI_BASE_URL`, and `EMBEDDING_MODEL` as optional dev overrides and required only for `EMBEDDING_PROVIDER=openai`.
   - Update [`.env.example`](/c:/Projects/jax-trading-assistant/.env.example), [`docker-compose.yml`](/c:/Projects/jax-trading-assistant/docker-compose.yml), and [`start.ps1`](/c:/Projects/jax-trading-assistant/start.ps1) together so runtime and deploy defaults match.
5. Add the missing tests.
   - Unit tests in [`libs/pgmemory/embedding_test.go`](/c:/Projects/jax-trading-assistant/libs/pgmemory/embedding_test.go) for `local` without key, `openai` without key fails, invalid openai base URL fails, and unknown provider fails.
   - Startup tests in [`cmd/research/main_test.go`](/c:/Projects/jax-trading-assistant/cmd/research/main_test.go) for mode-specific fail-fast behavior.
   - HTTP/handler tests in [`cmd/research/memory_proxy_test.go`](/c:/Projects/jax-trading-assistant/cmd/research/memory_proxy_test.go) that prove memory endpoints work in local mode without `OPENAI_API_KEY`.
   - Integration/smoke coverage so [`scripts/smoke-memory.ps1`](/c:/Projects/jax-trading-assistant/scripts/smoke-memory.ps1) passes in local mode and separately exercises remote mode when credentials are present.
   - Health/readiness tests proving invalid embedding config cannot report healthy.
6. Update release docs and operator runbooks.
   - Refresh [`Docs/TESTING/TEST_PLAN.md`](/c:/Projects/jax-trading-assistant/Docs/TESTING/TEST_PLAN.md), [`Docs/OPERATIONS/DEBUGGING.md`](/c:/Projects/jax-trading-assistant/Docs/OPERATIONS/DEBUGGING.md), and release checklists so they describe both modes and the exact startup validation rules.
7. Re-run the release gate.
   - Re-verify memory migrations, research/trader/agent0 health, local-mode smoke, remote-mode smoke, and the trader soak checks before removing the blockers.

### 4. Acceptance criteria

- `EMBEDDING_PROVIDER=local` starts `jax-research` without `OPENAI_API_KEY` and passes retain, structured recall, vector recall, reflect, and `/v1/memory/search` end to end.
- `EMBEDDING_PROVIDER=openai` fails fast before port bind when `OPENAI_API_KEY` is missing.
- `EMBEDDING_PROVIDER=openai` starts successfully with valid `OPENAI_API_KEY` and OpenAI-compatible endpoint settings and passes the same memory smoke flow.
- Startup logs, `/health`, and readiness checks agree on the selected embedding provider and memory readiness state.
- `agent0-service` no longer masks broken memory connectivity as an empty-success path during release smoke.
- Release-critical callers do not rely on implicit bank defaults; wrong or missing bank selection returns an explicit error.
- `schema_migrations` and live DB inspection prove both memory migrations are applied and constraints/indexes exist.
- `jax-trader`, `jax-research`, and `agent0-service` remain healthy under `docker compose up -d` and through the pre-production soak window.

### 5. Final go/no-go after these fixes

- Current status: no-go.
- Go only after all blockers above are closed and both embedding modes satisfy the acceptance criteria. Until then, the Jax memory migration is not release-ready.

### DB Verification

```sql
-- Connect: psql postgresql://jax:jax@localhost:5433/jax
SELECT count(*) FROM memory_items;
SELECT id, bank, type, symbol, LEFT(summary, 60) FROM memory_items ORDER BY created_at DESC LIMIT 10;
-- Confirm embedding is populated (not null):
SELECT id, embedding IS NOT NULL AS has_embedding FROM memory_items LIMIT 5;
```

## Staging Soak (Pre-Production)

Run a continuous paper-mode soak for at least one market session:

1. `JAX_RUNTIME_MODE=paper`
2. `JAX_REQUIRE_EXPLICIT_RUNTIME_MODE=true`
3. Run recurring health checks and capture order/error metrics
4. Verify no provenance integrity regressions during the soak window
5. Confirm memory items are being retained during live signal generation (check `memory_items` table row count increases)

Use `Docs/PRODUCTION_READINESS.md` as the promotion checklist after soak completion.
